package tun

import (
	"encoding/binary"
	"fmt"
)

// GSOType represents the type of segmentation offload.
type GSOType int

const (
	GSONone GSOType = iota
	GSOTCPv4
	GSOTCPv6
	GSOUDPL4
)

func (g GSOType) String() string {
	switch g {
	case GSONone:
		return "GSONone"
	case GSOTCPv4:
		return "GSOTCPv4"
	case GSOTCPv6:
		return "GSOTCPv6"
	case GSOUDPL4:
		return "GSOUDPL4"
	default:
		return "unknown"
	}
}

// GSOOptions is loosely modeled after struct virtio_net_hdr from the VIRTIO
// specification. It is a common representation of GSO metadata that can be
// applied to support packet GSO across tun.Device implementations.
type GSOOptions struct {
	// GSOType represents the type of segmentation offload.
	GSOType GSOType
	// HdrLen is the sum of the layer 3 and 4 header lengths. This field may be
	// zero when GSOType == GSONone.
	HdrLen uint16
	// CsumStart is the head byte index of the packet data to be checksummed,
	// i.e. the start of the TCP or UDP header.
	CsumStart uint16
	// CsumOffset is the offset from CsumStart where the 2-byte checksum value
	// should be placed.
	CsumOffset uint16
	// GSOSize is the size of each segment exclusive of HdrLen. The tail segment
	// may be smaller than this value.
	GSOSize uint16
	// NeedsCsum may be set where GSOType == GSONone. When set, the checksum
	// at CsumStart + CsumOffset must be a partial checksum, i.e. the
	// pseudo-header sum.
	NeedsCsum bool
}

const (
	ipv4SrcAddrOffset = 12
	ipv6SrcAddrOffset = 8
)

const tcpFlagsOffset = 13

const (
	tcpFlagFIN uint8 = 0x01
	tcpFlagPSH uint8 = 0x08
	tcpFlagACK uint8 = 0x10
)

const (
	// defined here in order to avoid importation of any platform-specific pkgs
	ipProtoTCP = 6
	ipProtoUDP = 17
)

// GSOSplit splits packets from 'in' into outBufs[<index>][outOffset:], writing
// the size of each element into sizes. It returns the number of buffers
// populated, and/or an error. Callers may pass an 'in' slice that overlaps with
// the first element of outBuffers, i.e. &in[0] may be equal to
// &outBufs[0][outOffset]. GSONone is a valid options.GSOType regardless of the
// value of options.NeedsCsum. Length of each outBufs element must be greater
// than or equal to the length of 'in', otherwise output may be silently
// truncated.
func GSOSplit(in []byte, options GSOOptions, outBufs [][]byte, sizes []int, outOffset int) (int, error) {
	cSumAt := int(options.CsumStart) + int(options.CsumOffset)
	if cSumAt+1 >= len(in) {
		return 0, fmt.Errorf("end of checksum offset (%d) exceeds packet length (%d)", cSumAt+1, len(in))
	}

	if len(in) < int(options.HdrLen) {
		return 0, fmt.Errorf("length of packet (%d) < GSO HdrLen (%d)", len(in), options.HdrLen)
	}

	// Handle the conditions where we are copying a single element to outBuffs.
	payloadLen := len(in) - int(options.HdrLen)
	if options.GSOType == GSONone || payloadLen < int(options.GSOSize) {
		if len(in) > len(outBufs[0][outOffset:]) {
			return 0, fmt.Errorf("length of packet (%d) exceeds output element length (%d)", len(in), len(outBufs[0][outOffset:]))
		}
		if options.NeedsCsum {
			// The initial value at the checksum offset should be summed with
			// the checksum we compute. This is typically the pseudo-header sum.
			initial := binary.BigEndian.Uint16(in[cSumAt:])
			in[cSumAt], in[cSumAt+1] = 0, 0
			binary.BigEndian.PutUint16(in[cSumAt:], ^Checksum(in[options.CsumStart:], initial))
		}
		sizes[0] = copy(outBufs[0][outOffset:], in)
		return 1, nil
	}

	if options.HdrLen < options.CsumStart {
		return 0, fmt.Errorf("GSO HdrLen (%d) < GSO CsumStart (%d)", options.HdrLen, options.CsumStart)
	}

	ipVersion := in[0] >> 4
	switch ipVersion {
	case 4:
		if options.GSOType != GSOTCPv4 && options.GSOType != GSOUDPL4 {
			return 0, fmt.Errorf("ip header version: %d, GSO type: %s", ipVersion, options.GSOType)
		}
		if len(in) < 20 {
			return 0, fmt.Errorf("length of packet (%d) < minimum ipv4 header size (%d)", len(in), 20)
		}
	case 6:
		if options.GSOType != GSOTCPv6 && options.GSOType != GSOUDPL4 {
			return 0, fmt.Errorf("ip header version: %d, GSO type: %s", ipVersion, options.GSOType)
		}
		if len(in) < 40 {
			return 0, fmt.Errorf("length of packet (%d) < minimum ipv6 header size (%d)", len(in), 40)
		}
	default:
		return 0, fmt.Errorf("invalid ip header version: %d", ipVersion)
	}

	iphLen := int(options.CsumStart)
	srcAddrOffset := ipv6SrcAddrOffset
	addrLen := 16
	if ipVersion == 4 {
		srcAddrOffset = ipv4SrcAddrOffset
		addrLen = 4
	}
	transportCsumAt := int(options.CsumStart + options.CsumOffset)
	var firstTCPSeqNum uint32
	var protocol uint8
	if options.GSOType == GSOTCPv4 || options.GSOType == GSOTCPv6 {
		protocol = ipProtoTCP
		if len(in) < int(options.CsumStart)+20 {
			return 0, fmt.Errorf("length of packet (%d) < GSO CsumStart (%d) + minimum TCP header size (%d)",
				len(in), options.CsumStart, 20)
		}
		firstTCPSeqNum = binary.BigEndian.Uint32(in[options.CsumStart+4:])
	} else {
		protocol = ipProtoUDP
	}
	nextSegmentDataAt := int(options.HdrLen)
	i := 0
	for ; nextSegmentDataAt < len(in); i++ {
		if i == len(outBufs) {
			return i - 1, ErrTooManySegments
		}
		nextSegmentEnd := nextSegmentDataAt + int(options.GSOSize)
		if nextSegmentEnd > len(in) {
			nextSegmentEnd = len(in)
		}
		segmentDataLen := nextSegmentEnd - nextSegmentDataAt
		totalLen := int(options.HdrLen) + segmentDataLen
		sizes[i] = totalLen
		out := outBufs[i][outOffset:]

		copy(out, in[:iphLen])
		if ipVersion == 4 {
			// For IPv4 we are responsible for incrementing the ID field,
			// updating the total len field, and recalculating the header
			// checksum.
			if i > 0 {
				id := binary.BigEndian.Uint16(out[4:])
				id += uint16(i)
				binary.BigEndian.PutUint16(out[4:], id)
			}
			out[10], out[11] = 0, 0 // clear ipv4 header checksum
			binary.BigEndian.PutUint16(out[2:], uint16(totalLen))
			ipv4CSum := ^Checksum(out[:iphLen], 0)
			binary.BigEndian.PutUint16(out[10:], ipv4CSum)
		} else {
			// For IPv6 we are responsible for updating the payload length field.
			binary.BigEndian.PutUint16(out[4:], uint16(totalLen-iphLen))
		}

		// copy transport header
		copy(out[options.CsumStart:options.HdrLen], in[options.CsumStart:options.HdrLen])

		if protocol == ipProtoTCP {
			// set TCP seq and adjust TCP flags
			tcpSeq := firstTCPSeqNum + uint32(options.GSOSize*uint16(i))
			binary.BigEndian.PutUint32(out[options.CsumStart+4:], tcpSeq)
			if nextSegmentEnd != len(in) {
				// FIN and PSH should only be set on last segment
				clearFlags := tcpFlagFIN | tcpFlagPSH
				out[options.CsumStart+tcpFlagsOffset] &^= clearFlags
			}
		} else {
			// set UDP header len
			binary.BigEndian.PutUint16(out[options.CsumStart+4:], uint16(segmentDataLen)+(options.HdrLen-options.CsumStart))
		}

		// payload
		copy(out[options.HdrLen:], in[nextSegmentDataAt:nextSegmentEnd])

		// transport checksum
		out[transportCsumAt], out[transportCsumAt+1] = 0, 0 // clear tcp/udp checksum
		transportHeaderLen := int(options.HdrLen - options.CsumStart)
		lenForPseudo := uint16(transportHeaderLen + segmentDataLen)
		transportCSum := PseudoHeaderChecksum(protocol, in[srcAddrOffset:srcAddrOffset+addrLen], in[srcAddrOffset+addrLen:srcAddrOffset+addrLen*2], lenForPseudo)
		transportCSum = ^Checksum(out[options.CsumStart:totalLen], transportCSum)
		binary.BigEndian.PutUint16(out[options.CsumStart+options.CsumOffset:], transportCSum)

		nextSegmentDataAt += int(options.GSOSize)
	}
	return i, nil
}
