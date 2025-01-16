# certstore

Certstore is a Go library for accessing user identities stored in platform certificate stores. On Windows and macOS, certstore can enumerate user identities and sign messages with their private keys.

## Fork from original module

This is a fork from the [cyolosecurity fork](https://github.com/cyolosecurity/certstore/)
of the the original [certstore module](https://github.com/github/certstore/). The cyolosecurity
fork adds some functionality that we require (RSA-PSS support and ability to use the machine
certificate store instead of the current user certificate store). However, the cyolosecurity
fork did not update the module name, so we are unable to import it directly, thus leading to
the Tailscale fork.

As of this writing, the [RSA-PSS PR](https://github.com/github/certstore/pull/18) has been
under review for several months and the machine certificate store change has not yet been sent
to the github maintainer for review. Ideally these changes will make their way back into the
original module and we can
[discontinue this fork](https://github.com/tailscale/tailscale/issues/2005) at some point in
the future.

## Example

```go
package main

import (
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"

	"crypto/rand"
	"crypto/sha256"

	"github.com/github/certstore"
)

func main() {
	sig, err := signWithMyIdentity("Ben Toews", "hello, world!")
	if err != nil {
		panic(err)
	}

	fmt.Println(hex.EncodeToString(sig))
}

func signWithMyIdentity(cn, msg string) ([]byte, error) {
	// Open the certificate store for use. This must be Close()'ed once you're
	// finished with the store and any identities it contains.
	store, err := certstore.Open()
	if err != nil {
		return nil, err
	}
	defer store.Close()

	// Get an Identity slice, containing every identity in the store. Each of
	// these must be Close()'ed when you're done with them.
	idents, err := store.Identities()
	if err != nil {
		return nil, err
	}

	// Iterate through the identities, looking for the one we want.
	var me certstore.Identity
	for _, ident := range idents {
		defer ident.Close()

		crt, errr := ident.Certificate()
		if errr != nil {
			return nil, errr
		}

		if crt.Subject.CommonName == "Ben Toews" {
			me = ident
		}
	}

	if me == nil {
		return nil, errors.New("Couldn't find my identity")
	}

	// Get a crypto.Signer for the identity.
	signer, err := me.Signer()
	if err != nil {
		return nil, err
	}

	// Digest and sign our message.
	digest := sha256.Sum256([]byte(msg))
	signature, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

```
