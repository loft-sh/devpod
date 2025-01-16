// Copyright 2023 Google LLC. All Rights Reserved.
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

package nftables

import (
	"github.com/google/nftables/binaryutil"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

const (
	NFTA_OBJ_USERDATA = 8
	NFT_OBJECT_QUOTA  = 2
)

type QuotaObj struct {
	Table    *Table
	Name     string
	Bytes    uint64
	Consumed uint64
	Over     bool
}

func (q *QuotaObj) unmarshal(ad *netlink.AttributeDecoder) error {
	for ad.Next() {
		switch ad.Type() {
		case unix.NFTA_QUOTA_BYTES:
			q.Bytes = ad.Uint64()
		case unix.NFTA_QUOTA_CONSUMED:
			q.Consumed = ad.Uint64()
		case unix.NFTA_QUOTA_FLAGS:
			q.Over = (ad.Uint32() & unix.NFT_QUOTA_F_INV) == 1
		}
	}
	return nil
}

func (q *QuotaObj) marshal(data bool) ([]byte, error) {
	flags := uint32(0)
	if q.Over {
		flags = unix.NFT_QUOTA_F_INV
	}
	obj, err := netlink.MarshalAttributes([]netlink.Attribute{
		{Type: unix.NFTA_QUOTA_BYTES, Data: binaryutil.BigEndian.PutUint64(q.Bytes)},
		{Type: unix.NFTA_QUOTA_CONSUMED, Data: binaryutil.BigEndian.PutUint64(q.Consumed)},
		{Type: unix.NFTA_QUOTA_FLAGS, Data: binaryutil.BigEndian.PutUint32(flags)},
	})
	if err != nil {
		return nil, err
	}
	attrs := []netlink.Attribute{
		{Type: unix.NFTA_OBJ_TABLE, Data: []byte(q.Table.Name + "\x00")},
		{Type: unix.NFTA_OBJ_NAME, Data: []byte(q.Name + "\x00")},
		{Type: unix.NFTA_OBJ_TYPE, Data: binaryutil.BigEndian.PutUint32(NFT_OBJECT_QUOTA)},
	}
	if data {
		attrs = append(attrs, netlink.Attribute{Type: unix.NLA_F_NESTED | unix.NFTA_OBJ_DATA, Data: obj})
	}
	return netlink.MarshalAttributes(attrs)
}

func (q *QuotaObj) table() *Table {
	return q.Table
}

func (q *QuotaObj) family() TableFamily {
	return q.Table.Family
}
