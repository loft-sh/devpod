/*
 *
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

// The ethtool package aims to provide a library that provides easy access
// to the Linux SIOCETHTOOL ioctl operations. It can be used to retrieve information
// from a network device such as statistics, driver related information or even
// the peer of a VETH interface.
package ethtool

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Maximum size of an interface name
const (
	IFNAMSIZ = 16
)

// ioctl ethtool request
const (
	SIOCETHTOOL = 0x8946
)

// ethtool stats related constants.
const (
	ETH_GSTRING_LEN = 32
	ETH_SS_STATS    = 1
	ETH_SS_FEATURES = 4

	// CMD supported
	ETHTOOL_GDRVINFO = 0x00000003 /* Get driver info. */
	ETHTOOL_GSTRINGS = 0x0000001b /* get specified string set */
	ETHTOOL_GSTATS   = 0x0000001d /* get NIC-specific statistics */
	// other CMDs from ethtool-copy.h of ethtool-3.5 package
	ETHTOOL_GSET      = 0x00000001 /* Get settings. */
	ETHTOOL_SSET      = 0x00000002 /* Set settings. */
	ETHTOOL_GMSGLVL   = 0x00000007 /* Get driver message level */
	ETHTOOL_SMSGLVL   = 0x00000008 /* Set driver msg level. */
	ETHTOOL_GCHANNELS = 0x0000003c /* Get no of channels */
	ETHTOOL_SCHANNELS = 0x0000003d /* Set no of channels */
	ETHTOOL_GCOALESCE = 0x0000000e /* Get coalesce config */
	/* Get link status for host, i.e. whether the interface *and* the
	 * physical port (if there is one) are up (ethtool_value). */
	ETHTOOL_GLINK         = 0x0000000a
	ETHTOOL_GMODULEINFO   = 0x00000042 /* Get plug-in module information */
	ETHTOOL_GMODULEEEPROM = 0x00000043 /* Get plug-in module eeprom */
	ETHTOOL_GPERMADDR     = 0x00000020 /* Get permanent hardware address */
	ETHTOOL_GFEATURES     = 0x0000003a /* Get device offload settings */
	ETHTOOL_SFEATURES     = 0x0000003b /* Change device offload settings */
	ETHTOOL_GFLAGS        = 0x00000025 /* Get flags bitmap(ethtool_value) */
	ETHTOOL_GSSET_INFO    = 0x00000037 /* Get string set info */
	ETHTOOL_GET_TS_INFO   = 0x00000041 /* Get time stamping and PHC info */
)

// MAX_GSTRINGS maximum number of stats entries that ethtool can
// retrieve currently.
const (
	MAX_GSTRINGS       = 32768
	MAX_FEATURE_BLOCKS = (MAX_GSTRINGS + 32 - 1) / 32
	EEPROM_LEN         = 640
	PERMADDR_LEN       = 32
)

type ifreq struct {
	ifr_name [IFNAMSIZ]byte
	ifr_data uintptr
}

// following structures comes from uapi/linux/ethtool.h
type ethtoolSsetInfo struct {
	cmd       uint32
	reserved  uint32
	sset_mask uint32
	data      uintptr
}

type ethtoolGetFeaturesBlock struct {
	available     uint32
	requested     uint32
	active        uint32
	never_changed uint32
}

type ethtoolGfeatures struct {
	cmd    uint32
	size   uint32
	blocks [MAX_FEATURE_BLOCKS]ethtoolGetFeaturesBlock
}

type ethtoolSetFeaturesBlock struct {
	valid     uint32
	requested uint32
}

type ethtoolSfeatures struct {
	cmd    uint32
	size   uint32
	blocks [MAX_FEATURE_BLOCKS]ethtoolSetFeaturesBlock
}

type ethtoolDrvInfo struct {
	cmd          uint32
	driver       [32]byte
	version      [32]byte
	fw_version   [32]byte
	bus_info     [32]byte
	erom_version [32]byte
	reserved2    [12]byte
	n_priv_flags uint32
	n_stats      uint32
	testinfo_len uint32
	eedump_len   uint32
	regdump_len  uint32
}

// DrvInfo contains driver information
// ethtool.h v3.5: struct ethtool_drvinfo
type DrvInfo struct {
	Cmd         uint32
	Driver      string
	Version     string
	FwVersion   string
	BusInfo     string
	EromVersion string
	Reserved2   string
	NPrivFlags  uint32
	NStats      uint32
	TestInfoLen uint32
	EedumpLen   uint32
	RegdumpLen  uint32
}

// Channels contains the number of channels for a given interface.
type Channels struct {
	Cmd           uint32
	MaxRx         uint32
	MaxTx         uint32
	MaxOther      uint32
	MaxCombined   uint32
	RxCount       uint32
	TxCount       uint32
	OtherCount    uint32
	CombinedCount uint32
}

// Coalesce is a coalesce config for an interface
type Coalesce struct {
	Cmd                      uint32
	RxCoalesceUsecs          uint32
	RxMaxCoalescedFrames     uint32
	RxCoalesceUsecsIrq       uint32
	RxMaxCoalescedFramesIrq  uint32
	TxCoalesceUsecs          uint32
	TxMaxCoalescedFrames     uint32
	TxCoalesceUsecsIrq       uint32
	TxMaxCoalescedFramesIrq  uint32
	StatsBlockCoalesceUsecs  uint32
	UseAdaptiveRxCoalesce    uint32
	UseAdaptiveTxCoalesce    uint32
	PktRateLow               uint32
	RxCoalesceUsecsLow       uint32
	RxMaxCoalescedFramesLow  uint32
	TxCoalesceUsecsLow       uint32
	TxMaxCoalescedFramesLow  uint32
	PktRateHigh              uint32
	RxCoalesceUsecsHigh      uint32
	RxMaxCoalescedFramesHigh uint32
	TxCoalesceUsecsHigh      uint32
	TxMaxCoalescedFramesHigh uint32
	RateSampleInterval       uint32
}

const (
	SOF_TIMESTAMPING_TX_HARDWARE  = (1 << 0)
	SOF_TIMESTAMPING_TX_SOFTWARE  = (1 << 1)
	SOF_TIMESTAMPING_RX_HARDWARE  = (1 << 2)
	SOF_TIMESTAMPING_RX_SOFTWARE  = (1 << 3)
	SOF_TIMESTAMPING_SOFTWARE     = (1 << 4)
	SOF_TIMESTAMPING_SYS_HARDWARE = (1 << 5)
	SOF_TIMESTAMPING_RAW_HARDWARE = (1 << 6)
	SOF_TIMESTAMPING_OPT_ID       = (1 << 7)
	SOF_TIMESTAMPING_TX_SCHED     = (1 << 8)
	SOF_TIMESTAMPING_TX_ACK       = (1 << 9)
	SOF_TIMESTAMPING_OPT_CMSG     = (1 << 10)
	SOF_TIMESTAMPING_OPT_TSONLY   = (1 << 11)
	SOF_TIMESTAMPING_OPT_STATS    = (1 << 12)
	SOF_TIMESTAMPING_OPT_PKTINFO  = (1 << 13)
	SOF_TIMESTAMPING_OPT_TX_SWHW  = (1 << 14)
	SOF_TIMESTAMPING_BIND_PHC     = (1 << 15)
)

const (
	/*
	 * No outgoing packet will need hardware time stamping;
	 * should a packet arrive which asks for it, no hardware
	 * time stamping will be done.
	 */
	HWTSTAMP_TX_OFF = iota

	/*
	 * Enables hardware time stamping for outgoing packets;
	 * the sender of the packet decides which are to be
	 * time stamped by setting %SOF_TIMESTAMPING_TX_SOFTWARE
	 * before sending the packet.
	 */
	HWTSTAMP_TX_ON

	/*
	 * Enables time stamping for outgoing packets just as
	 * HWTSTAMP_TX_ON does, but also enables time stamp insertion
	 * directly into Sync packets. In this case, transmitted Sync
	 * packets will not received a time stamp via the socket error
	 * queue.
	 */
	HWTSTAMP_TX_ONESTEP_SYNC

	/*
	 * Same as HWTSTAMP_TX_ONESTEP_SYNC, but also enables time
	 * stamp insertion directly into PDelay_Resp packets. In this
	 * case, neither transmitted Sync nor PDelay_Resp packets will
	 * receive a time stamp via the socket error queue.
	 */
	HWTSTAMP_TX_ONESTEP_P2P
)

const (
	HWTSTAMP_FILTER_NONE                = iota /* time stamp no incoming packet at all */
	HWTSTAMP_FILTER_ALL                        /* time stamp any incoming packet */
	HWTSTAMP_FILTER_SOME                       /* return value: time stamp all packets requested plus some others */
	HWTSTAMP_FILTER_PTP_V1_L4_EVENT            /* PTP v1, UDP, any kind of event packet */
	HWTSTAMP_FILTER_PTP_V1_L4_SYNC             /* PTP v1, UDP, Sync packet */
	HWTSTAMP_FILTER_PTP_V1_L4_DELAY_REQ        /* PTP v1, UDP, Delay_req packet */
	HWTSTAMP_FILTER_PTP_V2_L4_EVENT            /* PTP v2, UDP, any kind of event packet */
	HWTSTAMP_FILTER_PTP_V2_L4_SYNC             /* PTP v2, UDP, Sync packet */
	HWTSTAMP_FILTER_PTP_V2_L4_DELAY_REQ        /* PTP v2, UDP, Delay_req packet */
	HWTSTAMP_FILTER_PTP_V2_L2_EVENT            /* 802.AS1, Ethernet, any kind of event packet */
	HWTSTAMP_FILTER_PTP_V2_L2_SYNC             /* 802.AS1, Ethernet, Sync packet */
	HWTSTAMP_FILTER_PTP_V2_L2_DELAY_REQ        /* 802.AS1, Ethernet, Delay_req packet */
	HWTSTAMP_FILTER_PTP_V2_EVENT               /* PTP v2/802.AS1, any layer, any kind of event packet */
	HWTSTAMP_FILTER_PTP_V2_SYNC                /* PTP v2/802.AS1, any layer, Sync packet */
	HWTSTAMP_FILTER_PTP_V2_DELAY_REQ           /* PTP v2/802.AS1, any layer, Delay_req packet */
	HWTSTAMP_FILTER_NTP_ALL                    /* NTP, UDP, all versions and packet modes */
)

type TimestampingInformation struct {
	Cmd            uint32
	SoTimestamping uint32 /* SOF_TIMESTAMPING_* bitmask */
	PhcIndex       int32
	TxTypes        uint32 /* HWTSTAMP_TX_* */
	txReserved     [3]uint32
	RxFilters      uint32 /* HWTSTAMP_FILTER_ */
	rxReserved     [3]uint32
}

type ethtoolGStrings struct {
	cmd        uint32
	string_set uint32
	len        uint32
	data       [MAX_GSTRINGS * ETH_GSTRING_LEN]byte
}

type ethtoolStats struct {
	cmd     uint32
	n_stats uint32
	data    [MAX_GSTRINGS]uint64
}

type ethtoolEeprom struct {
	cmd    uint32
	magic  uint32
	offset uint32
	len    uint32
	data   [EEPROM_LEN]byte
}

type ethtoolModInfo struct {
	cmd        uint32
	tpe        uint32
	eeprom_len uint32
	reserved   [8]uint32
}

type ethtoolLink struct {
	cmd  uint32
	data uint32
}

type ethtoolPermAddr struct {
	cmd  uint32
	size uint32
	data [PERMADDR_LEN]byte
}

type Ethtool struct {
	fd int
}

// Convert zero-terminated array of chars (string in C) to a Go string.
func goString(s []byte) string {
	strEnd := bytes.IndexByte(s, 0)
	if strEnd == -1 {
		return string(s[:])
	}
	return string(s[:strEnd])
}

// DriverName returns the driver name of the given interface name.
func (e *Ethtool) DriverName(intf string) (string, error) {
	info, err := e.getDriverInfo(intf)
	if err != nil {
		return "", err
	}
	return goString(info.driver[:]), nil
}

// BusInfo returns the bus information of the given interface name.
func (e *Ethtool) BusInfo(intf string) (string, error) {
	info, err := e.getDriverInfo(intf)
	if err != nil {
		return "", err
	}
	return goString(info.bus_info[:]), nil
}

// ModuleEeprom returns Eeprom information of the given interface name.
func (e *Ethtool) ModuleEeprom(intf string) ([]byte, error) {
	eeprom, _, err := e.getModuleEeprom(intf)
	if err != nil {
		return nil, err
	}

	return eeprom.data[:eeprom.len], nil
}

// ModuleEeprom returns Eeprom information of the given interface name.
func (e *Ethtool) ModuleEepromHex(intf string) (string, error) {
	eeprom, _, err := e.getModuleEeprom(intf)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(eeprom.data[:eeprom.len]), nil
}

// DriverInfo returns driver information of the given interface name.
func (e *Ethtool) DriverInfo(intf string) (DrvInfo, error) {
	i, err := e.getDriverInfo(intf)
	if err != nil {
		return DrvInfo{}, err
	}

	drvInfo := DrvInfo{
		Cmd:         i.cmd,
		Driver:      goString(i.driver[:]),
		Version:     goString(i.version[:]),
		FwVersion:   goString(i.fw_version[:]),
		BusInfo:     goString(i.bus_info[:]),
		EromVersion: goString(i.erom_version[:]),
		Reserved2:   goString(i.reserved2[:]),
		NPrivFlags:  i.n_priv_flags,
		NStats:      i.n_stats,
		TestInfoLen: i.testinfo_len,
		EedumpLen:   i.eedump_len,
		RegdumpLen:  i.regdump_len,
	}

	return drvInfo, nil
}

// GetChannels returns the number of channels for the given interface name.
func (e *Ethtool) GetChannels(intf string) (Channels, error) {
	channels, err := e.getChannels(intf)
	if err != nil {
		return Channels{}, err
	}

	return channels, nil
}

// SetChannels sets the number of channels for the given interface name and
// returns the new number of channels.
func (e *Ethtool) SetChannels(intf string, channels Channels) (Channels, error) {
	channels, err := e.setChannels(intf, channels)
	if err != nil {
		return Channels{}, err
	}

	return channels, nil
}

// GetCoalesce returns the coalesce config for the given interface name.
func (e *Ethtool) GetCoalesce(intf string) (Coalesce, error) {
	coalesce, err := e.getCoalesce(intf)
	if err != nil {
		return Coalesce{}, err
	}
	return coalesce, nil
}

// GetTimestampingInformation returns the PTP timestamping information for the given interface name.
func (e *Ethtool) GetTimestampingInformation(intf string) (TimestampingInformation, error) {
	ts, err := e.getTimestampingInformation(intf)
	if err != nil {
		return TimestampingInformation{}, err
	}
	return ts, nil
}

// PermAddr returns permanent address of the given interface name.
func (e *Ethtool) PermAddr(intf string) (string, error) {
	permAddr, err := e.getPermAddr(intf)
	if err != nil {
		return "", err
	}

	if permAddr.data[0] == 0 && permAddr.data[1] == 0 &&
		permAddr.data[2] == 0 && permAddr.data[3] == 0 &&
		permAddr.data[4] == 0 && permAddr.data[5] == 0 {
		return "", nil
	}

	return fmt.Sprintf("%x:%x:%x:%x:%x:%x",
		permAddr.data[0:1],
		permAddr.data[1:2],
		permAddr.data[2:3],
		permAddr.data[3:4],
		permAddr.data[4:5],
		permAddr.data[5:6],
	), nil
}

func (e *Ethtool) ioctl(intf string, data uintptr) error {
	var name [IFNAMSIZ]byte
	copy(name[:], []byte(intf))

	ifr := ifreq{
		ifr_name: name,
		ifr_data: data,
	}

	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(e.fd), SIOCETHTOOL, uintptr(unsafe.Pointer(&ifr)))
	if ep != 0 {
		return ep
	}

	return nil
}

func (e *Ethtool) getDriverInfo(intf string) (ethtoolDrvInfo, error) {
	drvinfo := ethtoolDrvInfo{
		cmd: ETHTOOL_GDRVINFO,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&drvinfo))); err != nil {
		return ethtoolDrvInfo{}, err
	}

	return drvinfo, nil
}

func (e *Ethtool) getChannels(intf string) (Channels, error) {
	channels := Channels{
		Cmd: ETHTOOL_GCHANNELS,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&channels))); err != nil {
		return Channels{}, err
	}

	return channels, nil
}

func (e *Ethtool) setChannels(intf string, channels Channels) (Channels, error) {
	channels.Cmd = ETHTOOL_SCHANNELS

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&channels))); err != nil {
		return Channels{}, err
	}

	return channels, nil
}

func (e *Ethtool) getCoalesce(intf string) (Coalesce, error) {
	coalesce := Coalesce{
		Cmd: ETHTOOL_GCOALESCE,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&coalesce))); err != nil {
		return Coalesce{}, err
	}

	return coalesce, nil
}

func (e *Ethtool) getTimestampingInformation(intf string) (TimestampingInformation, error) {
	ts := TimestampingInformation{
		Cmd: ETHTOOL_GET_TS_INFO,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&ts))); err != nil {
		return TimestampingInformation{}, err
	}

	return ts, nil
}

func (e *Ethtool) getPermAddr(intf string) (ethtoolPermAddr, error) {
	permAddr := ethtoolPermAddr{
		cmd:  ETHTOOL_GPERMADDR,
		size: PERMADDR_LEN,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&permAddr))); err != nil {
		return ethtoolPermAddr{}, err
	}

	return permAddr, nil
}

func (e *Ethtool) getModuleEeprom(intf string) (ethtoolEeprom, ethtoolModInfo, error) {
	modInfo := ethtoolModInfo{
		cmd: ETHTOOL_GMODULEINFO,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&modInfo))); err != nil {
		return ethtoolEeprom{}, ethtoolModInfo{}, err
	}

	eeprom := ethtoolEeprom{
		cmd:    ETHTOOL_GMODULEEEPROM,
		len:    modInfo.eeprom_len,
		offset: 0,
	}

	if modInfo.eeprom_len > EEPROM_LEN {
		return ethtoolEeprom{}, ethtoolModInfo{}, fmt.Errorf("eeprom size: %d is larger than buffer size: %d", modInfo.eeprom_len, EEPROM_LEN)
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&eeprom))); err != nil {
		return ethtoolEeprom{}, ethtoolModInfo{}, err
	}

	return eeprom, modInfo, nil
}

func isFeatureBitSet(blocks [MAX_FEATURE_BLOCKS]ethtoolGetFeaturesBlock, index uint) bool {
	return (blocks)[index/32].active&(1<<(index%32)) != 0
}

func setFeatureBit(blocks *[MAX_FEATURE_BLOCKS]ethtoolSetFeaturesBlock, index uint, value bool) {
	blockIndex, bitIndex := index/32, index%32

	blocks[blockIndex].valid |= 1 << bitIndex

	if value {
		blocks[blockIndex].requested |= 1 << bitIndex
	} else {
		blocks[blockIndex].requested &= ^(1 << bitIndex)
	}
}

// FeatureNames shows supported features by their name.
func (e *Ethtool) FeatureNames(intf string) (map[string]uint, error) {
	ssetInfo := ethtoolSsetInfo{
		cmd:       ETHTOOL_GSSET_INFO,
		sset_mask: 1 << ETH_SS_FEATURES,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&ssetInfo))); err != nil {
		return nil, err
	}

	length := uint32(ssetInfo.data)
	if length == 0 {
		return map[string]uint{}, nil
	} else if length > MAX_GSTRINGS {
		return nil, fmt.Errorf("ethtool currently doesn't support more than %d entries, received %d", MAX_GSTRINGS, length)
	}

	gstrings := ethtoolGStrings{
		cmd:        ETHTOOL_GSTRINGS,
		string_set: ETH_SS_FEATURES,
		len:        length,
		data:       [MAX_GSTRINGS * ETH_GSTRING_LEN]byte{},
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&gstrings))); err != nil {
		return nil, err
	}

	var result = make(map[string]uint)
	for i := 0; i != int(length); i++ {
		b := gstrings.data[i*ETH_GSTRING_LEN : i*ETH_GSTRING_LEN+ETH_GSTRING_LEN]
		key := goString(b)
		if key != "" {
			result[key] = uint(i)
		}
	}

	return result, nil
}

// Features retrieves features of the given interface name.
func (e *Ethtool) Features(intf string) (map[string]bool, error) {
	names, err := e.FeatureNames(intf)
	if err != nil {
		return nil, err
	}

	length := uint32(len(names))
	if length == 0 {
		return map[string]bool{}, nil
	}

	features := ethtoolGfeatures{
		cmd:  ETHTOOL_GFEATURES,
		size: (length + 32 - 1) / 32,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&features))); err != nil {
		return nil, err
	}

	var result = make(map[string]bool, length)
	for key, index := range names {
		result[key] = isFeatureBitSet(features.blocks, index)
	}

	return result, nil
}

// Change requests a change in the given device's features.
func (e *Ethtool) Change(intf string, config map[string]bool) error {
	names, err := e.FeatureNames(intf)
	if err != nil {
		return err
	}

	length := uint32(len(names))

	features := ethtoolSfeatures{
		cmd:  ETHTOOL_SFEATURES,
		size: (length + 32 - 1) / 32,
	}

	for key, value := range config {
		if index, ok := names[key]; ok {
			setFeatureBit(&features.blocks, index, value)
		} else {
			return fmt.Errorf("unsupported feature %q", key)
		}
	}

	return e.ioctl(intf, uintptr(unsafe.Pointer(&features)))
}

// Get state of a link.
func (e *Ethtool) LinkState(intf string) (uint32, error) {
	x := ethtoolLink{
		cmd: ETHTOOL_GLINK,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&x))); err != nil {
		return 0, err
	}

	return x.data, nil
}

// Stats retrieves stats of the given interface name.
func (e *Ethtool) Stats(intf string) (map[string]uint64, error) {
	drvinfo := ethtoolDrvInfo{
		cmd: ETHTOOL_GDRVINFO,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&drvinfo))); err != nil {
		return nil, err
	}

	if drvinfo.n_stats*ETH_GSTRING_LEN > MAX_GSTRINGS*ETH_GSTRING_LEN {
		return nil, fmt.Errorf("ethtool currently doesn't support more than %d entries, received %d", MAX_GSTRINGS, drvinfo.n_stats)
	}

	gstrings := ethtoolGStrings{
		cmd:        ETHTOOL_GSTRINGS,
		string_set: ETH_SS_STATS,
		len:        drvinfo.n_stats,
		data:       [MAX_GSTRINGS * ETH_GSTRING_LEN]byte{},
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&gstrings))); err != nil {
		return nil, err
	}

	stats := ethtoolStats{
		cmd:     ETHTOOL_GSTATS,
		n_stats: drvinfo.n_stats,
		data:    [MAX_GSTRINGS]uint64{},
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&stats))); err != nil {
		return nil, err
	}

	var result = make(map[string]uint64)
	for i := 0; i != int(drvinfo.n_stats); i++ {
		b := gstrings.data[i*ETH_GSTRING_LEN : i*ETH_GSTRING_LEN+ETH_GSTRING_LEN]
		strEnd := strings.Index(string(b), "\x00")
		if strEnd == -1 {
			strEnd = ETH_GSTRING_LEN
		}
		key := string(b[:strEnd])
		if len(key) != 0 {
			result[key] = stats.data[i]
		}
	}

	return result, nil
}

// Close closes the ethool handler
func (e *Ethtool) Close() {
	unix.Close(e.fd)
}

// NewEthtool returns a new ethtool handler
func NewEthtool() (*Ethtool, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_IP)
	if err != nil {
		return nil, err
	}

	return &Ethtool{
		fd: int(fd),
	}, nil
}

// BusInfo returns bus information of the given interface name.
func BusInfo(intf string) (string, error) {
	e, err := NewEthtool()
	if err != nil {
		return "", err
	}
	defer e.Close()
	return e.BusInfo(intf)
}

// DriverName returns the driver name of the given interface name.
func DriverName(intf string) (string, error) {
	e, err := NewEthtool()
	if err != nil {
		return "", err
	}
	defer e.Close()
	return e.DriverName(intf)
}

// Stats retrieves stats of the given interface name.
func Stats(intf string) (map[string]uint64, error) {
	e, err := NewEthtool()
	if err != nil {
		return nil, err
	}
	defer e.Close()
	return e.Stats(intf)
}

// PermAddr returns permanent address of the given interface name.
func PermAddr(intf string) (string, error) {
	e, err := NewEthtool()
	if err != nil {
		return "", err
	}
	defer e.Close()
	return e.PermAddr(intf)
}
