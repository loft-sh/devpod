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

const (
	// NFTA_TPROXY_FAMILY defines attribute for a table family
	NFTA_TPROXY_FAMILY = 0x01
	// NFTA_TPROXY_REG defines attribute for a register carrying redirection port value
	NFTA_TPROXY_REG = 0x03
)

// TProxy defines struct with parameters for the transparent proxy
type TProxy struct {
	Family      byte
	TableFamily byte
	RegPort     uint32
}

func (e *TProxy) marshal(fam byte) ([]byte, error) {
	data, err := netlink.MarshalAttributes([]netlink.Attribute{
		{Type: NFTA_TPROXY_FAMILY, Data: binaryutil.BigEndian.PutUint32(uint32(e.Family))},
		{Type: NFTA_TPROXY_REG, Data: binaryutil.BigEndian.PutUint32(e.RegPort)},
	})
	if err != nil {
		return nil, err
	}
	return netlink.MarshalAttributes([]netlink.Attribute{
		{Type: unix.NFTA_EXPR_NAME, Data: []byte("tproxy\x00")},
		{Type: unix.NLA_F_NESTED | unix.NFTA_EXPR_DATA, Data: data},
	})
}

func (e *TProxy) unmarshal(fam byte, data []byte) error {
	ad, err := netlink.NewAttributeDecoder(data)
	if err != nil {
		return err
	}
	ad.ByteOrder = binary.BigEndian
	for ad.Next() {
		switch ad.Type() {
		case NFTA_TPROXY_FAMILY:
			e.Family = ad.Uint8()
		case NFTA_TPROXY_REG:
			e.RegPort = ad.Uint32()
		}
	}
	return ad.Err()
}
