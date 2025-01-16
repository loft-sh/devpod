package tun

import "golang.org/x/sys/cpu"

// checksum computes an IP checksum starting with the provided initial value.
// The length of data should be at least 128 bytes for best performance. Smaller
// buffers will still compute a correct result. For best performance with
// smaller buffers, use shortChecksum().
var checksum = checksumAMD64

func init() {
	if cpu.X86.HasAVX && cpu.X86.HasAVX2 && cpu.X86.HasBMI2 {
		checksum = checksumAVX2
		return
	}
	if cpu.X86.HasSSE2 {
		checksum = checksumSSE2
		return
	}
}
