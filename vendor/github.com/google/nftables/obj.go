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

package nftables

import (
	"encoding/binary"
	"fmt"

	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

var (
	newObjHeaderType = netlink.HeaderType((unix.NFNL_SUBSYS_NFTABLES << 8) | unix.NFT_MSG_NEWOBJ)
	delObjHeaderType = netlink.HeaderType((unix.NFNL_SUBSYS_NFTABLES << 8) | unix.NFT_MSG_DELOBJ)
)

// Obj represents a netfilter stateful object. See also
// https://wiki.nftables.org/wiki-nftables/index.php/Stateful_objects
type Obj interface {
	table() *Table
	family() TableFamily
	unmarshal(*netlink.AttributeDecoder) error
	marshal(data bool) ([]byte, error)
}

// AddObject adds the specified Obj. Alias of AddObj.
func (cc *Conn) AddObject(o Obj) Obj {
	return cc.AddObj(o)
}

// AddObj adds the specified Obj. See also
// https://wiki.nftables.org/wiki-nftables/index.php/Stateful_objects
func (cc *Conn) AddObj(o Obj) Obj {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	data, err := o.marshal(true)
	if err != nil {
		cc.setErr(err)
		return nil
	}

	cc.messages = append(cc.messages, netlink.Message{
		Header: netlink.Header{
			Type:  netlink.HeaderType((unix.NFNL_SUBSYS_NFTABLES << 8) | unix.NFT_MSG_NEWOBJ),
			Flags: netlink.Request | netlink.Acknowledge | netlink.Create,
		},
		Data: append(extraHeader(uint8(o.family()), 0), data...),
	})
	return o
}

// DeleteObject deletes the specified Obj
func (cc *Conn) DeleteObject(o Obj) {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	data, err := o.marshal(false)
	if err != nil {
		cc.setErr(err)
		return
	}

	data = append(data, cc.marshalAttr([]netlink.Attribute{{Type: unix.NLA_F_NESTED | unix.NFTA_OBJ_DATA}})...)

	cc.messages = append(cc.messages, netlink.Message{
		Header: netlink.Header{
			Type:  netlink.HeaderType((unix.NFNL_SUBSYS_NFTABLES << 8) | unix.NFT_MSG_DELOBJ),
			Flags: netlink.Request | netlink.Acknowledge,
		},
		Data: append(extraHeader(uint8(o.family()), 0), data...),
	})
}

// GetObj is a legacy method that return all Obj that belongs
// to the same table as the given one
func (cc *Conn) GetObj(o Obj) ([]Obj, error) {
	return cc.getObj(nil, o.table(), unix.NFT_MSG_GETOBJ)
}

// GetObjReset is a legacy method that reset all Obj that belongs
// the same table as the given one
func (cc *Conn) GetObjReset(o Obj) ([]Obj, error) {
	return cc.getObj(nil, o.table(), unix.NFT_MSG_GETOBJ_RESET)
}

// GetObject gets the specified Object
func (cc *Conn) GetObject(o Obj) (Obj, error) {
	objs, err := cc.getObj(o, o.table(), unix.NFT_MSG_GETOBJ)

	if len(objs) == 0 {
		return nil, err
	}

	return objs[0], err
}

// GetObjects get all the Obj that belongs to the given table
func (cc *Conn) GetObjects(t *Table) ([]Obj, error) {
	return cc.getObj(nil, t, unix.NFT_MSG_GETOBJ)
}

// ResetObject reset the given Obj
func (cc *Conn) ResetObject(o Obj) (Obj, error) {
	objs, err := cc.getObj(o, o.table(), unix.NFT_MSG_GETOBJ_RESET)

	if len(objs) == 0 {
		return nil, err
	}

	return objs[0], err
}

// ResetObjects reset all the Obj that belongs to the given table
func (cc *Conn) ResetObjects(t *Table) ([]Obj, error) {
	return cc.getObj(nil, t, unix.NFT_MSG_GETOBJ_RESET)
}

func objFromMsg(msg netlink.Message) (Obj, error) {
	if got, want1, want2 := msg.Header.Type, newObjHeaderType, delObjHeaderType; got != want1 && got != want2 {
		return nil, fmt.Errorf("unexpected header type: got %v, want %v or %v", got, want1, want2)
	}
	ad, err := netlink.NewAttributeDecoder(msg.Data[4:])
	if err != nil {
		return nil, err
	}
	ad.ByteOrder = binary.BigEndian
	var (
		table      *Table
		name       string
		objectType uint32
	)
	for ad.Next() {
		switch ad.Type() {
		case unix.NFTA_OBJ_TABLE:
			table = &Table{Name: ad.String(), Family: TableFamily(msg.Data[0])}
		case unix.NFTA_OBJ_NAME:
			name = ad.String()
		case unix.NFTA_OBJ_TYPE:
			objectType = ad.Uint32()
		case unix.NFTA_OBJ_DATA:
			switch objectType {
			case unix.NFT_OBJECT_COUNTER:
				o := CounterObj{
					Table: table,
					Name:  name,
				}

				ad.Do(func(b []byte) error {
					ad, err := netlink.NewAttributeDecoder(b)
					if err != nil {
						return err
					}
					ad.ByteOrder = binary.BigEndian
					return o.unmarshal(ad)
				})
				return &o, ad.Err()
			case NFT_OBJECT_QUOTA:
				o := QuotaObj{
					Table: table,
					Name:  name,
				}

				ad.Do(func(b []byte) error {
					ad, err := netlink.NewAttributeDecoder(b)
					if err != nil {
						return err
					}
					ad.ByteOrder = binary.BigEndian
					return o.unmarshal(ad)
				})
				return &o, ad.Err()
			}
		}
	}
	if err := ad.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("malformed stateful object")
}

func (cc *Conn) getObj(o Obj, t *Table, msgType uint16) ([]Obj, error) {
	conn, closer, err := cc.netlinkConn()
	if err != nil {
		return nil, err
	}
	defer func() { _ = closer() }()

	var data []byte
	var flags netlink.HeaderFlags

	if o != nil {
		data, err = o.marshal(false)
	} else {
		flags = netlink.Dump
		data, err = netlink.MarshalAttributes([]netlink.Attribute{
			{Type: unix.NFTA_RULE_TABLE, Data: []byte(t.Name + "\x00")},
		})
	}
	if err != nil {
		return nil, err
	}

	message := netlink.Message{
		Header: netlink.Header{
			Type:  netlink.HeaderType((unix.NFNL_SUBSYS_NFTABLES << 8) | msgType),
			Flags: netlink.Request | netlink.Acknowledge | flags,
		},
		Data: append(extraHeader(uint8(t.Family), 0), data...),
	}

	if _, err := conn.SendMessages([]netlink.Message{message}); err != nil {
		return nil, fmt.Errorf("SendMessages: %v", err)
	}

	reply, err := receiveAckAware(conn, message.Header.Flags)
	if err != nil {
		return nil, fmt.Errorf("Receive: %v", err)
	}
	var objs []Obj
	for _, msg := range reply {
		o, err := objFromMsg(msg)
		if err != nil {
			return nil, err
		}
		objs = append(objs, o)
	}

	return objs, nil
}
