//go:build linux
// +build linux

package unix

import (
	linux "golang.org/x/sys/unix"
)

const (
	AF_INET                     = linux.AF_INET
	AF_INET6                    = linux.AF_INET6
	AF_UNSPEC                   = linux.AF_UNSPEC
	NETLINK_ROUTE               = linux.NETLINK_ROUTE
	SizeofIfAddrmsg             = linux.SizeofIfAddrmsg
	SizeofIfInfomsg             = linux.SizeofIfInfomsg
	SizeofNdMsg                 = linux.SizeofNdMsg
	SizeofRtMsg                 = linux.SizeofRtMsg
	SizeofRtNexthop             = linux.SizeofRtNexthop
	RTM_NEWADDR                 = linux.RTM_NEWADDR
	RTM_DELADDR                 = linux.RTM_DELADDR
	RTM_GETADDR                 = linux.RTM_GETADDR
	RTM_NEWLINK                 = linux.RTM_NEWLINK
	RTM_DELLINK                 = linux.RTM_DELLINK
	RTM_GETLINK                 = linux.RTM_GETLINK
	RTM_SETLINK                 = linux.RTM_SETLINK
	RTM_NEWROUTE                = linux.RTM_NEWROUTE
	RTM_DELROUTE                = linux.RTM_DELROUTE
	RTM_GETROUTE                = linux.RTM_GETROUTE
	RTM_NEWNEIGH                = linux.RTM_NEWNEIGH
	RTM_DELNEIGH                = linux.RTM_DELNEIGH
	RTM_GETNEIGH                = linux.RTM_GETNEIGH
	IFA_UNSPEC                  = linux.IFA_UNSPEC
	IFA_ADDRESS                 = linux.IFA_ADDRESS
	IFA_LOCAL                   = linux.IFA_LOCAL
	IFA_LABEL                   = linux.IFA_LABEL
	IFA_BROADCAST               = linux.IFA_BROADCAST
	IFA_ANYCAST                 = linux.IFA_ANYCAST
	IFA_CACHEINFO               = linux.IFA_CACHEINFO
	IFA_MULTICAST               = linux.IFA_MULTICAST
	IFA_FLAGS                   = linux.IFA_FLAGS
	IFF_UP                      = linux.IFF_UP
	IFF_BROADCAST               = linux.IFF_BROADCAST
	IFF_LOOPBACK                = linux.IFF_LOOPBACK
	IFF_POINTOPOINT             = linux.IFF_POINTOPOINT
	IFF_MULTICAST               = linux.IFF_MULTICAST
	IFLA_UNSPEC                 = linux.IFLA_UNSPEC
	IFLA_ADDRESS                = linux.IFLA_ADDRESS
	IFLA_BROADCAST              = linux.IFLA_BROADCAST
	IFLA_IFNAME                 = linux.IFLA_IFNAME
	IFLA_MTU                    = linux.IFLA_MTU
	IFLA_LINK                   = linux.IFLA_LINK
	IFLA_QDISC                  = linux.IFLA_QDISC
	IFLA_OPERSTATE              = linux.IFLA_OPERSTATE
	IFLA_STATS                  = linux.IFLA_STATS
	IFLA_STATS64                = linux.IFLA_STATS64
	IFLA_TXQLEN                 = linux.IFLA_TXQLEN
	IFLA_GROUP                  = linux.IFLA_GROUP
	IFLA_LINKINFO               = linux.IFLA_LINKINFO
	IFLA_LINKMODE               = linux.IFLA_LINKMODE
	IFLA_IFALIAS                = linux.IFLA_IFALIAS
	IFLA_MASTER                 = linux.IFLA_MASTER
	IFLA_CARRIER                = linux.IFLA_CARRIER
	IFLA_CARRIER_CHANGES        = linux.IFLA_CARRIER_CHANGES
	IFLA_CARRIER_UP_COUNT       = linux.IFLA_CARRIER_UP_COUNT
	IFLA_CARRIER_DOWN_COUNT     = linux.IFLA_CARRIER_DOWN_COUNT
	IFLA_PHYS_PORT_ID           = linux.IFLA_PHYS_PORT_ID
	IFLA_PHYS_SWITCH_ID         = linux.IFLA_PHYS_SWITCH_ID
	IFLA_PHYS_PORT_NAME         = linux.IFLA_PHYS_PORT_NAME
	IFLA_INFO_KIND              = linux.IFLA_INFO_KIND
	IFLA_INFO_SLAVE_KIND        = linux.IFLA_INFO_SLAVE_KIND
	IFLA_INFO_DATA              = linux.IFLA_INFO_DATA
	IFLA_INFO_SLAVE_DATA        = linux.IFLA_INFO_SLAVE_DATA
	IFLA_XDP                    = linux.IFLA_XDP
	IFLA_XDP_FD                 = linux.IFLA_XDP_FD
	IFLA_XDP_ATTACHED           = linux.IFLA_XDP_ATTACHED
	IFLA_XDP_FLAGS              = linux.IFLA_XDP_FLAGS
	IFLA_XDP_PROG_ID            = linux.IFLA_XDP_PROG_ID
	IFLA_XDP_EXPECTED_FD        = linux.IFLA_XDP_EXPECTED_FD
	XDP_FLAGS_DRV_MODE          = linux.XDP_FLAGS_DRV_MODE
	XDP_FLAGS_SKB_MODE          = linux.XDP_FLAGS_SKB_MODE
	XDP_FLAGS_HW_MODE           = linux.XDP_FLAGS_HW_MODE
	XDP_FLAGS_MODES             = linux.XDP_FLAGS_MODES
	XDP_FLAGS_MASK              = linux.XDP_FLAGS_MASK
	XDP_FLAGS_REPLACE           = linux.XDP_FLAGS_REPLACE
	XDP_FLAGS_UPDATE_IF_NOEXIST = linux.XDP_FLAGS_UPDATE_IF_NOEXIST
	LWTUNNEL_ENCAP_MPLS         = linux.LWTUNNEL_ENCAP_MPLS
	MPLS_IPTUNNEL_DST           = linux.MPLS_IPTUNNEL_DST
	MPLS_IPTUNNEL_TTL           = linux.MPLS_IPTUNNEL_TTL
	NDA_UNSPEC                  = linux.NDA_UNSPEC
	NDA_DST                     = linux.NDA_DST
	NDA_LLADDR                  = linux.NDA_LLADDR
	NDA_CACHEINFO               = linux.NDA_CACHEINFO
	NDA_IFINDEX                 = linux.NDA_IFINDEX
	RTA_UNSPEC                  = linux.RTA_UNSPEC
	RTA_DST                     = linux.RTA_DST
	RTA_ENCAP                   = linux.RTA_ENCAP
	RTA_ENCAP_TYPE              = linux.RTA_ENCAP_TYPE
	RTA_PREFSRC                 = linux.RTA_PREFSRC
	RTA_GATEWAY                 = linux.RTA_GATEWAY
	RTA_OIF                     = linux.RTA_OIF
	RTA_PRIORITY                = linux.RTA_PRIORITY
	RTA_TABLE                   = linux.RTA_TABLE
	RTA_MARK                    = linux.RTA_MARK
	RTA_EXPIRES                 = linux.RTA_EXPIRES
	RTA_METRICS                 = linux.RTA_METRICS
	RTA_MULTIPATH               = linux.RTA_MULTIPATH
	RTA_PREF                    = linux.RTA_PREF
	RTAX_ADVMSS                 = linux.RTAX_ADVMSS
	RTAX_FEATURES               = linux.RTAX_FEATURES
	RTAX_INITCWND               = linux.RTAX_INITCWND
	RTAX_INITRWND               = linux.RTAX_INITRWND
	RTAX_MTU                    = linux.RTAX_MTU
	NTF_PROXY                   = linux.NTF_PROXY
	RTN_UNICAST                 = linux.RTN_UNICAST
	RT_TABLE_MAIN               = linux.RT_TABLE_MAIN
	RTPROT_BOOT                 = linux.RTPROT_BOOT
	RTPROT_STATIC               = linux.RTPROT_STATIC
	RT_SCOPE_UNIVERSE           = linux.RT_SCOPE_UNIVERSE
	RT_SCOPE_HOST               = linux.RT_SCOPE_HOST
	RT_SCOPE_LINK               = linux.RT_SCOPE_LINK
	RTM_NEWRULE                 = linux.RTM_NEWRULE
	RTM_GETRULE                 = linux.RTM_GETRULE
	RTM_DELRULE                 = linux.RTM_DELRULE
	FRA_UNSPEC                  = linux.FRA_UNSPEC
	FRA_DST                     = linux.FRA_DST
	FRA_SRC                     = linux.FRA_SRC
	FRA_IIFNAME                 = linux.FRA_IIFNAME
	FRA_GOTO                    = linux.FRA_GOTO
	FRA_UNUSED2                 = linux.FRA_UNUSED2
	FRA_PRIORITY                = linux.FRA_PRIORITY
	FRA_UNUSED3                 = linux.FRA_UNUSED3
	FRA_UNUSED4                 = linux.FRA_UNUSED4
	FRA_UNUSED5                 = linux.FRA_UNUSED5
	FRA_FWMARK                  = linux.FRA_FWMARK
	FRA_FLOW                    = linux.FRA_FLOW
	FRA_TUN_ID                  = linux.FRA_TUN_ID
	FRA_SUPPRESS_IFGROUP        = linux.FRA_SUPPRESS_IFGROUP
	FRA_SUPPRESS_PREFIXLEN      = linux.FRA_SUPPRESS_PREFIXLEN
	FRA_TABLE                   = linux.FRA_TABLE
	FRA_FWMASK                  = linux.FRA_FWMASK
	FRA_OIFNAME                 = linux.FRA_OIFNAME
	FRA_PAD                     = linux.FRA_PAD
	FRA_L3MDEV                  = linux.FRA_L3MDEV
	FRA_UID_RANGE               = linux.FRA_UID_RANGE
	FRA_PROTOCOL                = linux.FRA_PROTOCOL
	FRA_IP_PROTO                = linux.FRA_IP_PROTO
	FRA_SPORT_RANGE             = linux.FRA_SPORT_RANGE
	FRA_DPORT_RANGE             = linux.FRA_DPORT_RANGE
)
