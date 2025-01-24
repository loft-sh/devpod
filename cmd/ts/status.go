// Inspired by: https://github.com/tailscale/tailscale/blob/v1.78.1/cmd/tailscale/cli/status.go
package ts

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	open2 "github.com/loft-sh/devpod/pkg/open"
	"github.com/loft-sh/devpod/pkg/tailscale"
	"github.com/spf13/cobra"
	"golang.org/x/net/idna"
	ts "tailscale.com/client/tailscale"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/net/netmon"
	"tailscale.com/util/dnsname"
)

type StatusCmd struct {
	*TsNetFlags

	json    bool   // JSON output mode
	web     bool   // run webserver
	listen  string // in web mode, webserver address to listen on, empty means auto
	browser bool   // in web mode, whether to open browser
	active  bool   // in CLI mode, filter output to only peers with active sessions
	self    bool   // in CLI mode, show status of local machine
	peers   bool   // in CLI mode, show status of peer machines
}

func NewStatusCmd() *cobra.Command {
	cmd := &StatusCmd{TsNetFlags: &TsNetFlags{}}

	var statusCmd = &cobra.Command{
		Use:     "status",
		Example: "tailscale status [--active] [--web] [--json]",
		Short:   "Show state of tailscaled and its connections",
		Args:    cobra.NoArgs,
		RunE:    cmd.Run,
	}

	statusCmd.Flags().BoolVar(&cmd.json, "json", false, "output in JSON format (WARNING: format subject to change)")
	statusCmd.Flags().BoolVar(&cmd.web, "web", false, "run webserver with HTML showing status")
	statusCmd.Flags().BoolVar(&cmd.active, "active", false, "filter output to only peers with active sessions (not applicable to web mode)")
	statusCmd.Flags().BoolVar(&cmd.self, "self", true, "show status of local machine")
	statusCmd.Flags().BoolVar(&cmd.peers, "peers", true, "show status of peers")
	statusCmd.Flags().StringVar(&cmd.listen, "listen", "127.0.0.1:8384", "listen address for web mode; use port 0 for automatic")
	statusCmd.Flags().BoolVar(&cmd.browser, "browser", true, "Open a browser in web mode")

	cmd.ParseFlags(statusCmd)

	return statusCmd
}

func (cmd *StatusCmd) Run(cobraCmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(args) > 0 {
		return errors.New("unexpected non-flag arguments to 'tailscale status'")
	}

	// Create network
	tsNet := tailscale.NewTSNet(&tailscale.TSNetConfig{
		AccessKey: cmd.AccessKey,
		Host:      tailscale.RemoveProtocol(cmd.PlatformHost),
		Hostname:  cmd.NetworkHostname,
	})
	// Run tailscale up and wait until we have a connected client
	done := make(chan bool)
	go func() {
		err := tsNet.Start(ctx, done)
		if err != nil {
			log.Fatalf("cannot start tsNet server: %v", err)
		}
	}()
	<-done

	// Get tailscale API client
	localClient, err := tsNet.LocalClient()
	if err != nil {
		return fmt.Errorf("cannot get local client: %w", err)
	}
	return cmd.runStatus(ctx, localClient)
}

func (cmd *StatusCmd) runStatus(ctx context.Context, localClient *ts.LocalClient) error {
	getStatus := localClient.Status
	if !cmd.peers {
		getStatus = localClient.StatusWithoutPeers
	}
	st, err := getStatus(ctx)
	if err != nil {
		return fixTailscaledConnectError(err)
	}
	if cmd.json {
		if cmd.active {
			for peer, ps := range st.Peer {
				if !ps.Active {
					delete(st.Peer, peer)
				}
			}
		}
		j, err := json.MarshalIndent(st, "", "  ")
		if err != nil {
			return err
		}
		printf("%s", j)
		return nil
	}
	if cmd.web {
		ln, err := net.Listen("tcp", cmd.listen)
		if err != nil {
			return err
		}
		statusURL := netmon.HTTPOfListener(ln)
		printf("Serving Tailscale status at %v ...\n", statusURL)
		go func() {
			<-ctx.Done()
			ln.Close()
		}()
		if cmd.browser {
			go func() {
				if err = open2.Open(ctx, statusURL, nil); err != nil {
					errf("Could not open: %w", err)
				}
			}()
		}
		err = http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.RequestURI != "/" {
				http.NotFound(w, r)
				return
			}
			st, err := localClient.Status(ctx)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			st.WriteHTML(w)
		}))
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}

	printHealth := func() {
		printf("# Health check:\n")
		for _, m := range st.Health {
			printf("#     - %s\n", m)
		}
	}

	description, ok := isRunningOrStarting(st)
	if !ok {
		// print health check information if we're in a weird state, as it might
		// provide context about why we're in that weird state.
		if len(st.Health) > 0 && (st.BackendState == ipn.Starting.String() || st.BackendState == ipn.NoState.String()) {
			printHealth()
			outln()
		}
		outln(description)
		os.Exit(1)
	}

	var buf bytes.Buffer
	f := func(format string, a ...any) { fmt.Fprintf(&buf, format, a...) }
	printPS := func(ps *ipnstate.PeerStatus) {
		f("%-15s %-20s %-12s %-7s ",
			firstIPString(ps.TailscaleIPs),
			dnsOrQuoteHostname(st, ps),
			ownerLogin(st, ps),
			ps.OS,
		)
		relay := ps.Relay
		anyTraffic := ps.TxBytes != 0 || ps.RxBytes != 0
		var offline string
		if !ps.Online {
			offline = "; offline"
		}
		if !ps.Active {
			if ps.ExitNode {
				f("idle; exit node" + offline)
			} else if ps.ExitNodeOption {
				f("idle; offers exit node" + offline)
			} else if anyTraffic {
				f("idle" + offline)
			} else if !ps.Online {
				f("offline")
			} else {
				f("-")
			}
		} else {
			f("active; ")
			if ps.ExitNode {
				f("exit node; ")
			} else if ps.ExitNodeOption {
				f("offers exit node; ")
			}
			if relay != "" && ps.CurAddr == "" {
				f("relay %q", relay)
			} else if ps.CurAddr != "" {
				f("direct %s", ps.CurAddr)
			}
			if !ps.Online {
				f("; offline")
			}
		}
		if anyTraffic {
			f(", tx %d rx %d", ps.TxBytes, ps.RxBytes)
		}
		f("\n")
	}

	if cmd.self && st.Self != nil {
		printPS(st.Self)
	}

	locBasedExitNode := false
	if cmd.peers {
		var peers []*ipnstate.PeerStatus
		for _, peer := range st.Peers() {
			ps := st.Peer[peer]
			if ps.ShareeNode {
				continue
			}
			if ps.Location != nil && ps.ExitNodeOption && !ps.ExitNode {
				// Location based exit nodes are only shown with the
				// `exit-node list` command.
				locBasedExitNode = true
				continue
			}
			peers = append(peers, ps)
		}
		ipnstate.SortPeers(peers)
		for _, ps := range peers {
			if cmd.active && !ps.Active {
				continue
			}
			printPS(ps)
		}
	}
	Stdout.Write(buf.Bytes())
	if locBasedExitNode {
		outln()
		printf("# To see the full list of exit nodes, including location-based exit nodes, run `tailscale exit-node list`  \n")
	}
	if len(st.Health) > 0 {
		outln()
		printHealth()
	}
	printFunnelStatus(ctx, localClient)
	return nil
}

// printFunnelStatus prints the status of the funnel, if it's running.
// It prints nothing if the funnel is not running.
func printFunnelStatus(ctx context.Context, localClient *ts.LocalClient) {
	sc, err := localClient.GetServeConfig(ctx)
	if err != nil {
		outln()
		printf("# Funnel:\n")
		printf("#     - Unable to get Funnel status: %v\n", err)
		return
	}
	if !sc.IsFunnelOn() {
		return
	}
	outln()
	printf("# Funnel on:\n")
	for hp, on := range sc.AllowFunnel {
		if !on { // if present, should be on
			continue
		}
		sni, portStr, _ := net.SplitHostPort(string(hp))
		p, _ := strconv.ParseUint(portStr, 10, 16)
		isTCP := sc.IsTCPForwardingOnPort(uint16(p))
		url := "https://"
		if isTCP {
			url = "tcp://"
		}
		url += sni
		if isTCP || p != 443 {
			url += ":" + portStr
		}
		printf("#     - %s\n", url)
	}
	outln()
}

// isRunningOrStarting reports whether st is in state Running or Starting.
// It also returns a description of the status suitable to display to a user.
func isRunningOrStarting(st *ipnstate.Status) (description string, ok bool) {
	switch st.BackendState {
	default:
		return fmt.Sprintf("unexpected state: %s", st.BackendState), false
	case ipn.Stopped.String():
		return "Tailscale is stopped.", false
	case ipn.NeedsLogin.String():
		s := "Logged out."
		if st.AuthURL != "" {
			s += fmt.Sprintf("\nLog in at: %s", st.AuthURL)
		}
		return s, false
	case ipn.NeedsMachineAuth.String():
		return "Machine is not yet approved by tailnet admin.", false
	case ipn.Running.String(), ipn.Starting.String():
		return st.BackendState, true
	}
}

func dnsOrQuoteHostname(st *ipnstate.Status, ps *ipnstate.PeerStatus) string {
	baseName := dnsname.TrimSuffix(ps.DNSName, st.CurrentTailnet.MagicDNSSuffix)
	if baseName != "" {
		if strings.HasPrefix(baseName, "xn-") {
			if u, err := idna.ToUnicode(baseName); err == nil {
				return fmt.Sprintf("%s (%s)", baseName, u)
			}
		}
		return baseName
	}
	return fmt.Sprintf("(%q)", dnsname.SanitizeHostname(ps.HostName))
}

func ownerLogin(st *ipnstate.Status, ps *ipnstate.PeerStatus) string {
	// We prioritize showing the name of the sharer as the owner of a node if
	// it's different from the node's user. This is less surprising: if user B
	// from a company shares user's C node from the same company with user A who
	// don't know user C, user A might be surprised to see user C listed in
	// their netmap. We've historically (2021-01..2023-08) always shown the
	// sharer's name in the UI. Perhaps we want to show both here? But the CLI's
	// a bit space constrained.
	uid := cmp.Or(ps.AltSharerUserID, ps.UserID)
	if uid.IsZero() {
		return "-"
	}
	u, ok := st.User[uid]
	if !ok {
		return fmt.Sprint(uid)
	}
	if i := strings.Index(u.LoginName, "@"); i != -1 {
		return u.LoginName[:i+1]
	}
	return u.LoginName
}

func firstIPString(v []netip.Addr) string {
	if len(v) == 0 {
		return ""
	}
	return v[0].String()
}
