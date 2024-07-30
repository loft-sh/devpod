// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package derp

// TODO(crawshaw): with predefined serverKey in clients and HMAC on packets we could skip TLS

import (
	"bufio"
	"context"
	"crypto/ed25519"
	crand "crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"net/netip"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go4.org/mem"
	"golang.org/x/sync/errgroup"
	"tailscale.com/client/tailscale"
	"tailscale.com/disco"
	"tailscale.com/envknob"
	"tailscale.com/metrics"
	"tailscale.com/syncs"
	"tailscale.com/tstime"
	"tailscale.com/tstime/rate"
	"tailscale.com/types/key"
	"tailscale.com/types/logger"
	"tailscale.com/util/set"
	"tailscale.com/version"
)

// verboseDropKeys is the set of destination public keys that should
// verbosely log whenever DERP drops a packet.
var verboseDropKeys = map[key.NodePublic]bool{}

func init() {
	keys := envknob.String("TS_DEBUG_VERBOSE_DROPS")
	if keys == "" {
		return
	}
	for _, keyStr := range strings.Split(keys, ",") {
		k, err := key.ParseNodePublicUntyped(mem.S(keyStr))
		if err != nil {
			log.Printf("ignoring invalid debug key %q: %v", keyStr, err)
		} else {
			verboseDropKeys[k] = true
		}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	perClientSendQueueDepth = 32 // packets buffered for sending
	writeTimeout            = 2 * time.Second
)

// dupPolicy is a temporary (2021-08-30) mechanism to change the policy
// of how duplicate connection for the same key are handled.
type dupPolicy int8

const (
	// lastWriterIsActive is a dupPolicy where the connection
	// to send traffic for a peer is the active one.
	lastWriterIsActive dupPolicy = iota

	// disableFighters is a dupPolicy that detects if peers
	// are trying to send interleaved with each other and
	// then disables all of them.
	disableFighters
)

type align64 [0]atomic.Int64 // for side effect of its 64-bit alignment

// Server is a DERP server.
type Server struct {
	// WriteTimeout, if non-zero, specifies how long to wait
	// before failing when writing to a client.
	WriteTimeout time.Duration

	privateKey  key.NodePrivate
	publicKey   key.NodePublic
	logf        logger.Logf
	memSys0     uint64 // runtime.MemStats.Sys at start (or early-ish)
	meshKey     string
	limitedLogf logger.Logf
	metaCert    []byte // the encoded x509 cert to send after LetsEncrypt cert+intermediate
	dupPolicy   dupPolicy
	debug       bool

	// Counters:
	packetsSent, bytesSent       expvar.Int
	packetsRecv, bytesRecv       expvar.Int
	packetsRecvByKind            metrics.LabelMap
	packetsRecvDisco             *expvar.Int
	packetsRecvOther             *expvar.Int
	_                            align64
	packetsDropped               expvar.Int
	packetsDroppedReason         metrics.LabelMap
	packetsDroppedReasonCounters []*expvar.Int // indexed by dropReason
	packetsDroppedType           metrics.LabelMap
	packetsDroppedTypeDisco      *expvar.Int
	packetsDroppedTypeOther      *expvar.Int
	_                            align64
	packetsForwardedOut          expvar.Int
	packetsForwardedIn           expvar.Int
	peerGoneDisconnectedFrames   expvar.Int // number of peer disconnected frames sent
	peerGoneNotHereFrames        expvar.Int // number of peer not here frames sent
	gotPing                      expvar.Int // number of ping frames from client
	sentPong                     expvar.Int // number of pong frames enqueued to client
	accepts                      expvar.Int
	curClients                   expvar.Int
	curHomeClients               expvar.Int // ones with preferred
	dupClientKeys                expvar.Int // current number of public keys we have 2+ connections for
	dupClientConns               expvar.Int // current number of connections sharing a public key
	dupClientConnTotal           expvar.Int // total number of accepted connections when a dup key existed
	unknownFrames                expvar.Int
	homeMovesIn                  expvar.Int // established clients announce home server moves in
	homeMovesOut                 expvar.Int // established clients announce home server moves out
	multiForwarderCreated        expvar.Int
	multiForwarderDeleted        expvar.Int
	removePktForwardOther        expvar.Int
	avgQueueDuration             *uint64          // In milliseconds; accessed atomically
	tcpRtt                       metrics.LabelMap // histogram

	// verifyClients only accepts client connections to the DERP server if the clientKey is a
	// known peer in the network, as specified by a running tailscaled's client's LocalAPI.
	verifyClients bool

	mu       sync.Mutex
	closed   bool
	netConns map[Conn]chan struct{} // chan is closed when conn closes
	clients  map[key.NodePublic]clientSet
	watchers set.Set[*sclient] // mesh peers
	// clientsMesh tracks all clients in the cluster, both locally
	// and to mesh peers.  If the value is nil, that means the
	// peer is only local (and thus in the clients Map, but not
	// remote). If the value is non-nil, it's remote (+ maybe also
	// local).
	clientsMesh map[key.NodePublic]PacketForwarder
	// sentTo tracks which peers have sent to which other peers,
	// and at which connection number. This isn't on sclient
	// because it includes intra-region forwarded packets as the
	// src.
	sentTo map[key.NodePublic]map[key.NodePublic]int64 // src => dst => dst's latest sclient.connNum

	// maps from netip.AddrPort to a client's public key
	keyOfAddr map[netip.AddrPort]key.NodePublic

	clock tstime.Clock
}

// clientSet represents 1 or more *sclients.
//
// The two implementations are singleClient and *dupClientSet.
//
// In the common case, client should only have one connection to the
// DERP server for a given key. When they're connected multiple times,
// we record their set of connections in dupClientSet and keep their
// connections open to make them happy (to keep them from spinning,
// etc) and keep track of which is the latest connection. If only the last
// is sending traffic, that last one is the active connection and it
// gets traffic.  Otherwise, in the case of a cloned node key, the
// whole set of dups doesn't receive data frames.
//
// All methods should only be called while holding Server.mu.
//
// TODO(bradfitz): Issue 2746: in the future we'll send some sort of
// "health_error" frame to them that'll communicate to the end users
// that they cloned a device key, and we'll also surface it in the
// admin panel, etc.
type clientSet interface {
	// ActiveClient returns the most recently added client to
	// the set, as long as it hasn't been disabled, in which
	// case it returns nil.
	ActiveClient() *sclient

	// Len returns the number of clients in the set.
	Len() int

	// ForeachClient calls f for each client in the set.
	ForeachClient(f func(*sclient))
}

// singleClient is a clientSet of a single connection.
// This is the common case.
type singleClient struct{ c *sclient }

func (s singleClient) ActiveClient() *sclient         { return s.c }
func (s singleClient) Len() int                       { return 1 }
func (s singleClient) ForeachClient(f func(*sclient)) { f(s.c) }

// A dupClientSet is a clientSet of more than 1 connection.
//
// This can occur in some reasonable cases (temporarily while users
// are changing networks) or in the case of a cloned key. In the
// cloned key case, both peers are speaking and the clients get
// disabled.
//
// All fields are guarded by Server.mu.
type dupClientSet struct {
	// set is the set of connected clients for sclient.key.
	set set.Set[*sclient]

	// last is the most recent addition to set, or nil if the most
	// recent one has since disconnected and nobody else has send
	// data since.
	last *sclient

	// sendHistory is a log of which members of set have sent
	// frames to the derp server, with adjacent duplicates
	// removed. When a member of set is removed, the same
	// element(s) are removed from sendHistory.
	sendHistory []*sclient
}

func (s *dupClientSet) ActiveClient() *sclient {
	if s.last != nil && !s.last.isDisabled.Load() {
		return s.last
	}
	return nil
}
func (s *dupClientSet) Len() int { return len(s.set) }
func (s *dupClientSet) ForeachClient(f func(*sclient)) {
	for c := range s.set {
		f(c)
	}
}

// removeClient removes c from s and reports whether it was in s
// to begin with.
func (s *dupClientSet) removeClient(c *sclient) bool {
	n := len(s.set)
	delete(s.set, c)
	if s.last == c {
		s.last = nil
	}
	if len(s.set) == n {
		return false
	}

	trim := s.sendHistory[:0]
	for _, v := range s.sendHistory {
		if s.set.Contains(v) && (len(trim) == 0 || trim[len(trim)-1] != v) {
			trim = append(trim, v)
		}
	}
	for i := len(trim); i < len(s.sendHistory); i++ {
		s.sendHistory[i] = nil
	}
	s.sendHistory = trim
	if s.last == nil && len(s.sendHistory) > 0 {
		s.last = s.sendHistory[len(s.sendHistory)-1]
	}
	return true
}

// PacketForwarder is something that can forward packets.
//
// It's mostly an interface for circular dependency reasons; the
// typical implementation is derphttp.Client. The other implementation
// is a multiForwarder, which this package creates as needed if a
// public key gets more than one PacketForwarder registered for it.
type PacketForwarder interface {
	ForwardPacket(src, dst key.NodePublic, payload []byte) error
	String() string
}

// Conn is the subset of the underlying net.Conn the DERP Server needs.
// It is a defined type so that non-net connections can be used.
type Conn interface {
	io.WriteCloser
	LocalAddr() net.Addr
	// The *Deadline methods follow the semantics of net.Conn.
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

// NewServer returns a new DERP server. It doesn't listen on its own.
// Connections are given to it via Server.Accept.
func NewServer(privateKey key.NodePrivate, logf logger.Logf) *Server {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	s := &Server{
		debug:                envknob.Bool("DERP_DEBUG_LOGS"),
		privateKey:           privateKey,
		publicKey:            privateKey.Public(),
		logf:                 logf,
		limitedLogf:          logger.RateLimitedFn(logf, 30*time.Second, 5, 100),
		packetsRecvByKind:    metrics.LabelMap{Label: "kind"},
		packetsDroppedReason: metrics.LabelMap{Label: "reason"},
		packetsDroppedType:   metrics.LabelMap{Label: "type"},
		clients:              map[key.NodePublic]clientSet{},
		clientsMesh:          map[key.NodePublic]PacketForwarder{},
		netConns:             map[Conn]chan struct{}{},
		memSys0:              ms.Sys,
		watchers:             set.Set[*sclient]{},
		sentTo:               map[key.NodePublic]map[key.NodePublic]int64{},
		avgQueueDuration:     new(uint64),
		tcpRtt:               metrics.LabelMap{Label: "le"},
		keyOfAddr:            map[netip.AddrPort]key.NodePublic{},
		clock:                tstime.StdClock{},
	}
	s.initMetacert()
	s.packetsRecvDisco = s.packetsRecvByKind.Get("disco")
	s.packetsRecvOther = s.packetsRecvByKind.Get("other")
	s.packetsDroppedReasonCounters = []*expvar.Int{
		s.packetsDroppedReason.Get("unknown_dest"),
		s.packetsDroppedReason.Get("unknown_dest_on_fwd"),
		s.packetsDroppedReason.Get("gone_disconnected"),
		s.packetsDroppedReason.Get("gone_not_here"),
		s.packetsDroppedReason.Get("queue_head"),
		s.packetsDroppedReason.Get("queue_tail"),
		s.packetsDroppedReason.Get("write_error"),
	}
	s.packetsDroppedTypeDisco = s.packetsDroppedType.Get("disco")
	s.packetsDroppedTypeOther = s.packetsDroppedType.Get("other")
	return s
}

// SetMesh sets the pre-shared key that regional DERP servers used to mesh
// amongst themselves.
//
// It must be called before serving begins.
func (s *Server) SetMeshKey(v string) {
	s.meshKey = v
}

// SetVerifyClients sets whether this DERP server verifies clients through tailscaled.
//
// It must be called before serving begins.
func (s *Server) SetVerifyClient(v bool) {
	s.verifyClients = v
}

// HasMeshKey reports whether the server is configured with a mesh key.
func (s *Server) HasMeshKey() bool { return s.meshKey != "" }

// MeshKey returns the configured mesh key, if any.
func (s *Server) MeshKey() string { return s.meshKey }

// PrivateKey returns the server's private key.
func (s *Server) PrivateKey() key.NodePrivate { return s.privateKey }

// PublicKey returns the server's public key.
func (s *Server) PublicKey() key.NodePublic { return s.publicKey }

// Close closes the server and waits for the connections to disconnect.
func (s *Server) Close() error {
	s.mu.Lock()
	wasClosed := s.closed
	s.closed = true
	s.mu.Unlock()
	if wasClosed {
		return nil
	}

	var closedChs []chan struct{}

	s.mu.Lock()
	for nc, closed := range s.netConns {
		nc.Close()
		closedChs = append(closedChs, closed)
	}
	s.mu.Unlock()

	for _, closed := range closedChs {
		<-closed
	}

	return nil
}

func (s *Server) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// IsClientConnectedForTest reports whether the client with specified key is connected.
// This is used in tests to verify that nodes are connected.
func (s *Server) IsClientConnectedForTest(k key.NodePublic) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.clients[k]
	if !ok {
		return false
	}
	return x.ActiveClient() != nil
}

// Accept adds a new connection to the server and serves it.
//
// The provided bufio ReadWriter must be already connected to nc.
// Accept blocks until the Server is closed or the connection closes
// on its own.
//
// Accept closes nc.
func (s *Server) Accept(ctx context.Context, nc Conn, brw *bufio.ReadWriter, remoteAddr string) {
	closed := make(chan struct{})

	s.mu.Lock()
	s.accepts.Add(1)             // while holding s.mu for connNum read on next line
	connNum := s.accepts.Value() // expvar sadly doesn't return new value on Add(1)
	s.netConns[nc] = closed
	s.mu.Unlock()

	defer func() {
		nc.Close()
		close(closed)

		s.mu.Lock()
		delete(s.netConns, nc)
		s.mu.Unlock()
	}()

	if err := s.accept(ctx, nc, brw, remoteAddr, connNum); err != nil && !s.isClosed() {
		s.logf("derp: %s: %v", remoteAddr, err)
	}
}

// initMetacert initialized s.metaCert with a self-signed x509 cert
// encoding this server's public key and protocol version. cmd/derper
// then sends this after the Let's Encrypt leaf + intermediate certs
// after the ServerHello (encrypted in TLS 1.3, not that it matters
// much).
//
// Then the client can save a round trip getting that and can start
// speaking DERP right away. (We don't use ALPN because that's sent in
// the clear and we're being paranoid to not look too weird to any
// middleboxes, given that DERP is an ultimate fallback path). But
// since the post-ServerHello certs are encrypted we can have the
// client also use them as a signal to be able to start speaking DERP
// right away, starting with its identity proof, encrypted to the
// server's public key.
//
// This RTT optimization fails where there's a corp-mandated
// TLS proxy with corp-mandated root certs on employee machines and
// and TLS proxy cleans up unnecessary certs. In that case we just fall
// back to the extra RTT.
func (s *Server) initMetacert() {
	pub, priv, err := ed25519.GenerateKey(crand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(ProtocolVersion),
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("derpkey%s", s.publicKey.UntypedHexString()),
		},
		// Windows requires NotAfter and NotBefore set:
		NotAfter:  s.clock.Now().Add(30 * 24 * time.Hour),
		NotBefore: s.clock.Now().Add(-30 * 24 * time.Hour),
		// Per https://github.com/golang/go/issues/51759#issuecomment-1071147836,
		// macOS requires BasicConstraints when subject == issuer:
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(crand.Reader, tmpl, tmpl, pub, priv)
	if err != nil {
		log.Fatalf("CreateCertificate: %v", err)
	}
	s.metaCert = cert
}

// MetaCert returns the server metadata cert that can be sent by the
// TLS server to let the client skip a round trip during start-up.
func (s *Server) MetaCert() []byte { return s.metaCert }

// registerClient notes that client c is now authenticated and ready for packets.
//
// If c.key is connected more than once, the earlier connection(s) are
// placed in a non-active state where we read from them (primarily to
// observe EOFs/timeouts) but won't send them frames on the assumption
// that they're dead.
func (s *Server) registerClient(c *sclient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	curSet := s.clients[c.key]
	switch curSet := curSet.(type) {
	case nil:
		s.clients[c.key] = singleClient{c}
		c.debugLogf("register single client")
	case singleClient:
		s.dupClientKeys.Add(1)
		s.dupClientConns.Add(2) // both old and new count
		s.dupClientConnTotal.Add(1)
		old := curSet.ActiveClient()
		old.isDup.Store(true)
		c.isDup.Store(true)
		s.clients[c.key] = &dupClientSet{
			last: c,
			set: set.Set[*sclient]{
				old: struct{}{},
				c:   struct{}{},
			},
			sendHistory: []*sclient{old},
		}
		c.debugLogf("register duplicate client")
	case *dupClientSet:
		s.dupClientConns.Add(1)     // the gauge
		s.dupClientConnTotal.Add(1) // the counter
		c.isDup.Store(true)
		curSet.set.Add(c)
		curSet.last = c
		curSet.sendHistory = append(curSet.sendHistory, c)
		c.debugLogf("register another duplicate client")
	}

	if _, ok := s.clientsMesh[c.key]; !ok {
		s.clientsMesh[c.key] = nil // just for varz of total users in cluster
	}
	s.keyOfAddr[c.remoteIPPort] = c.key
	s.curClients.Add(1)
	s.broadcastPeerStateChangeLocked(c.key, c.remoteIPPort, true)
}

// broadcastPeerStateChangeLocked enqueues a message to all watchers
// (other DERP nodes in the region, or trusted clients) that peer's
// presence changed.
//
// s.mu must be held.
func (s *Server) broadcastPeerStateChangeLocked(peer key.NodePublic, ipPort netip.AddrPort, present bool) {
	for w := range s.watchers {
		w.peerStateChange = append(w.peerStateChange, peerConnState{
			peer:    peer,
			present: present,
			ipPort:  ipPort,
		})
		go w.requestMeshUpdate()
	}
}

// unregisterClient removes a client from the server.
func (s *Server) unregisterClient(c *sclient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	set := s.clients[c.key]
	switch set := set.(type) {
	case nil:
		c.logf("[unexpected]; clients map is empty")
	case singleClient:
		c.debugLogf("removed connection")
		delete(s.clients, c.key)
		if v, ok := s.clientsMesh[c.key]; ok && v == nil {
			delete(s.clientsMesh, c.key)
			s.notePeerGoneFromRegionLocked(c.key)
		}
		s.broadcastPeerStateChangeLocked(c.key, netip.AddrPort{}, false)
	case *dupClientSet:
		c.debugLogf("removed duplicate client")
		if set.removeClient(c) {
			s.dupClientConns.Add(-1)
		} else {
			c.logf("[unexpected]; dup client set didn't shrink")
		}
		if set.Len() == 1 {
			s.dupClientConns.Add(-1) // again; for the original one's
			s.dupClientKeys.Add(-1)
			var remain *sclient
			for remain = range set.set {
				break
			}
			if remain == nil {
				panic("unexpected nil remain from single element dup set")
			}
			remain.isDisabled.Store(false)
			remain.isDup.Store(false)
			s.clients[c.key] = singleClient{remain}
		}
	}

	if c.canMesh {
		delete(s.watchers, c)
	}

	delete(s.keyOfAddr, c.remoteIPPort)

	s.curClients.Add(-1)
	if c.preferred {
		s.curHomeClients.Add(-1)
	}
}

// notePeerGoneFromRegionLocked sends peerGone frames to parties that
// key has sent to previously (whether those sends were from a local
// client or forwarded).  It must only be called after the key has
// been removed from clientsMesh.
func (s *Server) notePeerGoneFromRegionLocked(key key.NodePublic) {
	if _, ok := s.clientsMesh[key]; ok {
		panic("usage")
	}

	// Find still-connected peers and either notify that we've gone away
	// so they can drop their route entries to us (issue 150)
	// or move them over to the active client (in case a replaced client
	// connection is being unregistered).
	for pubKey, connNum := range s.sentTo[key] {
		set, ok := s.clients[pubKey]
		if !ok {
			continue
		}
		set.ForeachClient(func(peer *sclient) {
			if peer.connNum == connNum {
				go peer.requestPeerGoneWrite(key, PeerGoneReasonDisconnected)
			}
		})
	}
	delete(s.sentTo, key)
}

// requestPeerGoneWriteLimited sends a request to write a "peer gone"
// frame, but only in reply to a disco packet, and only if we haven't
// sent one recently.
func (c *sclient) requestPeerGoneWriteLimited(peer key.NodePublic, contents []byte, reason PeerGoneReasonType) {
	if disco.LooksLikeDiscoWrapper(contents) != true {
		return
	}

	if c.peerGoneLim.Allow() {
		go c.requestPeerGoneWrite(peer, reason)
	}
}

func (s *Server) addWatcher(c *sclient) {
	if !c.canMesh {
		panic("invariant: addWatcher called without permissions")
	}

	if c.key == s.publicKey {
		// We're connecting to ourself. Do nothing.
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Queue messages for each already-connected client.
	for peer, clientSet := range s.clients {
		ac := clientSet.ActiveClient()
		if ac == nil {
			continue
		}
		c.peerStateChange = append(c.peerStateChange, peerConnState{
			peer:    peer,
			present: true,
			ipPort:  ac.remoteIPPort,
		})
	}

	// And enroll the watcher in future updates (of both
	// connections & disconnections).
	s.watchers.Add(c)

	go c.requestMeshUpdate()
}

func (s *Server) accept(ctx context.Context, nc Conn, brw *bufio.ReadWriter, remoteAddr string, connNum int64) error {
	br := brw.Reader
	nc.SetDeadline(time.Now().Add(10 * time.Second))
	bw := &lazyBufioWriter{w: nc, lbw: brw.Writer}
	if err := s.sendServerKey(bw); err != nil {
		return fmt.Errorf("send server key: %v", err)
	}
	nc.SetDeadline(time.Now().Add(10 * time.Second))
	clientKey, clientInfo, err := s.recvClientKey(br)
	if err != nil {
		return fmt.Errorf("receive client key: %v", err)
	}
	if err := s.verifyClient(clientKey, clientInfo); err != nil {
		return fmt.Errorf("client %x rejected: %v", clientKey, err)
	}

	// At this point we trust the client so we don't time out.
	nc.SetDeadline(time.Time{})

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	remoteIPPort, _ := netip.ParseAddrPort(remoteAddr)

	c := &sclient{
		connNum:        connNum,
		s:              s,
		key:            clientKey,
		nc:             nc,
		br:             br,
		bw:             bw,
		logf:           logger.WithPrefix(s.logf, fmt.Sprintf("derp client %v%s: ", remoteAddr, clientKey.ShortString())),
		done:           ctx.Done(),
		remoteAddr:     remoteAddr,
		remoteIPPort:   remoteIPPort,
		connectedAt:    s.clock.Now(),
		sendQueue:      make(chan pkt, perClientSendQueueDepth),
		discoSendQueue: make(chan pkt, perClientSendQueueDepth),
		sendPongCh:     make(chan [8]byte, 1),
		peerGone:       make(chan peerGoneMsg),
		canMesh:        clientInfo.MeshKey != "" && clientInfo.MeshKey == s.meshKey,
		peerGoneLim:    rate.NewLimiter(rate.Every(time.Second), 3),
	}

	if c.canMesh {
		c.meshUpdate = make(chan struct{})
	}
	if clientInfo != nil {
		c.info = *clientInfo
		if envknob.Bool("DERP_PROBER_DEBUG_LOGS") && clientInfo.IsProber {
			c.debug = true
		}
	}
	if s.debug {
		c.debug = true
	}

	s.registerClient(c)
	defer s.unregisterClient(c)

	err = s.sendServerInfo(c.bw, clientKey)
	if err != nil {
		return fmt.Errorf("send server info: %v", err)
	}

	return c.run(ctx)
}

func (s *Server) debugLogf(format string, v ...any) {
	if s.debug {
		s.logf(format, v...)
	}
}

// run serves the client until there's an error.
// If the client hangs up or the server is closed, run returns nil, otherwise run returns an error.
func (c *sclient) run(ctx context.Context) error {
	// Launch sender, but don't return from run until sender goroutine is done.
	var grp errgroup.Group
	sendCtx, cancelSender := context.WithCancel(ctx)
	grp.Go(func() error { return c.sendLoop(sendCtx) })
	grp.Go(func() error { return c.statsLoop(sendCtx) })
	defer func() {
		cancelSender()
		if err := grp.Wait(); err != nil && !c.s.isClosed() {
			if errors.Is(err, context.Canceled) {
				c.debugLogf("sender canceled by reader exiting")
			} else {
				c.logf("sender failed: %v", err)
			}
		}
	}()

	for {
		ft, fl, err := readFrameHeader(c.br)
		c.debugLogf("read frame type %d len %d err %v", ft, fl, err)
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.debugLogf("read EOF")
				return nil
			}
			if c.s.isClosed() {
				c.logf("closing; server closed")
				return nil
			}
			return fmt.Errorf("client %s: readFrameHeader: %w", c.key.ShortString(), err)
		}
		c.s.noteClientActivity(c)
		switch ft {
		case frameNotePreferred:
			err = c.handleFrameNotePreferred(ft, fl)
		case frameSendPacket:
			err = c.handleFrameSendPacket(ft, fl)
		case frameForwardPacket:
			err = c.handleFrameForwardPacket(ft, fl)
		case frameWatchConns:
			err = c.handleFrameWatchConns(ft, fl)
		case frameClosePeer:
			err = c.handleFrameClosePeer(ft, fl)
		case framePing:
			err = c.handleFramePing(ft, fl)
		default:
			err = c.handleUnknownFrame(ft, fl)
		}
		if err != nil {
			return err
		}
	}
}

func (c *sclient) handleUnknownFrame(ft frameType, fl uint32) error {
	_, err := io.CopyN(io.Discard, c.br, int64(fl))
	return err
}

func (c *sclient) handleFrameNotePreferred(ft frameType, fl uint32) error {
	if fl != 1 {
		return fmt.Errorf("frameNotePreferred wrong size")
	}
	v, err := c.br.ReadByte()
	if err != nil {
		return fmt.Errorf("frameNotePreferred ReadByte: %v", err)
	}
	c.setPreferred(v != 0)
	return nil
}

func (c *sclient) handleFrameWatchConns(ft frameType, fl uint32) error {
	if fl != 0 {
		return fmt.Errorf("handleFrameWatchConns wrong size")
	}
	if !c.canMesh {
		return fmt.Errorf("insufficient permissions")
	}
	c.s.addWatcher(c)
	return nil
}

func (c *sclient) handleFramePing(ft frameType, fl uint32) error {
	c.s.gotPing.Add(1)
	var m PingMessage
	if fl < uint32(len(m)) {
		return fmt.Errorf("short ping: %v", fl)
	}
	if fl > 1000 {
		// unreasonably extra large. We leave some extra
		// space for future extensibility, but not too much.
		return fmt.Errorf("ping body too large: %v", fl)
	}
	_, err := io.ReadFull(c.br, m[:])
	if err != nil {
		return err
	}
	if extra := int64(fl) - int64(len(m)); extra > 0 {
		_, err = io.CopyN(io.Discard, c.br, extra)
	}
	select {
	case c.sendPongCh <- [8]byte(m):
	default:
		// They're pinging too fast. Ignore.
		// TODO(bradfitz): add a rate limiter too.
	}
	return err
}

func (c *sclient) handleFrameClosePeer(ft frameType, fl uint32) error {
	if fl != keyLen {
		return fmt.Errorf("handleFrameClosePeer wrong size")
	}
	if !c.canMesh {
		return fmt.Errorf("insufficient permissions")
	}
	var targetKey key.NodePublic
	if err := targetKey.ReadRawWithoutAllocating(c.br); err != nil {
		return err
	}
	s := c.s

	s.mu.Lock()
	defer s.mu.Unlock()

	if set, ok := s.clients[targetKey]; ok {
		if set.Len() == 1 {
			c.logf("frameClosePeer closing peer %x", targetKey)
		} else {
			c.logf("frameClosePeer closing peer %x (%d connections)", targetKey, set.Len())
		}
		set.ForeachClient(func(target *sclient) {
			go target.nc.Close()
		})
	} else {
		c.logf("frameClosePeer failed to find peer %x", targetKey)
	}

	return nil
}

// handleFrameForwardPacket reads a "forward packet" frame from the client
// (which must be a trusted client, a peer in our mesh).
func (c *sclient) handleFrameForwardPacket(ft frameType, fl uint32) error {
	if !c.canMesh {
		return fmt.Errorf("insufficient permissions")
	}
	s := c.s

	srcKey, dstKey, contents, err := s.recvForwardPacket(c.br, fl)
	if err != nil {
		return fmt.Errorf("client %x: recvForwardPacket: %v", c.key, err)
	}
	s.packetsForwardedIn.Add(1)

	var dstLen int
	var dst *sclient

	s.mu.Lock()
	if set, ok := s.clients[dstKey]; ok {
		dstLen = set.Len()
		dst = set.ActiveClient()
	}
	if dst != nil {
		s.notePeerSendLocked(srcKey, dst)
	}
	s.mu.Unlock()

	if dst == nil {
		reason := dropReasonUnknownDestOnFwd
		if dstLen > 1 {
			reason = dropReasonDupClient
		} else {
			c.requestPeerGoneWriteLimited(dstKey, contents, PeerGoneReasonNotHere)
		}
		s.recordDrop(contents, srcKey, dstKey, reason)
		return nil
	}

	dst.debugLogf("received forwarded packet from %s via %s", srcKey.ShortString(), c.key.ShortString())

	return c.sendPkt(dst, pkt{
		bs:         contents,
		enqueuedAt: c.s.clock.Now(),
		src:        srcKey,
	})
}

// notePeerSendLocked records that src sent to dst.  We keep track of
// that so when src disconnects, we can tell dst (if it's still
// around) that src is gone (a peerGone frame).
func (s *Server) notePeerSendLocked(src key.NodePublic, dst *sclient) {
	m, ok := s.sentTo[src]
	if !ok {
		m = map[key.NodePublic]int64{}
		s.sentTo[src] = m
	}
	m[dst.key] = dst.connNum
}

// handleFrameSendPacket reads a "send packet" frame from the client.
func (c *sclient) handleFrameSendPacket(ft frameType, fl uint32) error {
	s := c.s

	dstKey, contents, err := s.recvPacket(c.br, fl)
	if err != nil {
		return fmt.Errorf("client %x: recvPacket: %v", c.key, err)
	}

	var fwd PacketForwarder
	var dstLen int
	var dst *sclient

	s.mu.Lock()
	if set, ok := s.clients[dstKey]; ok {
		dstLen = set.Len()
		dst = set.ActiveClient()
	}
	if dst != nil {
		s.notePeerSendLocked(c.key, dst)
	} else if dstLen < 1 {
		fwd = s.clientsMesh[dstKey]
	}
	s.mu.Unlock()

	if dst == nil {
		if fwd != nil {
			s.packetsForwardedOut.Add(1)
			err := fwd.ForwardPacket(c.key, dstKey, contents)
			c.debugLogf("SendPacket for %s, forwarding via %s: %v", dstKey.ShortString(), fwd, err)
			if err != nil {
				// TODO:
				return nil
			}
			return nil
		}
		reason := dropReasonUnknownDest
		if dstLen > 1 {
			reason = dropReasonDupClient
		} else {
			c.requestPeerGoneWriteLimited(dstKey, contents, PeerGoneReasonNotHere)
		}
		s.recordDrop(contents, c.key, dstKey, reason)
		c.debugLogf("SendPacket for %s, dropping with reason=%s", dstKey.ShortString(), reason)
		return nil
	}
	c.debugLogf("SendPacket for %s, sending directly", dstKey.ShortString())

	p := pkt{
		bs:         contents,
		enqueuedAt: c.s.clock.Now(),
		src:        c.key,
	}
	return c.sendPkt(dst, p)
}

func (c *sclient) debugLogf(format string, v ...any) {
	if c.debug {
		c.logf(format, v...)
	}
}

// dropReason is why we dropped a DERP frame.
type dropReason int

//go:generate go run tailscale.com/cmd/addlicense -file dropreason_string.go go run golang.org/x/tools/cmd/stringer -type=dropReason -trimprefix=dropReason

const (
	dropReasonUnknownDest      dropReason = iota // unknown destination pubkey
	dropReasonUnknownDestOnFwd                   // unknown destination pubkey on a derp-forwarded packet
	dropReasonGoneDisconnected                   // destination tailscaled disconnected before we could send
	dropReasonQueueHead                          // destination queue is full, dropped packet at queue head
	dropReasonQueueTail                          // destination queue is full, dropped packet at queue tail
	dropReasonWriteError                         // OS write() failed
	dropReasonDupClient                          // the public key is connected 2+ times (active/active, fighting)
)

func (s *Server) recordDrop(packetBytes []byte, srcKey, dstKey key.NodePublic, reason dropReason) {
	s.packetsDropped.Add(1)
	s.packetsDroppedReasonCounters[reason].Add(1)
	looksDisco := disco.LooksLikeDiscoWrapper(packetBytes)
	if looksDisco {
		s.packetsDroppedTypeDisco.Add(1)
	} else {
		s.packetsDroppedTypeOther.Add(1)
	}
	if verboseDropKeys[dstKey] {
		// Preformat the log string prior to calling limitedLogf. The
		// limiter acts based on the format string, and we want to
		// rate-limit per src/dst keys, not on the generic "dropped
		// stuff" message.
		msg := fmt.Sprintf("drop (%s) %s -> %s", srcKey.ShortString(), reason, dstKey.ShortString())
		s.limitedLogf(msg)
	}
	s.debugLogf("dropping packet reason=%s dst=%s disco=%v", reason, dstKey, looksDisco)
}

func (c *sclient) sendPkt(dst *sclient, p pkt) error {
	s := c.s
	dstKey := dst.key

	// Attempt to queue for sending up to 3 times. On each attempt, if
	// the queue is full, try to drop from queue head to prioritize
	// fresher packets.
	sendQueue := dst.sendQueue
	if disco.LooksLikeDiscoWrapper(p.bs) {
		sendQueue = dst.discoSendQueue
	}
	for attempt := 0; attempt < 3; attempt++ {
		select {
		case <-dst.done:
			s.recordDrop(p.bs, c.key, dstKey, dropReasonGoneDisconnected)
			dst.debugLogf("sendPkt attempt %d dropped, dst gone", attempt)
			return nil
		default:
		}
		select {
		case sendQueue <- p:
			dst.debugLogf("sendPkt attempt %d enqueued", attempt)
			return nil
		default:
		}

		select {
		case pkt := <-sendQueue:
			s.recordDrop(pkt.bs, c.key, dstKey, dropReasonQueueHead)
			c.recordQueueTime(pkt.enqueuedAt)
		default:
		}
	}
	// Failed to make room for packet. This can happen in a heavily
	// contended queue with racing writers. Give up and tail-drop in
	// this case to keep reader unblocked.
	s.recordDrop(p.bs, c.key, dstKey, dropReasonQueueTail)
	dst.debugLogf("sendPkt attempt %d dropped, queue full")

	return nil
}

// requestPeerGoneWrite sends a request to write a "peer gone" frame
// with an explanation of why it is gone. It blocks until either the
// write request is scheduled, or the client has closed.
func (c *sclient) requestPeerGoneWrite(peer key.NodePublic, reason PeerGoneReasonType) {
	select {
	case c.peerGone <- peerGoneMsg{
		peer:   peer,
		reason: reason,
	}:
	case <-c.done:
	}
}

func (c *sclient) requestMeshUpdate() {
	if !c.canMesh {
		panic("unexpected requestMeshUpdate")
	}
	select {
	case c.meshUpdate <- struct{}{}:
	case <-c.done:
	}
}

func (s *Server) verifyClient(clientKey key.NodePublic, info *clientInfo) error {
	if !s.verifyClients {
		return nil
	}
	status, err := tailscale.Status(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to query local tailscaled status: %w", err)
	}
	if clientKey == status.Self.PublicKey {
		return nil
	}
	if _, exists := status.Peer[clientKey]; !exists {
		return fmt.Errorf("client %v not in set of peers", clientKey)
	}
	// TODO(bradfitz): add policy for configurable bandwidth rate per client?
	return nil
}

func (s *Server) sendServerKey(lw *lazyBufioWriter) error {
	buf := make([]byte, 0, len(magic)+key.NodePublicRawLen)
	buf = append(buf, magic...)
	buf = s.publicKey.AppendTo(buf)
	err := writeFrame(lw.bw(), frameServerKey, buf)
	lw.Flush() // redundant (no-op) flush to release bufio.Writer
	return err
}

func (s *Server) noteClientActivity(c *sclient) {
	if !c.isDup.Load() {
		// Fast path for clients that aren't in a dup set.
		return
	}
	if c.isDisabled.Load() {
		// If they're already disabled, no point checking more.
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	ds, ok := s.clients[c.key].(*dupClientSet)
	if !ok {
		// It became unduped in between the isDup fast path check above
		// and the mutex check. Nothing to do.
		return
	}

	if s.dupPolicy == lastWriterIsActive {
		ds.last = c
	} else if ds.last == nil {
		// If we didn't have a primary, let the current
		// speaker be the primary.
		ds.last = c
	}

	if sh := ds.sendHistory; len(sh) != 0 && sh[len(sh)-1] == c {
		// The client c was the last client to make activity
		// in this set and it was already recorded. Nothing to
		// do.
		return
	}

	// If we saw this connection send previously, then consider
	// the group fighting and disable them all.
	if s.dupPolicy == disableFighters {
		for _, prior := range ds.sendHistory {
			if prior == c {
				ds.ForeachClient(func(c *sclient) {
					c.isDisabled.Store(true)
				})
				break
			}
		}
	}

	// Append this client to the list of clients who spoke last.
	ds.sendHistory = append(ds.sendHistory, c)
}

type serverInfo struct {
	Version int `json:"version,omitempty"`

	TokenBucketBytesPerSecond int `json:",omitempty"`
	TokenBucketBytesBurst     int `json:",omitempty"`
}

func (s *Server) sendServerInfo(bw *lazyBufioWriter, clientKey key.NodePublic) error {
	msg, err := json.Marshal(serverInfo{Version: ProtocolVersion})
	if err != nil {
		return err
	}

	msgbox := s.privateKey.SealTo(clientKey, msg)
	if err := writeFrameHeader(bw.bw(), frameServerInfo, uint32(len(msgbox))); err != nil {
		return err
	}
	if _, err := bw.Write(msgbox); err != nil {
		return err
	}
	return bw.Flush()
}

// recvClientKey reads the frameClientInfo frame from the client (its
// proof of identity) upon its initial connection. It should be
// considered especially untrusted at this point.
func (s *Server) recvClientKey(br *bufio.Reader) (clientKey key.NodePublic, info *clientInfo, err error) {
	fl, err := readFrameTypeHeader(br, frameClientInfo)
	if err != nil {
		return zpub, nil, err
	}
	const minLen = keyLen + nonceLen
	if fl < minLen {
		return zpub, nil, errors.New("short client info")
	}
	// We don't trust the client at all yet, so limit its input size to limit
	// things like JSON resource exhausting (http://github.com/golang/go/issues/31789).
	if fl > 256<<10 {
		return zpub, nil, errors.New("long client info")
	}
	if err := clientKey.ReadRawWithoutAllocating(br); err != nil {
		return zpub, nil, err
	}
	msgLen := int(fl - keyLen)
	msgbox := make([]byte, msgLen)
	if _, err := io.ReadFull(br, msgbox); err != nil {
		return zpub, nil, fmt.Errorf("msgbox: %v", err)
	}
	msg, ok := s.privateKey.OpenFrom(clientKey, msgbox)
	if !ok {
		return zpub, nil, fmt.Errorf("msgbox: cannot open len=%d with client key %s", msgLen, clientKey)
	}
	info = new(clientInfo)
	if err := json.Unmarshal(msg, info); err != nil {
		return zpub, nil, fmt.Errorf("msg: %v", err)
	}
	return clientKey, info, nil
}

func (s *Server) recvPacket(br *bufio.Reader, frameLen uint32) (dstKey key.NodePublic, contents []byte, err error) {
	if frameLen < keyLen {
		return zpub, nil, errors.New("short send packet frame")
	}
	if err := dstKey.ReadRawWithoutAllocating(br); err != nil {
		return zpub, nil, err
	}
	packetLen := frameLen - keyLen
	if packetLen > MaxPacketSize {
		return zpub, nil, fmt.Errorf("data packet longer (%d) than max of %v", packetLen, MaxPacketSize)
	}
	contents = make([]byte, packetLen)
	if _, err := io.ReadFull(br, contents); err != nil {
		return zpub, nil, err
	}
	s.packetsRecv.Add(1)
	s.bytesRecv.Add(int64(len(contents)))
	if disco.LooksLikeDiscoWrapper(contents) {
		s.packetsRecvDisco.Add(1)
	} else {
		s.packetsRecvOther.Add(1)
	}
	return dstKey, contents, nil
}

// zpub is the key.NodePublic zero value.
var zpub key.NodePublic

func (s *Server) recvForwardPacket(br *bufio.Reader, frameLen uint32) (srcKey, dstKey key.NodePublic, contents []byte, err error) {
	if frameLen < keyLen*2 {
		return zpub, zpub, nil, errors.New("short send packet frame")
	}
	if err := srcKey.ReadRawWithoutAllocating(br); err != nil {
		return zpub, zpub, nil, err
	}
	if err := dstKey.ReadRawWithoutAllocating(br); err != nil {
		return zpub, zpub, nil, err
	}
	packetLen := frameLen - keyLen*2
	if packetLen > MaxPacketSize {
		return zpub, zpub, nil, fmt.Errorf("data packet longer (%d) than max of %v", packetLen, MaxPacketSize)
	}
	contents = make([]byte, packetLen)
	if _, err := io.ReadFull(br, contents); err != nil {
		return zpub, zpub, nil, err
	}
	// TODO: was s.packetsRecv.Add(1)
	// TODO: was s.bytesRecv.Add(int64(len(contents)))
	return srcKey, dstKey, contents, nil
}

// sclient is a client connection to the server.
//
// (The "s" prefix is to more explicitly distinguish it from Client in derp_client.go)
type sclient struct {
	// Static after construction.
	connNum        int64 // process-wide unique counter, incremented each Accept
	s              *Server
	nc             Conn
	key            key.NodePublic
	info           clientInfo
	logf           logger.Logf
	done           <-chan struct{}  // closed when connection closes
	remoteAddr     string           // usually ip:port from net.Conn.RemoteAddr().String()
	remoteIPPort   netip.AddrPort   // zero if remoteAddr is not ip:port.
	sendQueue      chan pkt         // packets queued to this client; never closed
	discoSendQueue chan pkt         // important packets queued to this client; never closed
	sendPongCh     chan [8]byte     // pong replies to send to the client; never closed
	peerGone       chan peerGoneMsg // write request that a peer is not at this server (not used by mesh peers)
	meshUpdate     chan struct{}    // write request to write peerStateChange
	canMesh        bool             // clientInfo had correct mesh token for inter-region routing
	isDup          atomic.Bool      // whether more than 1 sclient for key is connected
	isDisabled     atomic.Bool      // whether sends to this peer are disabled due to active/active dups
	debug          bool             // turn on for verbose logging

	// Owned by run, not thread-safe.
	br          *bufio.Reader
	connectedAt time.Time
	preferred   bool

	// Owned by sender, not thread-safe.
	bw *lazyBufioWriter

	// Guarded by s.mu
	//
	// peerStateChange is used by mesh peers (a set of regional
	// DERP servers) and contains records that need to be sent to
	// the client for them to update their map of who's connected
	// to this node.
	peerStateChange []peerConnState

	// peerGoneLimiter limits how often the server will inform a
	// client that it's trying to establish a direct connection
	// through us with a peer we have no record of.
	peerGoneLim *rate.Limiter
}

// peerConnState represents whether a peer is connected to the server
// or not.
type peerConnState struct {
	peer    key.NodePublic
	present bool
	ipPort  netip.AddrPort // if present, the peer's IP:port
}

// pkt is a request to write a data frame to an sclient.
type pkt struct {
	// src is the who's the sender of the packet.
	src key.NodePublic

	// enqueuedAt is when a packet was put onto a queue before it was sent,
	// and is used for reporting metrics on the duration of packets in the queue.
	enqueuedAt time.Time

	// bs is the data packet bytes.
	// The memory is owned by pkt.
	bs []byte
}

// peerGoneMsg is a request to write a peerGone frame to an sclient
type peerGoneMsg struct {
	peer   key.NodePublic
	reason PeerGoneReasonType
}

func (c *sclient) setPreferred(v bool) {
	if c.preferred == v {
		return
	}
	c.preferred = v
	var homeMove *expvar.Int
	if v {
		c.s.curHomeClients.Add(1)
		homeMove = &c.s.homeMovesIn
	} else {
		c.s.curHomeClients.Add(-1)
		homeMove = &c.s.homeMovesOut
	}

	// Keep track of varz for home serve moves in/out.  But ignore
	// the initial packet set when a client connects, which we
	// assume happens within 5 seconds. In any case, just for
	// graphs, so not important to miss a move. But it shouldn't:
	// the netcheck/re-STUNs in magicsock only happen about every
	// 30 seconds.
	if c.s.clock.Since(c.connectedAt) > 5*time.Second {
		homeMove.Add(1)
	}
}

// expMovingAverage returns the new moving average given the previous average,
// a new value, and an alpha decay factor.
// https://en.wikipedia.org/wiki/Moving_average#Exponential_moving_average
func expMovingAverage(prev, newValue, alpha float64) float64 {
	return alpha*newValue + (1-alpha)*prev
}

// recordQueueTime updates the average queue duration metric after a packet has been sent.
func (c *sclient) recordQueueTime(enqueuedAt time.Time) {
	elapsed := float64(c.s.clock.Since(enqueuedAt).Milliseconds())
	for {
		old := atomic.LoadUint64(c.s.avgQueueDuration)
		newAvg := expMovingAverage(math.Float64frombits(old), elapsed, 0.1)
		if atomic.CompareAndSwapUint64(c.s.avgQueueDuration, old, math.Float64bits(newAvg)) {
			break
		}
	}
}

func (c *sclient) sendLoop(ctx context.Context) error {
	defer func() {
		// If the sender shuts down unilaterally due to an error, close so
		// that the receive loop unblocks and cleans up the rest.
		c.nc.Close()

		// Drain the send queue to count dropped packets
		for {
			select {
			case pkt := <-c.sendQueue:
				c.s.recordDrop(pkt.bs, pkt.src, c.key, dropReasonGoneDisconnected)
			case pkt := <-c.discoSendQueue:
				c.s.recordDrop(pkt.bs, pkt.src, c.key, dropReasonGoneDisconnected)
			default:
				return
			}
		}
	}()

	jitter := time.Duration(rand.Intn(5000)) * time.Millisecond
	keepAliveTick, keepAliveTickChannel := c.s.clock.NewTicker(keepAlive + jitter)
	defer keepAliveTick.Stop()

	var werr error // last write error
	for {
		if werr != nil {
			return werr
		}
		// First, a non-blocking select (with a default) that
		// does as many non-flushing writes as possible.
		select {
		case <-ctx.Done():
			return nil
		case msg := <-c.peerGone:
			werr = c.sendPeerGone(msg.peer, msg.reason)
			continue
		case <-c.meshUpdate:
			werr = c.sendMeshUpdates()
			continue
		case msg := <-c.sendQueue:
			werr = c.sendPacket(msg.src, msg.bs)
			c.recordQueueTime(msg.enqueuedAt)
			continue
		case msg := <-c.discoSendQueue:
			werr = c.sendPacket(msg.src, msg.bs)
			c.recordQueueTime(msg.enqueuedAt)
			continue
		case msg := <-c.sendPongCh:
			werr = c.sendPong(msg)
			continue
		case <-keepAliveTickChannel:
			werr = c.sendKeepAlive()
			continue
		default:
			// Flush any writes from the 3 sends above, or from
			// the blocking loop below.
			if werr = c.bw.Flush(); werr != nil {
				return werr
			}
		}

		// Then a blocking select with same:
		select {
		case <-ctx.Done():
			return nil
		case msg := <-c.peerGone:
			werr = c.sendPeerGone(msg.peer, msg.reason)
		case <-c.meshUpdate:
			werr = c.sendMeshUpdates()
			continue
		case msg := <-c.sendQueue:
			werr = c.sendPacket(msg.src, msg.bs)
			c.recordQueueTime(msg.enqueuedAt)
		case msg := <-c.discoSendQueue:
			werr = c.sendPacket(msg.src, msg.bs)
			c.recordQueueTime(msg.enqueuedAt)
		case msg := <-c.sendPongCh:
			werr = c.sendPong(msg)
			continue
		case <-keepAliveTickChannel:
			werr = c.sendKeepAlive()
		}
	}
}

func (c *sclient) setWriteDeadline() {
	c.nc.SetWriteDeadline(time.Now().Add(writeTimeout))
}

// sendKeepAlive sends a keep-alive frame, without flushing.
func (c *sclient) sendKeepAlive() error {
	c.setWriteDeadline()
	return writeFrameHeader(c.bw.bw(), frameKeepAlive, 0)
}

// sendPong sends a pong reply, without flushing.
func (c *sclient) sendPong(data [8]byte) error {
	c.s.sentPong.Add(1)
	c.setWriteDeadline()
	if err := writeFrameHeader(c.bw.bw(), framePong, uint32(len(data))); err != nil {
		return err
	}
	_, err := c.bw.Write(data[:])
	return err
}

// sendPeerGone sends a peerGone frame, without flushing.
func (c *sclient) sendPeerGone(peer key.NodePublic, reason PeerGoneReasonType) error {
	switch reason {
	case PeerGoneReasonDisconnected:
		c.s.peerGoneDisconnectedFrames.Add(1)
	case PeerGoneReasonNotHere:
		c.s.peerGoneNotHereFrames.Add(1)
	}
	c.setWriteDeadline()
	data := make([]byte, 0, keyLen+1)
	data = peer.AppendTo(data)
	data = append(data, byte(reason))
	if err := writeFrameHeader(c.bw.bw(), framePeerGone, uint32(len(data))); err != nil {
		return err
	}

	_, err := c.bw.Write(data)
	return err
}

// sendPeerPresent sends a peerPresent frame, without flushing.
func (c *sclient) sendPeerPresent(peer key.NodePublic, ipPort netip.AddrPort) error {
	c.setWriteDeadline()
	const frameLen = keyLen + 16 + 2
	if err := writeFrameHeader(c.bw.bw(), framePeerPresent, frameLen); err != nil {
		return err
	}
	payload := make([]byte, frameLen)
	_ = peer.AppendTo(payload[:0])
	a16 := ipPort.Addr().As16()
	copy(payload[keyLen:], a16[:])
	binary.BigEndian.PutUint16(payload[keyLen+16:], ipPort.Port())
	_, err := c.bw.Write(payload)
	return err
}

// sendMeshUpdates drains as many mesh peerStateChange entries as
// possible into the write buffer WITHOUT flushing or otherwise
// blocking (as it holds c.s.mu while working). If it can't drain them
// all, it schedules itself to be called again in the future.
func (c *sclient) sendMeshUpdates() error {
	c.s.mu.Lock()
	defer c.s.mu.Unlock()

	writes := 0
	for _, pcs := range c.peerStateChange {
		if c.bw.Available() <= frameHeaderLen+keyLen {
			break
		}
		var err error
		if pcs.present {
			err = c.sendPeerPresent(pcs.peer, pcs.ipPort)
		} else {
			err = c.sendPeerGone(pcs.peer, PeerGoneReasonDisconnected)
		}
		if err != nil {
			// Shouldn't happen, though, as we're writing
			// into available buffer space, not the
			// network.
			return err
		}
		writes++
	}

	remain := copy(c.peerStateChange, c.peerStateChange[writes:])
	c.peerStateChange = c.peerStateChange[:remain]

	// Did we manage to write them all into the bufio buffer without flushing?
	if len(c.peerStateChange) == 0 {
		if cap(c.peerStateChange) > 16 {
			c.peerStateChange = nil
		}
	} else {
		// Didn't finish in the buffer space provided; schedule a future run.
		go c.requestMeshUpdate()
	}
	return nil
}

// sendPacket writes contents to the client in a RecvPacket frame. If
// srcKey.IsZero, uses the old DERPv1 framing format, otherwise uses
// DERPv2. The bytes of contents are only valid until this function
// returns, do not retain slices.
// It does not flush its bufio.Writer.
func (c *sclient) sendPacket(srcKey key.NodePublic, contents []byte) (err error) {
	defer func() {
		// Stats update.
		if err != nil {
			c.s.recordDrop(contents, srcKey, c.key, dropReasonWriteError)
		} else {
			c.s.packetsSent.Add(1)
			c.s.bytesSent.Add(int64(len(contents)))
		}
		c.debugLogf("sendPacket from %s: %v", srcKey.ShortString(), err)
	}()

	c.setWriteDeadline()

	withKey := !srcKey.IsZero()
	pktLen := len(contents)
	if withKey {
		pktLen += key.NodePublicRawLen
	}
	if err = writeFrameHeader(c.bw.bw(), frameRecvPacket, uint32(pktLen)); err != nil {
		return err
	}
	if withKey {
		if err := srcKey.WriteRawWithoutAllocating(c.bw.bw()); err != nil {
			return err
		}
	}
	_, err = c.bw.Write(contents)
	return err
}

// AddPacketForwarder registers fwd as a packet forwarder for dst.
// fwd must be comparable.
func (s *Server) AddPacketForwarder(dst key.NodePublic, fwd PacketForwarder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if prev, ok := s.clientsMesh[dst]; ok {
		if prev == fwd {
			// Duplicate registration of same forwarder. Ignore.
			return
		}
		if m, ok := prev.(*multiForwarder); ok {
			if _, ok := m.all[fwd]; ok {
				// Duplicate registration of same forwarder in set; ignore.
				return
			}
			m.add(fwd)
			return
		}
		if prev != nil {
			// Otherwise, the existing value is not a set,
			// not a dup, and not local-only (nil) so make
			// it a set. `prev` existed first, so will have higher
			// priority.
			fwd = newMultiForwarder(prev, fwd)
			s.multiForwarderCreated.Add(1)
		}
	}
	s.clientsMesh[dst] = fwd
}

// RemovePacketForwarder removes fwd as a packet forwarder for dst.
// fwd must be comparable.
func (s *Server) RemovePacketForwarder(dst key.NodePublic, fwd PacketForwarder) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.clientsMesh[dst]
	if !ok {
		return
	}
	if m, ok := v.(*multiForwarder); ok {
		if len(m.all) < 2 {
			panic("unexpected")
		}
		if remain, isLast := m.deleteLocked(fwd); isLast {
			// If fwd was in m and we no longer need to be a
			// multiForwarder, replace the entry with the
			// remaining PacketForwarder.
			s.clientsMesh[dst] = remain
			s.multiForwarderDeleted.Add(1)
		}
		return
	}
	if v != fwd {
		s.removePktForwardOther.Add(1)
		// Delete of an entry that wasn't in the
		// map. Harmless, so ignore.
		// (This might happen if a user is moving around
		// between nodes and/or the server sent duplicate
		// connection change broadcasts.)
		return
	}

	if _, isLocal := s.clients[dst]; isLocal {
		s.clientsMesh[dst] = nil
	} else {
		delete(s.clientsMesh, dst)
		s.notePeerGoneFromRegionLocked(dst)
	}
}

// multiForwarder is a PacketForwarder that represents a set of
// forwarding options. It's used in the rare cases that a client is
// connected to multiple DERP nodes in a region. That shouldn't really
// happen except for perhaps during brief moments while the client is
// reconfiguring, in which case we don't want to forget where the
// client is. The map value is unique connection number; the lowest
// one has been seen the longest. It's used to make sure we forward
// packets consistently to the same node and don't pick randomly.
type multiForwarder struct {
	fwd syncs.AtomicValue[PacketForwarder] // preferred forwarder.
	all map[PacketForwarder]uint8          // all forwarders, protected by s.mu.
}

// newMultiForwarder creates a new multiForwarder.
// The first PacketForwarder passed to this function will be the preferred one.
func newMultiForwarder(fwds ...PacketForwarder) *multiForwarder {
	f := &multiForwarder{all: make(map[PacketForwarder]uint8)}
	f.fwd.Store(fwds[0])
	for idx, fwd := range fwds {
		f.all[fwd] = uint8(idx)
	}
	return f
}

// add adds a new forwarder to the map with a connection number that
// is higher than the existing ones.
func (f *multiForwarder) add(fwd PacketForwarder) {
	var max uint8
	for _, v := range f.all {
		if v > max {
			max = v
		}
	}
	f.all[fwd] = max + 1
}

// deleteLocked removes a packet forwarder from the map. It expects Server.mu to be held.
// If only one forwarder remains after the removal, it will be returned alongside a `true` boolean value.
func (f *multiForwarder) deleteLocked(fwd PacketForwarder) (_ PacketForwarder, isLast bool) {
	delete(f.all, fwd)

	if fwd == f.fwd.Load() {
		// The preferred forwarder has been removed, choose a new one
		// based on the lowest index.
		var lowestfwd PacketForwarder
		var lowest uint8
		for k, v := range f.all {
			if lowestfwd == nil || v < lowest {
				lowestfwd = k
				lowest = v
			}
		}
		if lowestfwd != nil {
			f.fwd.Store(lowestfwd)
		}
	}

	if len(f.all) == 1 {
		for k := range f.all {
			return k, true
		}
	}
	return nil, false
}

func (f *multiForwarder) ForwardPacket(src, dst key.NodePublic, payload []byte) error {
	return f.fwd.Load().ForwardPacket(src, dst, payload)
}

func (f *multiForwarder) String() string {
	return fmt.Sprintf("<MultiForwarder fwd=%s total=%d>", f.fwd.Load(), len(f.all))
}

func (s *Server) expVarFunc(f func() any) expvar.Func {
	return expvar.Func(func() any {
		s.mu.Lock()
		defer s.mu.Unlock()
		return f()
	})
}

// ExpVar returns an expvar variable suitable for registering with expvar.Publish.
func (s *Server) ExpVar() expvar.Var {
	m := new(metrics.Set)
	m.Set("gauge_memstats_sys0", expvar.Func(func() any { return int64(s.memSys0) }))
	m.Set("gauge_watchers", s.expVarFunc(func() any { return len(s.watchers) }))
	m.Set("gauge_current_file_descriptors", expvar.Func(func() any { return metrics.CurrentFDs() }))
	m.Set("gauge_current_connections", &s.curClients)
	m.Set("gauge_current_home_connections", &s.curHomeClients)
	m.Set("gauge_clients_total", expvar.Func(func() any { return len(s.clientsMesh) }))
	m.Set("gauge_clients_local", expvar.Func(func() any { return len(s.clients) }))
	m.Set("gauge_clients_remote", expvar.Func(func() any { return len(s.clientsMesh) - len(s.clients) }))
	m.Set("gauge_current_dup_client_keys", &s.dupClientKeys)
	m.Set("gauge_current_dup_client_conns", &s.dupClientConns)
	m.Set("counter_total_dup_client_conns", &s.dupClientConnTotal)
	m.Set("accepts", &s.accepts)
	m.Set("bytes_received", &s.bytesRecv)
	m.Set("bytes_sent", &s.bytesSent)
	m.Set("packets_dropped", &s.packetsDropped)
	m.Set("counter_packets_dropped_reason", &s.packetsDroppedReason)
	m.Set("counter_packets_dropped_type", &s.packetsDroppedType)
	m.Set("counter_packets_received_kind", &s.packetsRecvByKind)
	m.Set("packets_sent", &s.packetsSent)
	m.Set("packets_received", &s.packetsRecv)
	m.Set("unknown_frames", &s.unknownFrames)
	m.Set("home_moves_in", &s.homeMovesIn)
	m.Set("home_moves_out", &s.homeMovesOut)
	m.Set("got_ping", &s.gotPing)
	m.Set("sent_pong", &s.sentPong)
	m.Set("peer_gone_disconnected_frames", &s.peerGoneDisconnectedFrames)
	m.Set("peer_gone_not_here_frames", &s.peerGoneNotHereFrames)
	m.Set("packets_forwarded_out", &s.packetsForwardedOut)
	m.Set("packets_forwarded_in", &s.packetsForwardedIn)
	m.Set("multiforwarder_created", &s.multiForwarderCreated)
	m.Set("multiforwarder_deleted", &s.multiForwarderDeleted)
	m.Set("packet_forwarder_delete_other_value", &s.removePktForwardOther)
	m.Set("average_queue_duration_ms", expvar.Func(func() any {
		return math.Float64frombits(atomic.LoadUint64(s.avgQueueDuration))
	}))
	m.Set("counter_tcp_rtt", &s.tcpRtt)
	var expvarVersion expvar.String
	expvarVersion.Set(version.Long())
	m.Set("version", &expvarVersion)
	return m
}

func (s *Server) ConsistencyCheck() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []string

	var nilMeshNotInClient int
	for k, f := range s.clientsMesh {
		if f == nil {
			if _, ok := s.clients[k]; !ok {
				nilMeshNotInClient++
			}
		}
	}
	if nilMeshNotInClient != 0 {
		errs = append(errs, fmt.Sprintf("%d s.clientsMesh keys not in s.clients", nilMeshNotInClient))
	}

	var clientNotInMesh int
	for k := range s.clients {
		if _, ok := s.clientsMesh[k]; !ok {
			clientNotInMesh++
		}
	}
	if clientNotInMesh != 0 {
		errs = append(errs, fmt.Sprintf("%d s.clients keys not in s.clientsMesh", clientNotInMesh))
	}

	if s.curClients.Value() != int64(len(s.clients)) {
		errs = append(errs, fmt.Sprintf("expvar connections = %d != clients map says of %d",
			s.curClients.Value(),
			len(s.clients)))
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, ", "))
}

const minTimeBetweenLogs = 2 * time.Second

// BytesSentRecv records the number of bytes that have been sent since the last traffic check
// for a given process, as well as the public key of the process sending those bytes.
type BytesSentRecv struct {
	Sent uint64
	Recv uint64
	// Key is the public key of the client which sent/received these bytes.
	Key key.NodePublic
}

// parseSSOutput parses the output from the specific call to ss in ServeDebugTraffic.
// Separated out for ease of testing.
func parseSSOutput(raw string) map[netip.AddrPort]BytesSentRecv {
	newState := map[netip.AddrPort]BytesSentRecv{}
	// parse every 2 lines and get src and dst ips, and kv pairs
	lines := strings.Split(raw, "\n")
	for i := 0; i < len(lines); i += 2 {
		ipInfo := strings.Fields(strings.TrimSpace(lines[i]))
		if len(ipInfo) < 5 {
			continue
		}
		src, err := netip.ParseAddrPort(ipInfo[4])
		if err != nil {
			continue
		}
		stats := strings.Fields(strings.TrimSpace(lines[i+1]))
		stat := BytesSentRecv{}
		for _, s := range stats {
			if strings.Contains(s, "bytes_sent") {
				sent, err := strconv.Atoi(s[strings.Index(s, ":")+1:])
				if err == nil {
					stat.Sent = uint64(sent)
				}
			} else if strings.Contains(s, "bytes_received") {
				recv, err := strconv.Atoi(s[strings.Index(s, ":")+1:])
				if err == nil {
					stat.Recv = uint64(recv)
				}
			}
		}
		newState[src] = stat
	}
	return newState
}

func (s *Server) ServeDebugTraffic(w http.ResponseWriter, r *http.Request) {
	prevState := map[netip.AddrPort]BytesSentRecv{}
	enc := json.NewEncoder(w)
	for r.Context().Err() == nil {
		output, err := exec.Command("ss", "-i", "-H", "-t").Output()
		if err != nil {
			fmt.Fprintf(w, "ss failed: %v", err)
			return
		}
		newState := parseSSOutput(string(output))
		s.mu.Lock()
		for k, next := range newState {
			prev := prevState[k]
			if prev.Sent < next.Sent || prev.Recv < next.Recv {
				if pkey, ok := s.keyOfAddr[k]; ok {
					next.Key = pkey
					if err := enc.Encode(next); err != nil {
						s.mu.Unlock()
						return
					}
				}
			}
		}
		s.mu.Unlock()
		prevState = newState
		if _, err := fmt.Fprintln(w); err != nil {
			return
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(minTimeBetweenLogs)
	}
}

var bufioWriterPool = &sync.Pool{
	New: func() any {
		return bufio.NewWriterSize(io.Discard, 2<<10)
	},
}

// lazyBufioWriter is a bufio.Writer-like wrapping writer that lazily
// allocates its actual bufio.Writer from a sync.Pool, releasing it to
// the pool upon flush.
//
// We do this to reduce memory overhead; most DERP connections are
// idle and the idle bufio.Writers were 30% of overall memory usage.
type lazyBufioWriter struct {
	w   io.Writer     // underlying
	lbw *bufio.Writer // lazy; nil means it needs an associated buffer
}

func (w *lazyBufioWriter) bw() *bufio.Writer {
	if w.lbw == nil {
		w.lbw = bufioWriterPool.Get().(*bufio.Writer)
		w.lbw.Reset(w.w)
	}
	return w.lbw
}

func (w *lazyBufioWriter) Available() int { return w.bw().Available() }

func (w *lazyBufioWriter) Write(p []byte) (int, error) { return w.bw().Write(p) }

func (w *lazyBufioWriter) Flush() error {
	if w.lbw == nil {
		return nil
	}
	err := w.lbw.Flush()

	w.lbw.Reset(io.Discard)
	bufioWriterPool.Put(w.lbw)
	w.lbw = nil

	return err
}
