package tun

import (
	"encoding/binary"
	"math/bits"
	"strconv"

	"golang.org/x/sys/cpu"
)

// checksumGeneric64 is a reference implementation of checksum using 64 bit
// arithmetic for use in testing or when an architecture-specific implementation
// is not available.
func checksumGeneric64(b []byte, initial uint16) uint16 {
	var ac uint64
	var carry uint64

	if cpu.IsBigEndian {
		ac = uint64(initial)
	} else {
		ac = uint64(bits.ReverseBytes16(initial))
	}

	for len(b) >= 128 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[:8]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[8:16]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[16:24]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[24:32]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[32:40]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[40:48]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[48:56]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[56:64]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[64:72]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[72:80]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[80:88]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[88:96]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[96:104]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[104:112]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[112:120]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[120:128]), carry)
		} else {
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[:8]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[8:16]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[16:24]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[24:32]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[32:40]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[40:48]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[48:56]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[56:64]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[64:72]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[72:80]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[80:88]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[88:96]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[96:104]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[104:112]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[112:120]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[120:128]), carry)
		}
		b = b[128:]
	}
	if len(b) >= 64 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[:8]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[8:16]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[16:24]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[24:32]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[32:40]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[40:48]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[48:56]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[56:64]), carry)
		} else {
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[:8]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[8:16]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[16:24]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[24:32]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[32:40]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[40:48]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[48:56]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[56:64]), carry)
		}
		b = b[64:]
	}
	if len(b) >= 32 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[:8]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[8:16]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[16:24]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[24:32]), carry)
		} else {
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[:8]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[8:16]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[16:24]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[24:32]), carry)
		}
		b = b[32:]
	}
	if len(b) >= 16 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[:8]), carry)
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b[8:16]), carry)
		} else {
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[:8]), carry)
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b[8:16]), carry)
		}
		b = b[16:]
	}
	if len(b) >= 8 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add64(ac, binary.BigEndian.Uint64(b), carry)
		} else {
			ac, carry = bits.Add64(ac, binary.LittleEndian.Uint64(b), carry)
		}
		b = b[8:]
	}
	if len(b) >= 4 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add64(ac, uint64(binary.BigEndian.Uint32(b)), carry)
		} else {
			ac, carry = bits.Add64(ac, uint64(binary.LittleEndian.Uint32(b)), carry)
		}
		b = b[4:]
	}
	if len(b) >= 2 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add64(ac, uint64(binary.BigEndian.Uint16(b)), carry)
		} else {
			ac, carry = bits.Add64(ac, uint64(binary.LittleEndian.Uint16(b)), carry)
		}
		b = b[2:]
	}
	if len(b) >= 1 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add64(ac, uint64(b[0])<<8, carry)
		} else {
			ac, carry = bits.Add64(ac, uint64(b[0]), carry)
		}
	}

	folded := ipChecksumFold64(ac, carry)
	if !cpu.IsBigEndian {
		folded = bits.ReverseBytes16(folded)
	}
	return folded
}

// checksumGeneric32 is a reference implementation of checksum using 32 bit
// arithmetic for use in testing or when an architecture-specific implementation
// is not available.
func checksumGeneric32(b []byte, initial uint16) uint16 {
	var ac uint32
	var carry uint32

	if cpu.IsBigEndian {
		ac = uint32(initial)
	} else {
		ac = uint32(bits.ReverseBytes16(initial))
	}

	for len(b) >= 64 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[:8]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[4:8]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[8:12]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[12:16]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[16:20]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[20:24]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[24:28]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[28:32]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[32:36]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[36:40]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[40:44]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[44:48]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[48:52]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[52:56]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[56:60]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[60:64]), carry)
		} else {
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[:8]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[4:8]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[8:12]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[12:16]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[16:20]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[20:24]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[24:28]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[28:32]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[32:36]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[36:40]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[40:44]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[44:48]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[48:52]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[52:56]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[56:60]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[60:64]), carry)
		}
		b = b[64:]
	}
	if len(b) >= 32 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[:4]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[4:8]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[8:12]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[12:16]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[16:20]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[20:24]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[24:28]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[28:32]), carry)
		} else {
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[:4]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[4:8]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[8:12]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[12:16]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[16:20]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[20:24]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[24:28]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[28:32]), carry)
		}
		b = b[32:]
	}
	if len(b) >= 16 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[:4]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[4:8]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[8:12]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[12:16]), carry)
		} else {
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[:4]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[4:8]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[8:12]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[12:16]), carry)
		}
		b = b[16:]
	}
	if len(b) >= 8 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[:4]), carry)
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b[4:8]), carry)
		} else {
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[:4]), carry)
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b[4:8]), carry)
		}
		b = b[8:]
	}
	if len(b) >= 4 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add32(ac, binary.BigEndian.Uint32(b), carry)
		} else {
			ac, carry = bits.Add32(ac, binary.LittleEndian.Uint32(b), carry)
		}
		b = b[4:]
	}
	if len(b) >= 2 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add32(ac, uint32(binary.BigEndian.Uint16(b)), carry)
		} else {
			ac, carry = bits.Add32(ac, uint32(binary.LittleEndian.Uint16(b)), carry)
		}
		b = b[2:]
	}
	if len(b) >= 1 {
		if cpu.IsBigEndian {
			ac, carry = bits.Add32(ac, uint32(b[0])<<8, carry)
		} else {
			ac, carry = bits.Add32(ac, uint32(b[0]), carry)
		}
	}

	folded := ipChecksumFold32(ac, carry)
	if !cpu.IsBigEndian {
		folded = bits.ReverseBytes16(folded)
	}
	return folded
}

// checksumGeneric32Alternate is an alternate reference implementation of
// checksum using 32 bit arithmetic for use in testing or when an
// architecture-specific implementation is not available.
func checksumGeneric32Alternate(b []byte, initial uint16) uint16 {
	var ac uint32

	if cpu.IsBigEndian {
		ac = uint32(initial)
	} else {
		ac = uint32(bits.ReverseBytes16(initial))
	}

	for len(b) >= 64 {
		if cpu.IsBigEndian {
			ac += uint32(binary.BigEndian.Uint16(b[:2]))
			ac += uint32(binary.BigEndian.Uint16(b[2:4]))
			ac += uint32(binary.BigEndian.Uint16(b[4:6]))
			ac += uint32(binary.BigEndian.Uint16(b[6:8]))
			ac += uint32(binary.BigEndian.Uint16(b[8:10]))
			ac += uint32(binary.BigEndian.Uint16(b[10:12]))
			ac += uint32(binary.BigEndian.Uint16(b[12:14]))
			ac += uint32(binary.BigEndian.Uint16(b[14:16]))
			ac += uint32(binary.BigEndian.Uint16(b[16:18]))
			ac += uint32(binary.BigEndian.Uint16(b[18:20]))
			ac += uint32(binary.BigEndian.Uint16(b[20:22]))
			ac += uint32(binary.BigEndian.Uint16(b[22:24]))
			ac += uint32(binary.BigEndian.Uint16(b[24:26]))
			ac += uint32(binary.BigEndian.Uint16(b[26:28]))
			ac += uint32(binary.BigEndian.Uint16(b[28:30]))
			ac += uint32(binary.BigEndian.Uint16(b[30:32]))
			ac += uint32(binary.BigEndian.Uint16(b[32:34]))
			ac += uint32(binary.BigEndian.Uint16(b[34:36]))
			ac += uint32(binary.BigEndian.Uint16(b[36:38]))
			ac += uint32(binary.BigEndian.Uint16(b[38:40]))
			ac += uint32(binary.BigEndian.Uint16(b[40:42]))
			ac += uint32(binary.BigEndian.Uint16(b[42:44]))
			ac += uint32(binary.BigEndian.Uint16(b[44:46]))
			ac += uint32(binary.BigEndian.Uint16(b[46:48]))
			ac += uint32(binary.BigEndian.Uint16(b[48:50]))
			ac += uint32(binary.BigEndian.Uint16(b[50:52]))
			ac += uint32(binary.BigEndian.Uint16(b[52:54]))
			ac += uint32(binary.BigEndian.Uint16(b[54:56]))
			ac += uint32(binary.BigEndian.Uint16(b[56:58]))
			ac += uint32(binary.BigEndian.Uint16(b[58:60]))
			ac += uint32(binary.BigEndian.Uint16(b[60:62]))
			ac += uint32(binary.BigEndian.Uint16(b[62:64]))
		} else {
			ac += uint32(binary.LittleEndian.Uint16(b[:2]))
			ac += uint32(binary.LittleEndian.Uint16(b[2:4]))
			ac += uint32(binary.LittleEndian.Uint16(b[4:6]))
			ac += uint32(binary.LittleEndian.Uint16(b[6:8]))
			ac += uint32(binary.LittleEndian.Uint16(b[8:10]))
			ac += uint32(binary.LittleEndian.Uint16(b[10:12]))
			ac += uint32(binary.LittleEndian.Uint16(b[12:14]))
			ac += uint32(binary.LittleEndian.Uint16(b[14:16]))
			ac += uint32(binary.LittleEndian.Uint16(b[16:18]))
			ac += uint32(binary.LittleEndian.Uint16(b[18:20]))
			ac += uint32(binary.LittleEndian.Uint16(b[20:22]))
			ac += uint32(binary.LittleEndian.Uint16(b[22:24]))
			ac += uint32(binary.LittleEndian.Uint16(b[24:26]))
			ac += uint32(binary.LittleEndian.Uint16(b[26:28]))
			ac += uint32(binary.LittleEndian.Uint16(b[28:30]))
			ac += uint32(binary.LittleEndian.Uint16(b[30:32]))
			ac += uint32(binary.LittleEndian.Uint16(b[32:34]))
			ac += uint32(binary.LittleEndian.Uint16(b[34:36]))
			ac += uint32(binary.LittleEndian.Uint16(b[36:38]))
			ac += uint32(binary.LittleEndian.Uint16(b[38:40]))
			ac += uint32(binary.LittleEndian.Uint16(b[40:42]))
			ac += uint32(binary.LittleEndian.Uint16(b[42:44]))
			ac += uint32(binary.LittleEndian.Uint16(b[44:46]))
			ac += uint32(binary.LittleEndian.Uint16(b[46:48]))
			ac += uint32(binary.LittleEndian.Uint16(b[48:50]))
			ac += uint32(binary.LittleEndian.Uint16(b[50:52]))
			ac += uint32(binary.LittleEndian.Uint16(b[52:54]))
			ac += uint32(binary.LittleEndian.Uint16(b[54:56]))
			ac += uint32(binary.LittleEndian.Uint16(b[56:58]))
			ac += uint32(binary.LittleEndian.Uint16(b[58:60]))
			ac += uint32(binary.LittleEndian.Uint16(b[60:62]))
			ac += uint32(binary.LittleEndian.Uint16(b[62:64]))
		}
		b = b[64:]
	}
	if len(b) >= 32 {
		if cpu.IsBigEndian {
			ac += uint32(binary.BigEndian.Uint16(b[:2]))
			ac += uint32(binary.BigEndian.Uint16(b[2:4]))
			ac += uint32(binary.BigEndian.Uint16(b[4:6]))
			ac += uint32(binary.BigEndian.Uint16(b[6:8]))
			ac += uint32(binary.BigEndian.Uint16(b[8:10]))
			ac += uint32(binary.BigEndian.Uint16(b[10:12]))
			ac += uint32(binary.BigEndian.Uint16(b[12:14]))
			ac += uint32(binary.BigEndian.Uint16(b[14:16]))
			ac += uint32(binary.BigEndian.Uint16(b[16:18]))
			ac += uint32(binary.BigEndian.Uint16(b[18:20]))
			ac += uint32(binary.BigEndian.Uint16(b[20:22]))
			ac += uint32(binary.BigEndian.Uint16(b[22:24]))
			ac += uint32(binary.BigEndian.Uint16(b[24:26]))
			ac += uint32(binary.BigEndian.Uint16(b[26:28]))
			ac += uint32(binary.BigEndian.Uint16(b[28:30]))
			ac += uint32(binary.BigEndian.Uint16(b[30:32]))
		} else {
			ac += uint32(binary.LittleEndian.Uint16(b[:2]))
			ac += uint32(binary.LittleEndian.Uint16(b[2:4]))
			ac += uint32(binary.LittleEndian.Uint16(b[4:6]))
			ac += uint32(binary.LittleEndian.Uint16(b[6:8]))
			ac += uint32(binary.LittleEndian.Uint16(b[8:10]))
			ac += uint32(binary.LittleEndian.Uint16(b[10:12]))
			ac += uint32(binary.LittleEndian.Uint16(b[12:14]))
			ac += uint32(binary.LittleEndian.Uint16(b[14:16]))
			ac += uint32(binary.LittleEndian.Uint16(b[16:18]))
			ac += uint32(binary.LittleEndian.Uint16(b[18:20]))
			ac += uint32(binary.LittleEndian.Uint16(b[20:22]))
			ac += uint32(binary.LittleEndian.Uint16(b[22:24]))
			ac += uint32(binary.LittleEndian.Uint16(b[24:26]))
			ac += uint32(binary.LittleEndian.Uint16(b[26:28]))
			ac += uint32(binary.LittleEndian.Uint16(b[28:30]))
			ac += uint32(binary.LittleEndian.Uint16(b[30:32]))
		}
		b = b[32:]
	}
	if len(b) >= 16 {
		if cpu.IsBigEndian {
			ac += uint32(binary.BigEndian.Uint16(b[:2]))
			ac += uint32(binary.BigEndian.Uint16(b[2:4]))
			ac += uint32(binary.BigEndian.Uint16(b[4:6]))
			ac += uint32(binary.BigEndian.Uint16(b[6:8]))
			ac += uint32(binary.BigEndian.Uint16(b[8:10]))
			ac += uint32(binary.BigEndian.Uint16(b[10:12]))
			ac += uint32(binary.BigEndian.Uint16(b[12:14]))
			ac += uint32(binary.BigEndian.Uint16(b[14:16]))
		} else {
			ac += uint32(binary.LittleEndian.Uint16(b[:2]))
			ac += uint32(binary.LittleEndian.Uint16(b[2:4]))
			ac += uint32(binary.LittleEndian.Uint16(b[4:6]))
			ac += uint32(binary.LittleEndian.Uint16(b[6:8]))
			ac += uint32(binary.LittleEndian.Uint16(b[8:10]))
			ac += uint32(binary.LittleEndian.Uint16(b[10:12]))
			ac += uint32(binary.LittleEndian.Uint16(b[12:14]))
			ac += uint32(binary.LittleEndian.Uint16(b[14:16]))
		}
		b = b[16:]
	}
	if len(b) >= 8 {
		if cpu.IsBigEndian {
			ac += uint32(binary.BigEndian.Uint16(b[:2]))
			ac += uint32(binary.BigEndian.Uint16(b[2:4]))
			ac += uint32(binary.BigEndian.Uint16(b[4:6]))
			ac += uint32(binary.BigEndian.Uint16(b[6:8]))
		} else {
			ac += uint32(binary.LittleEndian.Uint16(b[:2]))
			ac += uint32(binary.LittleEndian.Uint16(b[2:4]))
			ac += uint32(binary.LittleEndian.Uint16(b[4:6]))
			ac += uint32(binary.LittleEndian.Uint16(b[6:8]))
		}
		b = b[8:]
	}
	if len(b) >= 4 {
		if cpu.IsBigEndian {
			ac += uint32(binary.BigEndian.Uint16(b[:2]))
			ac += uint32(binary.BigEndian.Uint16(b[2:4]))
		} else {
			ac += uint32(binary.LittleEndian.Uint16(b[:2]))
			ac += uint32(binary.LittleEndian.Uint16(b[2:4]))
		}
		b = b[4:]
	}
	if len(b) >= 2 {
		if cpu.IsBigEndian {
			ac += uint32(binary.BigEndian.Uint16(b))
		} else {
			ac += uint32(binary.LittleEndian.Uint16(b))
		}
		b = b[2:]
	}
	if len(b) >= 1 {
		if cpu.IsBigEndian {
			ac += uint32(b[0]) << 8
		} else {
			ac += uint32(b[0])
		}
	}

	folded := ipChecksumFold32(ac, 0)
	if !cpu.IsBigEndian {
		folded = bits.ReverseBytes16(folded)
	}
	return folded
}

// checksumGeneric64Alternate is an alternate reference implementation of
// checksum using 64 bit arithmetic for use in testing or when an
// architecture-specific implementation is not available.
func checksumGeneric64Alternate(b []byte, initial uint16) uint16 {
	var ac uint64

	if cpu.IsBigEndian {
		ac = uint64(initial)
	} else {
		ac = uint64(bits.ReverseBytes16(initial))
	}

	for len(b) >= 64 {
		if cpu.IsBigEndian {
			ac += uint64(binary.BigEndian.Uint32(b[:4]))
			ac += uint64(binary.BigEndian.Uint32(b[4:8]))
			ac += uint64(binary.BigEndian.Uint32(b[8:12]))
			ac += uint64(binary.BigEndian.Uint32(b[12:16]))
			ac += uint64(binary.BigEndian.Uint32(b[16:20]))
			ac += uint64(binary.BigEndian.Uint32(b[20:24]))
			ac += uint64(binary.BigEndian.Uint32(b[24:28]))
			ac += uint64(binary.BigEndian.Uint32(b[28:32]))
			ac += uint64(binary.BigEndian.Uint32(b[32:36]))
			ac += uint64(binary.BigEndian.Uint32(b[36:40]))
			ac += uint64(binary.BigEndian.Uint32(b[40:44]))
			ac += uint64(binary.BigEndian.Uint32(b[44:48]))
			ac += uint64(binary.BigEndian.Uint32(b[48:52]))
			ac += uint64(binary.BigEndian.Uint32(b[52:56]))
			ac += uint64(binary.BigEndian.Uint32(b[56:60]))
			ac += uint64(binary.BigEndian.Uint32(b[60:64]))
		} else {
			ac += uint64(binary.LittleEndian.Uint32(b[:4]))
			ac += uint64(binary.LittleEndian.Uint32(b[4:8]))
			ac += uint64(binary.LittleEndian.Uint32(b[8:12]))
			ac += uint64(binary.LittleEndian.Uint32(b[12:16]))
			ac += uint64(binary.LittleEndian.Uint32(b[16:20]))
			ac += uint64(binary.LittleEndian.Uint32(b[20:24]))
			ac += uint64(binary.LittleEndian.Uint32(b[24:28]))
			ac += uint64(binary.LittleEndian.Uint32(b[28:32]))
			ac += uint64(binary.LittleEndian.Uint32(b[32:36]))
			ac += uint64(binary.LittleEndian.Uint32(b[36:40]))
			ac += uint64(binary.LittleEndian.Uint32(b[40:44]))
			ac += uint64(binary.LittleEndian.Uint32(b[44:48]))
			ac += uint64(binary.LittleEndian.Uint32(b[48:52]))
			ac += uint64(binary.LittleEndian.Uint32(b[52:56]))
			ac += uint64(binary.LittleEndian.Uint32(b[56:60]))
			ac += uint64(binary.LittleEndian.Uint32(b[60:64]))
		}
		b = b[64:]
	}
	if len(b) >= 32 {
		if cpu.IsBigEndian {
			ac += uint64(binary.BigEndian.Uint32(b[:4]))
			ac += uint64(binary.BigEndian.Uint32(b[4:8]))
			ac += uint64(binary.BigEndian.Uint32(b[8:12]))
			ac += uint64(binary.BigEndian.Uint32(b[12:16]))
			ac += uint64(binary.BigEndian.Uint32(b[16:20]))
			ac += uint64(binary.BigEndian.Uint32(b[20:24]))
			ac += uint64(binary.BigEndian.Uint32(b[24:28]))
			ac += uint64(binary.BigEndian.Uint32(b[28:32]))
		} else {
			ac += uint64(binary.LittleEndian.Uint32(b[:4]))
			ac += uint64(binary.LittleEndian.Uint32(b[4:8]))
			ac += uint64(binary.LittleEndian.Uint32(b[8:12]))
			ac += uint64(binary.LittleEndian.Uint32(b[12:16]))
			ac += uint64(binary.LittleEndian.Uint32(b[16:20]))
			ac += uint64(binary.LittleEndian.Uint32(b[20:24]))
			ac += uint64(binary.LittleEndian.Uint32(b[24:28]))
			ac += uint64(binary.LittleEndian.Uint32(b[28:32]))
		}
		b = b[32:]
	}
	if len(b) >= 16 {
		if cpu.IsBigEndian {
			ac += uint64(binary.BigEndian.Uint32(b[:4]))
			ac += uint64(binary.BigEndian.Uint32(b[4:8]))
			ac += uint64(binary.BigEndian.Uint32(b[8:12]))
			ac += uint64(binary.BigEndian.Uint32(b[12:16]))
		} else {
			ac += uint64(binary.LittleEndian.Uint32(b[:4]))
			ac += uint64(binary.LittleEndian.Uint32(b[4:8]))
			ac += uint64(binary.LittleEndian.Uint32(b[8:12]))
			ac += uint64(binary.LittleEndian.Uint32(b[12:16]))
		}
		b = b[16:]
	}
	if len(b) >= 8 {
		if cpu.IsBigEndian {
			ac += uint64(binary.BigEndian.Uint32(b[:4]))
			ac += uint64(binary.BigEndian.Uint32(b[4:8]))
		} else {
			ac += uint64(binary.LittleEndian.Uint32(b[:4]))
			ac += uint64(binary.LittleEndian.Uint32(b[4:8]))
		}
		b = b[8:]
	}
	if len(b) >= 4 {
		if cpu.IsBigEndian {
			ac += uint64(binary.BigEndian.Uint32(b))
		} else {
			ac += uint64(binary.LittleEndian.Uint32(b))
		}
		b = b[4:]
	}
	if len(b) >= 2 {
		if cpu.IsBigEndian {
			ac += uint64(binary.BigEndian.Uint16(b))
		} else {
			ac += uint64(binary.LittleEndian.Uint16(b))
		}
		b = b[2:]
	}
	if len(b) >= 1 {
		if cpu.IsBigEndian {
			ac += uint64(b[0]) << 8
		} else {
			ac += uint64(b[0])
		}
	}

	folded := ipChecksumFold64(ac, 0)
	if !cpu.IsBigEndian {
		folded = bits.ReverseBytes16(folded)
	}
	return folded
}

func ipChecksumFold64(unfolded uint64, initialCarry uint64) uint16 {
	sum, carry := bits.Add32(uint32(unfolded>>32), uint32(unfolded&0xffff_ffff), uint32(initialCarry))
	// if carry != 0, sum <= 0xffff_fffe, otherwise sum <= 0xffff_ffff
	// therefore (sum >> 16) + (sum & 0xffff) + carry <= 0x1_fffe; so there is
	// no need to save the carry flag
	sum = (sum >> 16) + (sum & 0xffff) + carry
	// sum <= 0x1_fffe therefore this is the last fold needed:
	//   if (sum >> 16) > 0 then
	//     (sum >> 16) == 1 && (sum & 0xffff) <= 0xfffe and therefore
	//     the addition will not overflow
	// otherwise (sum >> 16) == 0 and sum will be unchanged
	sum = (sum >> 16) + (sum & 0xffff)
	return uint16(sum)
}

func ipChecksumFold32(unfolded uint32, initialCarry uint32) uint16 {
	sum := (unfolded >> 16) + (unfolded & 0xffff) + initialCarry
	// sum <= 0x1_ffff:
	//   0xffff + 0xffff = 0x1_fffe
	//   initialCarry is 0 or 1, for a combined maximum of 0x1_ffff
	sum = (sum >> 16) + (sum & 0xffff)
	// sum <= 0x1_0000 therefore this is the last fold needed:
	//   if (sum >> 16) > 0 then
	//     (sum >> 16) == 1 && (sum & 0xffff) == 0 and therefore
	//     the addition will not overflow
	// otherwise (sum >> 16) == 0 and sum will be unchanged
	sum = (sum >> 16) + (sum & 0xffff)
	return uint16(sum)
}

func addrPartialChecksum64(addr []byte, initial, carryIn uint64) (sum, carry uint64) {
	sum, carry = initial, carryIn
	switch len(addr) {
	case 4: // IPv4
		if cpu.IsBigEndian {
			sum, carry = bits.Add64(sum, uint64(binary.BigEndian.Uint32(addr)), carry)
		} else {
			sum, carry = bits.Add64(sum, uint64(binary.LittleEndian.Uint32(addr)), carry)
		}
	case 16: // IPv6
		if cpu.IsBigEndian {
			sum, carry = bits.Add64(sum, binary.BigEndian.Uint64(addr), carry)
			sum, carry = bits.Add64(sum, binary.BigEndian.Uint64(addr[8:]), carry)
		} else {
			sum, carry = bits.Add64(sum, binary.LittleEndian.Uint64(addr), carry)
			sum, carry = bits.Add64(sum, binary.LittleEndian.Uint64(addr[8:]), carry)
		}
	default:
		panic("bad addr length")
	}
	return sum, carry
}

func addrPartialChecksum32(addr []byte, initial, carryIn uint32) (sum, carry uint32) {
	sum, carry = initial, carryIn
	switch len(addr) {
	case 4: // IPv4
		if cpu.IsBigEndian {
			sum, carry = bits.Add32(sum, binary.BigEndian.Uint32(addr), carry)
		} else {
			sum, carry = bits.Add32(sum, binary.LittleEndian.Uint32(addr), carry)
		}
	case 16: // IPv6
		if cpu.IsBigEndian {
			sum, carry = bits.Add32(sum, binary.BigEndian.Uint32(addr), carry)
			sum, carry = bits.Add32(sum, binary.BigEndian.Uint32(addr[4:8]), carry)
			sum, carry = bits.Add32(sum, binary.BigEndian.Uint32(addr[8:12]), carry)
			sum, carry = bits.Add32(sum, binary.BigEndian.Uint32(addr[12:16]), carry)
		} else {
			sum, carry = bits.Add32(sum, binary.LittleEndian.Uint32(addr), carry)
			sum, carry = bits.Add32(sum, binary.LittleEndian.Uint32(addr[4:8]), carry)
			sum, carry = bits.Add32(sum, binary.LittleEndian.Uint32(addr[8:12]), carry)
			sum, carry = bits.Add32(sum, binary.LittleEndian.Uint32(addr[12:16]), carry)
		}
	default:
		panic("bad addr length")
	}
	return sum, carry
}

func pseudoHeaderChecksum64(protocol uint8, srcAddr, dstAddr []byte, totalLen uint16) uint16 {
	var sum uint64
	if cpu.IsBigEndian {
		sum = uint64(totalLen) + uint64(protocol)
	} else {
		sum = uint64(bits.ReverseBytes16(totalLen)) + uint64(protocol)<<8
	}
	sum, carry := addrPartialChecksum64(srcAddr, sum, 0)
	sum, carry = addrPartialChecksum64(dstAddr, sum, carry)

	foldedSum := ipChecksumFold64(sum, carry)
	if !cpu.IsBigEndian {
		foldedSum = bits.ReverseBytes16(foldedSum)
	}
	return foldedSum
}

func pseudoHeaderChecksum32(protocol uint8, srcAddr, dstAddr []byte, totalLen uint16) uint16 {
	var sum uint32
	if cpu.IsBigEndian {
		sum = uint32(totalLen) + uint32(protocol)
	} else {
		sum = uint32(bits.ReverseBytes16(totalLen)) + uint32(protocol)<<8
	}
	sum, carry := addrPartialChecksum32(srcAddr, sum, 0)
	sum, carry = addrPartialChecksum32(dstAddr, sum, carry)

	foldedSum := ipChecksumFold32(sum, carry)
	if !cpu.IsBigEndian {
		foldedSum = bits.ReverseBytes16(foldedSum)
	}
	return foldedSum
}

func pseudoHeaderChecksum(protocol uint8, srcAddr, dstAddr []byte, totalLen uint16) uint16 {
	if strconv.IntSize < 64 {
		return pseudoHeaderChecksum32(protocol, srcAddr, dstAddr, totalLen)
	}
	return pseudoHeaderChecksum64(protocol, srcAddr, dstAddr, totalLen)
}
