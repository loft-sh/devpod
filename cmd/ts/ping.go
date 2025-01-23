package ts

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/tailscale"

	"github.com/spf13/cobra"
	ts "tailscale.com/client/tailscale"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

type PingCmd struct {
	*TsNetFlags

	num         int
	size        int
	untilDirect bool
	verbose     bool
	tsmp        bool
	icmp        bool
	peerAPI     bool
	timeout     time.Duration
}

func NewPingCmd() *cobra.Command {
	cmd := &PingCmd{TsNetFlags: &TsNetFlags{}}
	var pingCmd = &cobra.Command{
		Use:     "ping",
		Example: "tailscale ping <hostname-or-IP>",
		Short:   "Ping a host at the Tailscale layer, see how it routed",
		Long: strings.TrimSpace(`
			The 'tailscale ping' command pings a peer node from the Tailscale layer
			and reports which route it took for each response. The first ping or
			so will likely go over DERP (Tailscale's TCP relay protocol) while NAT
			traversal finds a direct path through.
			
			If 'tailscale ping' works but a normal ping does not, that means one
			side's operating system firewall is blocking packets; 'tailscale ping'
			does not inject packets into either side's TUN devices.
			
			By default, 'tailscale ping' stops after 10 pings or once a direct
			(non-DERP) path has been established, whichever comes first.
			
			The provided hostname must resolve to or be a Tailscale IP
			(e.g. 100.x.y.z) or a subnet IP advertised by a Tailscale
			relay node.`,
		),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			ctx := cobraCmd.Context()
			tsNet := tailscale.NewTSNet(&tailscale.TSNetConfig{
				AccessKey: cmd.AccessKey,
				Host:      tailscale.RemoveProtocol(cmd.PlatformHost),
				Hostname:  cmd.NetworkHostname,
			})

			done := make(chan bool)

			go func() {
				err := tsNet.Start(ctx, done)
				if err != nil {
					log.Fatalf("cannot start tsNet server: %v", err)
				}
			}()

			time.Sleep(5 * time.Second)

			localClient, err := tsNet.LocalClient()
			if err != nil {
				return fmt.Errorf("cannot create local client: %w", err)
			}
			return cmd.runPing(ctx, args, localClient)
		},
	}

	pingCmd.Flags().BoolVar(&cmd.verbose, "verbose", false, "verbose output")
	pingCmd.Flags().BoolVar(&cmd.untilDirect, "until-direct", true, "stop once a direct path is established")
	pingCmd.Flags().BoolVar(&cmd.tsmp, "tsmp", false, "do a TSMP-level ping (through WireGuard, but not either host OS stack)")
	pingCmd.Flags().BoolVar(&cmd.icmp, "icmp", false, "do a ICMP-level ping (through WireGuard, but not the local host OS stack)")
	pingCmd.Flags().BoolVar(&cmd.peerAPI, "peerapi", false, "try hitting the peer's peerapi HTTP server")
	pingCmd.Flags().IntVar(&cmd.num, "c", 10, "max number of pings to send. 0 for infinity.")
	pingCmd.Flags().DurationVar(&cmd.timeout, "timeout", 5*time.Second, "timeout before giving up on a ping")
	pingCmd.Flags().IntVar(&cmd.size, "size", 0, "size of the ping message (disco pings only). 0 for minimum size.")

	cmd.ParseFlags(pingCmd)

	return pingCmd
}

func (cmd *PingCmd) pingType() tailcfg.PingType {
	if cmd.tsmp {
		return tailcfg.PingTSMP
	}
	if cmd.icmp {
		return tailcfg.PingICMP
	}
	if cmd.peerAPI {
		return tailcfg.PingPeerAPI
	}
	return tailcfg.PingDisco
}

func (cmd *PingCmd) runPing(ctx context.Context, args []string, localClient *ts.LocalClient) error {
	st, err := localClient.Status(ctx)
	if err != nil {
		return fixTailscaledConnectError(err)
	}
	description, ok := isRunningOrStarting(st)
	if !ok {
		printf("%s\n", description)
		os.Exit(1)
	}

	if len(args) != 1 || args[0] == "" {
		return errors.New("usage: tailscale ping <hostname-or-IP>")
	}
	var ip string

	hostOrIP := args[0]
	ip, self, err := cmd.tailscaleIPFromArg(ctx, hostOrIP, localClient)
	if err != nil {
		return err
	}
	if self {
		printf("%v is local Tailscale IP\n", ip)
		return nil
	}

	if cmd.verbose && ip != hostOrIP {
		log.Printf("lookup %q => %q", hostOrIP, ip)
	}

	n := 0
	anyPong := false
	for {
		n++
		ctx, cancel := context.WithTimeout(ctx, cmd.timeout)
		pr, err := localClient.PingWithOpts(ctx, netip.MustParseAddr(ip), cmd.pingType(), ts.PingOpts{Size: cmd.size})
		cancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				printf("ping %q timed out\n", ip)
				if n == cmd.num {
					if !anyPong {
						return errors.New("no reply")
					}
					return nil
				}
				continue
			}
			return err
		}
		if pr.Err != "" {
			if pr.IsLocalIP {
				outln(pr.Err)
				return nil
			}
			return errors.New(pr.Err)
		}
		latency := time.Duration(pr.LatencySeconds * float64(time.Second)).Round(time.Millisecond)
		via := pr.Endpoint
		if pr.DERPRegionID != 0 {
			via = fmt.Sprintf("DERP(%s)", pr.DERPRegionCode)
		}
		if via == "" {
			// TODO(bradfitz): populate the rest of ipnstate.PingResult for TSMP queries?
			// For now just say which protocol it used.
			via = string(cmd.pingType())
		}
		if cmd.peerAPI {
			printf("hit peerapi of %s (%s) at %s in %s\n", pr.NodeIP, pr.NodeName, pr.PeerAPIURL, latency)
			return nil
		}
		anyPong = true
		extra := ""
		if pr.PeerAPIPort != 0 {
			extra = fmt.Sprintf(", %d", pr.PeerAPIPort)
		}
		printf("pong from %s (%s%s) via %v in %v\n", pr.NodeName, pr.NodeIP, extra, via, latency)
		if cmd.tsmp || cmd.icmp {
			return nil
		}
		if pr.Endpoint != "" && cmd.untilDirect {
			return nil
		}
		time.Sleep(time.Second)

		if n == cmd.num {
			if !anyPong {
				return errors.New("no reply")
			}
			if cmd.untilDirect {
				return errors.New("direct connection not established")
			}
			return nil
		}
	}
}

func (cmd *PingCmd) tailscaleIPFromArg(ctx context.Context, hostOrIP string, localClient *ts.LocalClient) (ip string, self bool, err error) {
	// If the argument is an IP address, use it directly without any resolution.
	if net.ParseIP(hostOrIP) != nil {
		return hostOrIP, false, nil
	}

	// Otherwise, try to resolve it first from the network peer list.
	st, err := localClient.Status(ctx)
	if err != nil {
		return "", false, err
	}
	match := func(ps *ipnstate.PeerStatus) bool {
		return strings.EqualFold(hostOrIP, dnsOrQuoteHostname(st, ps)) || hostOrIP == ps.DNSName
	}
	for _, ps := range st.Peer {
		if match(ps) {
			if len(ps.TailscaleIPs) == 0 {
				return "", false, errors.New("node found but lacks an IP")
			}
			return ps.TailscaleIPs[0].String(), false, nil
		}
	}
	if match(st.Self) && len(st.Self.TailscaleIPs) > 0 {
		return st.Self.TailscaleIPs[0].String(), true, nil
	}

	// Finally, use DNS.
	var res net.Resolver
	if addrs, err := res.LookupHost(ctx, hostOrIP); err != nil {
		return "", false, fmt.Errorf("error looking up IP of %q: %v", hostOrIP, err)
	} else if len(addrs) == 0 {
		return "", false, fmt.Errorf("no IPs found for %q", hostOrIP)
	} else {
		return addrs[0], false, nil
	}
}
