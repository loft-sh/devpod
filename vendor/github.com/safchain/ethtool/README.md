# ethtool go package #

![Build Status](https://github.com/safchain/ethtool/actions/workflows/unittests.yml/badge.svg)
[![GoDoc](https://godoc.org/github.com/safchain/ethtool?status.svg)](https://godoc.org/github.com/safchain/ethtool)


The ethtool package aims to provide a library that provides easy access to the Linux SIOCETHTOOL ioctl operations. It can be used to retrieve information from a network device such as statistics, driver related information or even the peer of a VETH interface.

# Installation

```shell
go get github.com/safchain/ethtool
```

# How to use

```go
package main

import (
	"fmt"

	"github.com/safchain/ethtool"
)

func main() {
	ethHandle, err := ethtool.NewEthtool()
	if err != nil {
		panic(err.Error())
	}
	defer ethHandle.Close()

	// Retrieve tx from eth0
	stats, err := ethHandle.Stats("eth0")
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("TX: %d\n", stats["tx_bytes"])

	// Retrieve peer index of a veth interface
	stats, err = ethHandle.Stats("veth0")
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("Peer Index: %d\n", stats["peer_ifindex"])
}
```

## LICENSE ##

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
