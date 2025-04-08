// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package netmon

import (
	"fmt"
	"log"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
	"tailscale.com/tsconst"
)

const (
	fallbackInterfaceMetric = uint32(0)       // Used if we cannot get the actual interface metric
	LOFT_ADMIN_HOST         = "admin.loft.sh" // Same host as used in Darwin/Linux implementation
)

func init() {
	likelyHomeRouterIP = likelyHomeRouterIPWindows
	getPAC = getPACWindows
}

func likelyHomeRouterIPWindows() (ret netip.Addr, _ netip.Addr, ok bool) {
	rs, err := winipcfg.GetIPForwardTable2(windows.AF_INET)
	if err != nil {
		log.Printf("routerIP/GetIPForwardTable2 error: %v", err)
		return
	}

	var ifaceMetricCache map[winipcfg.LUID]uint32

	getIfaceMetric := func(luid winipcfg.LUID) (metric uint32) {
		if ifaceMetricCache == nil {
			ifaceMetricCache = make(map[winipcfg.LUID]uint32)
		} else if m, ok := ifaceMetricCache[luid]; ok {
			return m
		}

		if iface, err := luid.IPInterface(windows.AF_INET); err == nil {
			metric = iface.Metric
		} else {
			log.Printf("routerIP/luid.IPInterface error: %v", err)
			metric = fallbackInterfaceMetric
		}

		ifaceMetricCache[luid] = metric
		return
	}

	v4unspec := netip.IPv4Unspecified()
	var best *winipcfg.MibIPforwardRow2 // best (lowest metric) found so far, or nil

	for i := range rs {
		r := &rs[i]
		if r.Loopback || r.DestinationPrefix.PrefixLength != 0 || r.DestinationPrefix.Prefix().Addr().Unmap() != v4unspec {
			// Not a default route, so skip
			continue
		}

		ip := r.NextHop.Addr().Unmap()
		if !ip.IsValid() {
			// Not a valid gateway, so skip (won't happen though)
			continue
		}

		if best == nil {
			best = r
			ret = ip
			continue
		}

		// We can get here only if there are multiple default gateways defined (rare case),
		// in which case we need to calculate the effective metric.
		// Effective metric is sum of interface metric and route metric offset
		if ifaceMetricCache == nil {
			// If we're here it means that previous route still isn't updated, so update it
			best.Metric += getIfaceMetric(best.InterfaceLUID)
		}
		r.Metric += getIfaceMetric(r.InterfaceLUID)

		if best.Metric > r.Metric || best.Metric == r.Metric && ret.Compare(ip) > 0 {
			// Pick the route with lower metric, or lower IP if metrics are equal
			best = r
			ret = ip
		}
	}

	if ret.IsValid() && !ret.IsPrivate() {
		// Default route has a non-private gateway
		return netip.Addr{}, netip.Addr{}, false
	}

	return ret, netip.Addr{}, ret.IsValid()
}

// NonTailscaleMTUs returns a map of interface LUID to interface MTU,
// for all interfaces except Tailscale tunnels.
func NonTailscaleMTUs() (map[winipcfg.LUID]uint32, error) {
	mtus := map[winipcfg.LUID]uint32{}
	ifs, err := NonTailscaleInterfaces()
	for luid, iface := range ifs {
		mtus[luid] = iface.MTU
	}
	return mtus, err
}

func notTailscaleInterface(iface *winipcfg.IPAdapterAddresses) bool {
	// TODO(bradfitz): do this without the Description method's
	// utf16-to-string allocation. But at least we only do it for
	// the virtual interfaces, for which there won't be many.
	if iface.IfType != winipcfg.IfTypePropVirtual {
		return true
	}
	desc := iface.Description()
	return !(strings.Contains(desc, tsconst.WintunInterfaceDesc) ||
		strings.Contains(desc, tsconst.WintunInterfaceDesc0_14))
}

// NonTailscaleInterfaces returns a map of interface LUID to interface
// for all interfaces except Tailscale tunnels.
func NonTailscaleInterfaces() (map[winipcfg.LUID]*winipcfg.IPAdapterAddresses, error) {
	return getInterfaces(windows.AF_UNSPEC, winipcfg.GAAFlagIncludeAllInterfaces, notTailscaleInterface)
}

// getInterfaces returns a map of interfaces keyed by their LUID for
// all interfaces matching the provided match predicate.
//
// The family (AF_UNSPEC, AF_INET, or AF_INET6) and flags are passed
// to winipcfg.GetAdaptersAddresses.
func getInterfaces(family winipcfg.AddressFamily, flags winipcfg.GAAFlags, match func(*winipcfg.IPAdapterAddresses) bool) (map[winipcfg.LUID]*winipcfg.IPAdapterAddresses, error) {
	ifs, err := winipcfg.GetAdaptersAddresses(family, flags)
	if err != nil {
		return nil, err
	}
	ret := map[winipcfg.LUID]*winipcfg.IPAdapterAddresses{}
	for _, iface := range ifs {
		if match(iface) {
			ret[iface.LUID] = iface
		}
	}
	return ret, nil
}

// GetWindowsDefault returns the interface that has the non-Tailscale
// default route for the given address family.
//
// It returns (nil, nil) if no interface is found.
//
// The family must be one of AF_INET or AF_INET6.
func GetWindowsDefault(family winipcfg.AddressFamily) (*winipcfg.IPAdapterAddresses, error) {
	// let's try to resolve the interface that's used to contact loft first
	// First try to get interface by pinging remote endpoint
	if iface, err := getInterfaceForHost(LOFT_ADMIN_HOST); err == nil {
		return iface, nil
	}
	// then fall back to the default implementation if it doesn't work
	ifs, err := getInterfaces(family, winipcfg.GAAFlagIncludeAllInterfaces, func(iface *winipcfg.IPAdapterAddresses) bool {
		switch iface.IfType {
		case winipcfg.IfTypeSoftwareLoopback:
			return false
		}
		switch family {
		case windows.AF_INET:
			if iface.Flags&winipcfg.IPAAFlagIpv4Enabled == 0 {
				return false
			}
		case windows.AF_INET6:
			if iface.Flags&winipcfg.IPAAFlagIpv6Enabled == 0 {
				return false
			}
		}
		return iface.OperStatus == winipcfg.IfOperStatusUp && notTailscaleInterface(iface)
	})
	if err != nil {
		return nil, err
	}

	routes, err := winipcfg.GetIPForwardTable2(family)
	if err != nil {
		return nil, err
	}

	bestMetric := ^uint32(0)
	var bestIface *winipcfg.IPAdapterAddresses
	for _, route := range routes {
		if route.DestinationPrefix.PrefixLength != 0 {
			// Not a default route.
			continue
		}
		iface := ifs[route.InterfaceLUID]
		if iface == nil {
			continue
		}

		// Microsoft docs say:
		//
		// "The actual route metric used to compute the route
		// preferences for IPv4 is the summation of the route
		// metric offset specified in the Metric member of the
		// MIB_IPFORWARD_ROW2 structure and the interface
		// metric specified in this member for IPv4"
		metric := route.Metric
		switch family {
		case windows.AF_INET:
			metric += iface.Ipv4Metric
		case windows.AF_INET6:
			metric += iface.Ipv6Metric
		}
		if metric < bestMetric {
			bestMetric = metric
			bestIface = iface
		}
	}

	return bestIface, nil
}

func defaultRoute() (d DefaultRouteDetails, err error) {
	// We always return the IPv4 default route.
	// TODO(bradfitz): adjust API if/when anything cares. They could in theory differ, though,
	// in which case we might send traffic to the wrong interface.
	iface, err := GetWindowsDefault(windows.AF_INET)
	if err != nil {
		return d, err
	}
	if iface != nil {
		d.InterfaceName = iface.FriendlyName()
		d.InterfaceDesc = iface.Description()
		d.InterfaceIndex = int(iface.IfIndex)
	}
	return d, nil
}

var (
	winHTTP                  = windows.NewLazySystemDLL("winhttp.dll")
	detectAutoProxyConfigURL = winHTTP.NewProc("WinHttpDetectAutoProxyConfigUrl")

	kernel32   = windows.NewLazySystemDLL("kernel32.dll")
	globalFree = kernel32.NewProc("GlobalFree")
)

const (
	winHTTP_AUTO_DETECT_TYPE_DHCP  = 0x00000001
	winHTTP_AUTO_DETECT_TYPE_DNS_A = 0x00000002
)

func getPACWindows() string {
	var res *uint16
	r, _, e := detectAutoProxyConfigURL.Call(
		winHTTP_AUTO_DETECT_TYPE_DHCP|winHTTP_AUTO_DETECT_TYPE_DNS_A,
		uintptr(unsafe.Pointer(&res)),
	)
	if r == 1 {
		if res == nil {
			log.Printf("getPACWindows: unexpected success with nil result")
			return ""
		}
		defer globalFree.Call(uintptr(unsafe.Pointer(res)))
		s := windows.UTF16PtrToString(res)
		s = strings.TrimSpace(s)
		if s == "" {
			return "" // Issue 2357: invalid URL "\n" from winhttp; ignoring
		}
		if _, err := url.Parse(s); err != nil {
			log.Printf("getPACWindows: invalid URL %q from winhttp; ignoring", s)
			return ""
		}
		return s
	}
	const (
		ERROR_WINHTTP_AUTODETECTION_FAILED = 12180
	)
	if e == syscall.Errno(ERROR_WINHTTP_AUTODETECTION_FAILED) {
		// Common case on networks without advertised PAC.
		return ""
	}
	log.Printf("getPACWindows: %T=%v", e, e) // syscall.Errno=0x....
	return ""
}

// getInterfaceForHost gets default interface by checking route to specific host
func getInterfaceForHost(host string) (*winipcfg.IPAdapterAddresses, error) {
	ip, err := resolveHostname(host)
	if err != nil {
		return nil, err
	}

	// Get all interfaces
	ifs, err := getInterfaces(windows.AF_INET, winipcfg.GAAFlagIncludeAllInterfaces, func(iface *winipcfg.IPAdapterAddresses) bool {
		switch iface.IfType {
		case winipcfg.IfTypeSoftwareLoopback:
			return false
		}
		if iface.Flags&winipcfg.IPAAFlagIpv4Enabled == 0 {
			return false
		}
		return iface.OperStatus == winipcfg.IfOperStatusUp && notTailscaleInterface(iface)
	})
	if err != nil {
		return nil, err
	}

	// Get routing table
	routes, err := winipcfg.GetIPForwardTable2(windows.AF_INET)
	if err != nil {
		return nil, err
	}

	// Find the best route to reach the target IP
	var bestRoute *winipcfg.MibIPforwardRow2
	bestMetric := ^uint32(0)
	bestPrefixLength := uint8(0)

	for _, route := range routes {
		// Skip if interface not found or not up
		iface := ifs[route.InterfaceLUID]
		if iface == nil {
			continue
		}

		// Check if this route matches our destination
		if !ipMatchesRoute(ip, route) {
			continue
		}

		// compare routes by metric _and_ prefix length (specificity)
		// otherwise we would most likely always choose the default route
		metric := route.Metric + iface.Ipv4Metric
		if route.DestinationPrefix.PrefixLength > bestPrefixLength ||
			(route.DestinationPrefix.PrefixLength == bestPrefixLength && metric < bestMetric) {
			bestPrefixLength = route.DestinationPrefix.PrefixLength
			bestMetric = metric
			bestRoute = &route
		}
	}

	if bestRoute == nil {
		return nil, fmt.Errorf("no route found to %v", ip)
	}

	return ifs[bestRoute.InterfaceLUID], nil
}

// resolveHostname resolves net.IP of given hostname
func resolveHostname(hostname string) (net.IP, error) {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil, err
	}

	// Prefer IPv4 addresses
	for _, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4, nil
		}
	}

	return nil, fmt.Errorf("no IPv4 address found for %s", hostname)
}

// ipMatchesRoute checks if the given IP matches the route's destination/mask
func ipMatchesRoute(ip net.IP, route winipcfg.MibIPforwardRow2) bool {
	// Convert IP to uint32 for easier manipulation
	ipInt := ipToUint32(ip.To4())
	if ipInt == 0 {
		return false
	}

	// Get destination and mask from route
	destInt := ipToUint32(route.DestinationPrefix.Prefix().Addr().AsSlice())
	maskInt := ^uint32(0) << (32 - route.DestinationPrefix.PrefixLength)

	return (ipInt & maskInt) == (destInt & maskInt)
}

// ipToUint32 converts an IPv4 address to uint32
func ipToUint32(ip []byte) uint32 {
	if len(ip) != 4 {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}
