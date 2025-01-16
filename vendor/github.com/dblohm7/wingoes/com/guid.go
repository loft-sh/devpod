// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package com

import (
	"github.com/dblohm7/wingoes"
)

// We intentionally export these types across all GOOSes

// IID is a GUID that represents an interface ID.
type IID wingoes.GUID

// CLSID is a GUID that represents a class ID.
type CLSID wingoes.GUID

// AppID is a GUID that represents an application ID.
type AppID wingoes.GUID

// ServiceID is a GUID that represents a service ID.
type ServiceID wingoes.GUID
