// Copyright 2018 Google LLC. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package expr

import (
	"encoding/binary"

	"github.com/google/nftables/binaryutil"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

// CtKey specifies which piece of conntrack information should be loaded. See
// also https://wiki.nftables.org/wiki-nftables/index.php/Matching_connection_tracking_stateful_metainformation
type CtKey uint32

// Possible CtKey values.
const (
	CtKeySTATE      CtKey = unix.NFT_CT_STATE
	CtKeyDIRECTION  CtKey = unix.NFT_CT_DIRECTION
	CtKeySTATUS     CtKey = unix.NFT_CT_STATUS
	CtKeyMARK       CtKey = unix.NFT_CT_MARK
	CtKeySECMARK    CtKey = unix.NFT_CT_SECMARK
	CtKeyEXPIRATION CtKey = unix.NFT_CT_EXPIRATION
	CtKeyHELPER     CtKey = unix.NFT_CT_HELPER
	CtKeyL3PROTOCOL CtKey = unix.NFT_CT_L3PROTOCOL
	CtKeySRC        CtKey = unix.NFT_CT_SRC
	CtKeyDST        CtKey = unix.NFT_CT_DST
	CtKeyPROTOCOL   CtKey = unix.NFT_CT_PROTOCOL
	CtKeyPROTOSRC   CtKey = unix.NFT_CT_PROTO_SRC
	CtKeyPROTODST   CtKey = unix.NFT_CT_PROTO_DST
	CtKeyLABELS     CtKey = unix.NFT_CT_LABELS
	CtKeyPKTS       CtKey = unix.NFT_CT_PKTS
	CtKeyBYTES      CtKey = unix.NFT_CT_BYTES
	CtKeyAVGPKT     CtKey = unix.NFT_CT_AVGPKT
	CtKeyZONE       CtKey = unix.NFT_CT_ZONE
	CtKeyEVENTMASK  CtKey = unix.NFT_CT_EVENTMASK

	// https://sources.debian.org/src//nftables/0.9.8-3/src/ct.c/?hl=39#L39
	CtStateBitINVALID     uint32 = 1
	CtStateBitESTABLISHED uint32 = 2
	CtStateBitRELATED     uint32 = 4
	CtStateBitNEW         uint32 = 8
	CtStateBitUNTRACKED   uint32 = 64
)

// Ct defines type for NFT connection tracking
type Ct struct {
	Register       uint32
	SourceRegister bool
	Key            CtKey
}

func (e *Ct) marshal(fam byte) ([]byte, error) {
	regData := []byte{}
	exprData, err := netlink.MarshalAttributes(
		[]netlink.Attribute{
			{Type: unix.NFTA_CT_KEY, Data: binaryutil.BigEndian.PutUint32(uint32(e.Key))},
		},
	)
	if err != nil {
		return nil, err
	}
	if e.SourceRegister {
		regData, err = netlink.MarshalAttributes(
			[]netlink.Attribute{
				{Type: unix.NFTA_CT_SREG, Data: binaryutil.BigEndian.PutUint32(e.Register)},
			},
		)
	} else {
		regData, err = netlink.MarshalAttributes(
			[]netlink.Attribute{
				{Type: unix.NFTA_CT_DREG, Data: binaryutil.BigEndian.PutUint32(e.Register)},
			},
		)
	}
	if err != nil {
		return nil, err
	}
	exprData = append(exprData, regData...)

	return netlink.MarshalAttributes([]netlink.Attribute{
		{Type: unix.NFTA_EXPR_NAME, Data: []byte("ct\x00")},
		{Type: unix.NLA_F_NESTED | unix.NFTA_EXPR_DATA, Data: exprData},
	})
}

func (e *Ct) unmarshal(fam byte, data []byte) error {
	ad, err := netlink.NewAttributeDecoder(data)
	if err != nil {
		return err
	}
	ad.ByteOrder = binary.BigEndian
	for ad.Next() {
		switch ad.Type() {
		case unix.NFTA_CT_KEY:
			e.Key = CtKey(ad.Uint32())
		case unix.NFTA_CT_DREG:
			e.Register = ad.Uint32()
		}
	}
	return ad.Err()
}
