package rtnetlink

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"

	"github.com/jsimonetti/rtnetlink/internal/unix"
	"github.com/mdlayher/netlink"
)

var (
	// errInvalidRuleMessage is returned when a RuleMessage is malformed.
	errInvalidRuleMessage = errors.New("rtnetlink RuleMessage is invalid or too short")

	// errInvalidRuleAttribute is returned when a RuleMessage contains an unknown attribute.
	errInvalidRuleAttribute = errors.New("rtnetlink RuleMessage contains an unknown Attribute")
)

var _ Message = &RuleMessage{}

// A RuleMessage is a route netlink link message.
type RuleMessage struct {
	// Address family
	Family uint8

	// Length of destination prefix
	DstLength uint8

	// Length of source prefix
	SrcLength uint8

	// Rule TOS
	TOS uint8

	// Routing table identifier
	Table uint8

	// Rule action
	Action uint8

	// Rule flags
	Flags uint32

	// Attributes List
	Attributes *RuleAttributes
}

// MarshalBinary marshals a LinkMessage into a byte slice.
func (m *RuleMessage) MarshalBinary() ([]byte, error) {
	b := make([]byte, 12)

	// fib_rule_hdr
	b[0] = m.Family
	b[1] = m.DstLength
	b[2] = m.SrcLength
	b[3] = m.TOS
	b[4] = m.Table
	b[7] = m.Action
	nativeEndian.PutUint32(b[8:12], m.Flags)

	if m.Attributes != nil {
		ae := netlink.NewAttributeEncoder()
		ae.ByteOrder = nativeEndian
		err := m.Attributes.encode(ae)
		if err != nil {
			return nil, err
		}

		a, err := ae.Encode()
		if err != nil {
			return nil, err
		}

		return append(b, a...), nil
	}

	return b, nil
}

// UnmarshalBinary unmarshals the contents of a byte slice into a LinkMessage.
func (m *RuleMessage) UnmarshalBinary(b []byte) error {
	l := len(b)
	if l < 12 {
		return errInvalidRuleMessage
	}
	m.Family = b[0]
	m.DstLength = b[1]
	m.SrcLength = b[2]
	m.TOS = b[3]
	m.Table = b[4]
	// b[5] and b[6] are reserved fields
	m.Action = b[7]
	m.Flags = nativeEndian.Uint32(b[8:12])

	if l > 12 {
		m.Attributes = &RuleAttributes{}
		ad, err := netlink.NewAttributeDecoder(b[12:])
		if err != nil {
			return err
		}
		ad.ByteOrder = nativeEndian
		return m.Attributes.decode(ad)
	}
	return nil
}

// rtMessage is an empty method to sattisfy the Message interface.
func (*RuleMessage) rtMessage() {}

// RuleService is used to retrieve rtnetlink family information.
type RuleService struct {
	c *Conn
}

func (r *RuleService) execute(m Message, family uint16, flags netlink.HeaderFlags) ([]RuleMessage, error) {
	msgs, err := r.c.Execute(m, family, flags)

	rules := make([]RuleMessage, len(msgs))
	for i := range msgs {
		rules[i] = *msgs[i].(*RuleMessage)
	}

	return rules, err
}

// Add new rule
func (r *RuleService) Add(req *RuleMessage) error {
	flags := netlink.Request | netlink.Create | netlink.Acknowledge | netlink.Excl
	_, err := r.c.Execute(req, unix.RTM_NEWRULE, flags)

	return err
}

// Replace or add new rule
func (r *RuleService) Replace(req *RuleMessage) error {
	flags := netlink.Request | netlink.Create | netlink.Replace | netlink.Acknowledge
	_, err := r.c.Execute(req, unix.RTM_NEWRULE, flags)

	return err
}

// Delete existing rule
func (r *RuleService) Delete(req *RuleMessage) error {
	flags := netlink.Request | netlink.Acknowledge
	_, err := r.c.Execute(req, unix.RTM_DELRULE, flags)

	return err
}

// Get Rule(s)
func (r *RuleService) Get(req *RuleMessage) ([]RuleMessage, error) {
	flags := netlink.Request | netlink.DumpFiltered
	return r.execute(req, unix.RTM_GETRULE, flags)
}

// List all rules
func (r *RuleService) List() ([]RuleMessage, error) {
	flags := netlink.Request | netlink.Dump
	return r.execute(&RuleMessage{}, unix.RTM_GETRULE, flags)
}

// RuleAttributes contains all attributes for a rule.
type RuleAttributes struct {
	Src, Dst          *net.IP
	IIFName, OIFName  *string
	Goto              *uint32
	Priority          *uint32
	FwMark, FwMask    *uint32
	SrcRealm          *uint16
	DstRealm          *uint16
	TunID             *uint64
	Table             *uint32
	L3MDev            *uint8
	Protocol          *uint8
	IPProto           *uint8
	SuppressPrefixLen *uint32
	SuppressIFGroup   *uint32
	UIDRange          *RuleUIDRange
	SPortRange        *RulePortRange
	DPortRange        *RulePortRange
}

// unmarshalBinary unmarshals the contents of a byte slice into a RuleMessage.
func (r *RuleAttributes) decode(ad *netlink.AttributeDecoder) error {
	for ad.Next() {
		switch ad.Type() {
		case unix.FRA_UNSPEC:
			// unused
			continue
		case unix.FRA_DST:
			r.Dst = &net.IP{}
			ad.Do(decodeIP(r.Dst))
		case unix.FRA_SRC:
			r.Src = &net.IP{}
			ad.Do(decodeIP(r.Src))
		case unix.FRA_IIFNAME:
			v := ad.String()
			r.IIFName = &v
		case unix.FRA_GOTO:
			v := ad.Uint32()
			r.Goto = &v
		case unix.FRA_UNUSED2:
			// unused
			continue
		case unix.FRA_PRIORITY:
			v := ad.Uint32()
			r.Priority = &v
		case unix.FRA_UNUSED3:
			// unused
			continue
		case unix.FRA_UNUSED4:
			// unused
			continue
		case unix.FRA_UNUSED5:
			// unused
			continue
		case unix.FRA_FWMARK:
			v := ad.Uint32()
			r.FwMark = &v
		case unix.FRA_FLOW:
			dst32 := ad.Uint32()
			src32 := uint32(dst32 >> 16)
			src32 &= 0xFFFF
			dst32 &= 0xFFFF
			src16 := uint16(src32)
			dst16 := uint16(dst32)
			r.SrcRealm = &src16
			r.DstRealm = &dst16
		case unix.FRA_TUN_ID:
			v := ad.Uint64()
			r.TunID = &v
		case unix.FRA_SUPPRESS_IFGROUP:
			v := ad.Uint32()
			r.SuppressIFGroup = &v
		case unix.FRA_SUPPRESS_PREFIXLEN:
			v := ad.Uint32()
			r.SuppressPrefixLen = &v
		case unix.FRA_TABLE:
			v := ad.Uint32()
			r.Table = &v
		case unix.FRA_FWMASK:
			v := ad.Uint32()
			r.FwMask = &v
		case unix.FRA_OIFNAME:
			v := ad.String()
			r.OIFName = &v
		case unix.FRA_PAD:
			// unused
			continue
		case unix.FRA_L3MDEV:
			v := ad.Uint8()
			r.L3MDev = &v
		case unix.FRA_UID_RANGE:
			r.UIDRange = &RuleUIDRange{}
			err := r.UIDRange.unmarshalBinary(ad.Bytes())
			if err != nil {
				return err
			}
		case unix.FRA_PROTOCOL:
			v := ad.Uint8()
			r.Protocol = &v
		case unix.FRA_IP_PROTO:
			v := ad.Uint8()
			r.IPProto = &v
		case unix.FRA_SPORT_RANGE:
			r.SPortRange = &RulePortRange{}
			err := r.SPortRange.unmarshalBinary(ad.Bytes())
			if err != nil {
				return err
			}
		case unix.FRA_DPORT_RANGE:
			r.DPortRange = &RulePortRange{}
			err := r.DPortRange.unmarshalBinary(ad.Bytes())
			if err != nil {
				return err
			}
		default:
			return errInvalidRuleAttribute
		}
	}
	return ad.Err()
}

// MarshalBinary marshals a RuleAttributes into a byte slice.
func (r *RuleAttributes) encode(ae *netlink.AttributeEncoder) error {
	if r.Table != nil {
		ae.Uint32(unix.FRA_TABLE, *r.Table)
	}
	if r.Protocol != nil {
		ae.Uint8(unix.FRA_PROTOCOL, *r.Protocol)
	}
	if r.Src != nil {
		ae.Do(unix.FRA_SRC, encodeIP(*r.Src))
	}
	if r.Dst != nil {
		ae.Do(unix.FRA_DST, encodeIP(*r.Dst))
	}
	if r.IIFName != nil {
		ae.String(unix.FRA_IIFNAME, *r.IIFName)
	}
	if r.OIFName != nil {
		ae.String(unix.FRA_OIFNAME, *r.OIFName)
	}
	if r.Goto != nil {
		ae.Uint32(unix.FRA_GOTO, *r.Goto)
	}
	if r.Priority != nil {
		ae.Uint32(unix.FRA_PRIORITY, *r.Priority)
	}
	if r.FwMark != nil {
		ae.Uint32(unix.FRA_FWMARK, *r.FwMark)
	}
	if r.FwMask != nil {
		ae.Uint32(unix.FRA_FWMASK, *r.FwMask)
	}
	if r.DstRealm != nil {
		value := uint32(*r.DstRealm)
		if r.SrcRealm != nil {
			value |= (uint32(*r.SrcRealm&0xFFFF) << 16)
		}
		ae.Uint32(unix.FRA_FLOW, value)
	}
	if r.TunID != nil {
		ae.Uint64(unix.FRA_TUN_ID, *r.TunID)
	}
	if r.L3MDev != nil {
		ae.Uint8(unix.FRA_L3MDEV, *r.L3MDev)
	}
	if r.IPProto != nil {
		ae.Uint8(unix.FRA_IP_PROTO, *r.IPProto)
	}
	if r.SuppressIFGroup != nil {
		ae.Uint32(unix.FRA_SUPPRESS_IFGROUP, *r.SuppressIFGroup)
	}
	if r.SuppressPrefixLen != nil {
		ae.Uint32(unix.FRA_SUPPRESS_PREFIXLEN, *r.SuppressPrefixLen)
	}
	if r.UIDRange != nil {
		data, err := marshalRuleUIDRange(*r.UIDRange)
		if err != nil {
			return err
		}
		ae.Bytes(unix.FRA_UID_RANGE, data)
	}
	if r.SPortRange != nil {
		data, err := marshalRulePortRange(*r.SPortRange)
		if err != nil {
			return err
		}
		ae.Bytes(unix.FRA_SPORT_RANGE, data)
	}
	if r.DPortRange != nil {
		data, err := marshalRulePortRange(*r.DPortRange)
		if err != nil {
			return err
		}
		ae.Bytes(unix.FRA_DPORT_RANGE, data)
	}
	return nil
}

// RulePortRange defines start and end ports for a rule
type RulePortRange struct {
	Start, End uint16
}

func (r *RulePortRange) unmarshalBinary(data []byte) error {
	b := bytes.NewReader(data)
	return binary.Read(b, nativeEndian, r)
}

func marshalRulePortRange(s RulePortRange) ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, nativeEndian, s)
	return buf.Bytes(), err
}

// RuleUIDRange defines the start and end for UID matches
type RuleUIDRange struct {
	Start, End uint16
}

func (r *RuleUIDRange) unmarshalBinary(data []byte) error {
	b := bytes.NewReader(data)
	return binary.Read(b, nativeEndian, r)
}

func marshalRuleUIDRange(s RuleUIDRange) ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, nativeEndian, s)
	return buf.Bytes(), err
}
