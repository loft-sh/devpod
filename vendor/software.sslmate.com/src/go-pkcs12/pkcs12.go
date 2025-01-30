// Copyright 2015, 2018, 2019 Opsmate, Inc. All rights reserved.
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pkcs12 implements some of PKCS#12 (also known as P12 or PFX).
// It is intended for decoding DER-encoded P12/PFX files for use with the crypto/tls
// package, and for encoding P12/PFX files for use by legacy applications which
// do not support newer formats.  Since PKCS#12 uses weak encryption
// primitives, it SHOULD NOT be used for new applications.
//
// Note that only DER-encoded PKCS#12 files are supported, even though PKCS#12
// allows BER encoding.  This is because encoding/asn1 only supports DER.
//
// This package is forked from golang.org/x/crypto/pkcs12, which is frozen.
// The implementation is distilled from https://tools.ietf.org/html/rfc7292
// and referenced documents.
package pkcs12 // import "software.sslmate.com/src/go-pkcs12"

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
)

// DefaultPassword is the string "changeit", a commonly-used password for
// PKCS#12 files.
const DefaultPassword = "changeit"

// An Encoder contains methods for encoding PKCS#12 files.  This package
// defines several different Encoders with different parameters.
type Encoder struct {
	macAlgorithm         asn1.ObjectIdentifier
	certAlgorithm        asn1.ObjectIdentifier
	keyAlgorithm         asn1.ObjectIdentifier
	macIterations        int
	encryptionIterations int
	saltLen              int
	rand                 io.Reader
}

// WithIterations creates a new Encoder identical to enc except that
// it will use the given number of KDF iterations for deriving the MAC
// and encryption keys.
//
// Note that even with a large number of iterations, a weak
// password can still be brute-forced in much less time than it would
// take to brute-force a high-entropy encrytion key.  For the best
// security, don't worry about the number of iterations and just
// use a high-entropy password (e.g. one generated with `openssl rand -hex 16`).
// See https://neilmadden.blog/2023/01/09/on-pbkdf2-iterations/ for more detail.
//
// Panics if iterations is less than 1.
func (enc Encoder) WithIterations(iterations int) *Encoder {
	if iterations < 1 {
		panic("pkcs12: number of iterations is less than 1")
	}
	enc.macIterations = iterations
	enc.encryptionIterations = iterations
	return &enc
}

// WithRand creates a new Encoder identical to enc except that
// it will use the given io.Reader for its random number generator
// instead of [crypto/rand.Reader].
func (enc Encoder) WithRand(rand io.Reader) *Encoder {
	enc.rand = rand
	return &enc
}

// LegacyRC2 encodes PKCS#12 files using weak algorithms that were
// traditionally used in PKCS#12 files, including those produced
// by OpenSSL before 3.0.0, go-pkcs12 before 0.3.0, and Java when
// keystore.pkcs12.legacy is defined.  Specifically, certificates
// are encrypted using PBE with RC2, and keys are encrypted using PBE
// with 3DES, using keys derived with 2048 iterations of HMAC-SHA-1.
// MACs use HMAC-SHA-1 with keys derived with 1 iteration of HMAC-SHA-1.
//
// Due to the weak encryption, it is STRONGLY RECOMMENDED that you use [DefaultPassword]
// when encoding PKCS#12 files using this encoder, and protect the PKCS#12 files
// using other means.
//
// By default, OpenSSL 3 can't decode PKCS#12 files created using this encoder.
// For better compatibility, use [LegacyDES].  For better security, use
// [Modern2023].
var LegacyRC2 = &Encoder{
	macAlgorithm:         oidSHA1,
	certAlgorithm:        oidPBEWithSHAAnd40BitRC2CBC,
	keyAlgorithm:         oidPBEWithSHAAnd3KeyTripleDESCBC,
	macIterations:        1,
	encryptionIterations: 2048,
	saltLen:              8,
	rand:                 rand.Reader,
}

// LegacyDES encodes PKCS#12 files using weak algorithms that are
// supported by a wide variety of software.  Certificates and keys
// are encrypted using PBE with 3DES using keys derived with 2048
// iterations of HMAC-SHA-1.  MACs use HMAC-SHA-1 with keys derived
// with 1 iteration of HMAC-SHA-1.  These are the same parameters
// used by OpenSSL's -descert option.  As of 2023, this encoder is
// likely to produce files that can be read by the most software.
//
// Due to the weak encryption, it is STRONGLY RECOMMENDED that you use [DefaultPassword]
// when encoding PKCS#12 files using this encoder, and protect the PKCS#12 files
// using other means.  To create more secure PKCS#12 files, use [Modern2023].
var LegacyDES = &Encoder{
	macAlgorithm:         oidSHA1,
	certAlgorithm:        oidPBEWithSHAAnd3KeyTripleDESCBC,
	keyAlgorithm:         oidPBEWithSHAAnd3KeyTripleDESCBC,
	macIterations:        1,
	encryptionIterations: 2048,
	saltLen:              8,
	rand:                 rand.Reader,
}

// Passwordless encodes PKCS#12 files without any encryption or MACs.
// A lot of software has trouble reading such files, so it's probably only
// useful for creating Java trust stores using [Encoder.EncodeTrustStore]
// or [Encoder.EncodeTrustStoreEntries].
//
// When using this encoder, you MUST specify an empty password.
var Passwordless = &Encoder{
	macAlgorithm:  nil,
	certAlgorithm: nil,
	keyAlgorithm:  nil,
	rand:          rand.Reader,
}

// Modern2023 encodes PKCS#12 files using algorithms that are considered modern
// as of 2023.  Private keys and certificates are encrypted using PBES2 with
// PBKDF2-HMAC-SHA-256 and AES-256-CBC.  The MAC algorithm is HMAC-SHA-2.  These
// are the same algorithms used by OpenSSL 3 (by default), Java 20 (by default),
// and Windows Server 2019 (when "stronger" is used).
//
// Files produced with this encoder can be read by OpenSSL 1.1.1 and higher,
// Java 12 and higher, and Windows Server 2019 and higher.
//
// For passwords, it is RECOMMENDED that you do one of the following:
// 1) Use [DefaultPassword] and protect the file using other means, or
// 2) Use a high-entropy password, such as one generated with `openssl rand -hex 16`.
//
// You SHOULD NOT use a lower-entropy password with this encoder because the number of KDF
// iterations is only 2048 and doesn't provide meaningful protection against
// brute-forcing.  You can increase the number of iterations using [Encoder.WithIterations],
// but as https://neilmadden.blog/2023/01/09/on-pbkdf2-iterations/ explains, this doesn't
// help as much as you think.
var Modern2023 = &Encoder{
	macAlgorithm:         oidSHA256,
	certAlgorithm:        oidPBES2,
	keyAlgorithm:         oidPBES2,
	macIterations:        2048,
	encryptionIterations: 2048,
	saltLen:              16,
	rand:                 rand.Reader,
}

// Legacy encodes PKCS#12 files using weak, legacy parameters that work in
// a wide variety of software.
//
// Currently, this encoder is the same as [LegacyDES], but this
// may change in the future if another encoder is found to provide better
// compatibility.
//
// Due to the weak encryption, it is STRONGLY RECOMMENDED that you use [DefaultPassword]
// when encoding PKCS#12 files using this encoder, and protect the PKCS#12 files
// using other means.
var Legacy = LegacyDES

// Modern encodes PKCS#12 files using modern, robust parameters.
//
// Currently, this encoder is the same as [Modern2023], but this
// may change in the future to keep up with modern practices.
var Modern = Modern2023

var (
	oidDataContentType          = asn1.ObjectIdentifier([]int{1, 2, 840, 113549, 1, 7, 1})
	oidEncryptedDataContentType = asn1.ObjectIdentifier([]int{1, 2, 840, 113549, 1, 7, 6})

	oidFriendlyName     = asn1.ObjectIdentifier([]int{1, 2, 840, 113549, 1, 9, 20})
	oidLocalKeyID       = asn1.ObjectIdentifier([]int{1, 2, 840, 113549, 1, 9, 21})
	oidMicrosoftCSPName = asn1.ObjectIdentifier([]int{1, 3, 6, 1, 4, 1, 311, 17, 1})

	oidJavaTrustStore      = asn1.ObjectIdentifier([]int{2, 16, 840, 1, 113894, 746875, 1, 1})
	oidAnyExtendedKeyUsage = asn1.ObjectIdentifier([]int{2, 5, 29, 37, 0})
)

type pfxPdu struct {
	Version  int
	AuthSafe contentInfo
	MacData  macData `asn1:"optional"`
}

type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"tag:0,explicit,optional"`
}

type encryptedData struct {
	Version              int
	EncryptedContentInfo encryptedContentInfo
}

type encryptedContentInfo struct {
	ContentType                asn1.ObjectIdentifier
	ContentEncryptionAlgorithm pkix.AlgorithmIdentifier
	EncryptedContent           []byte `asn1:"tag:0,optional"`
}

func (i encryptedContentInfo) Algorithm() pkix.AlgorithmIdentifier {
	return i.ContentEncryptionAlgorithm
}

func (i encryptedContentInfo) Data() []byte { return i.EncryptedContent }

func (i *encryptedContentInfo) SetData(data []byte) { i.EncryptedContent = data }

type safeBag struct {
	Id         asn1.ObjectIdentifier
	Value      asn1.RawValue     `asn1:"tag:0,explicit"`
	Attributes []pkcs12Attribute `asn1:"set,optional"`
}

func (bag *safeBag) hasAttribute(id asn1.ObjectIdentifier) bool {
	for _, attr := range bag.Attributes {
		if attr.Id.Equal(id) {
			return true
		}
	}
	return false
}

type pkcs12Attribute struct {
	Id    asn1.ObjectIdentifier
	Value asn1.RawValue `asn1:"set"`
}

type encryptedPrivateKeyInfo struct {
	AlgorithmIdentifier pkix.AlgorithmIdentifier
	EncryptedData       []byte
}

func (i encryptedPrivateKeyInfo) Algorithm() pkix.AlgorithmIdentifier {
	return i.AlgorithmIdentifier
}

func (i encryptedPrivateKeyInfo) Data() []byte {
	return i.EncryptedData
}

func (i *encryptedPrivateKeyInfo) SetData(data []byte) {
	i.EncryptedData = data
}

// PEM block types
const (
	certificateType = "CERTIFICATE"
	privateKeyType  = "PRIVATE KEY"
)

// unmarshal calls asn1.Unmarshal, but also returns an error if there is any
// trailing data after unmarshaling.
func unmarshal(in []byte, out interface{}) error {
	trailing, err := asn1.Unmarshal(in, out)
	if err != nil {
		return err
	}
	if len(trailing) != 0 {
		return errors.New("pkcs12: trailing data found")
	}
	return nil
}

// ToPEM converts all "safe bags" contained in pfxData to PEM blocks.
//
// Deprecated: ToPEM creates invalid PEM blocks (private keys
// are encoded as raw RSA or EC private keys rather than PKCS#8 despite being
// labeled "PRIVATE KEY").  To decode a PKCS#12 file, use [DecodeChain] instead,
// and use the [encoding/pem] package to convert to PEM if necessary.
func ToPEM(pfxData []byte, password string) ([]*pem.Block, error) {
	encodedPassword, err := bmpStringZeroTerminated(password)
	if err != nil {
		return nil, ErrIncorrectPassword
	}

	bags, encodedPassword, err := getSafeContents(pfxData, encodedPassword, 2, 2)

	if err != nil {
		return nil, err
	}

	blocks := make([]*pem.Block, 0, len(bags))
	for _, bag := range bags {
		block, err := convertBag(&bag, encodedPassword)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

func convertBag(bag *safeBag, password []byte) (*pem.Block, error) {
	block := &pem.Block{
		Headers: make(map[string]string),
	}

	for _, attribute := range bag.Attributes {
		k, v, err := convertAttribute(&attribute)
		if err != nil {
			return nil, err
		}
		block.Headers[k] = v
	}

	switch {
	case bag.Id.Equal(oidCertBag):
		block.Type = certificateType
		certsData, err := decodeCertBag(bag.Value.Bytes)
		if err != nil {
			return nil, err
		}
		block.Bytes = certsData
	case bag.Id.Equal(oidPKCS8ShroundedKeyBag):
		block.Type = privateKeyType

		key, err := decodePkcs8ShroudedKeyBag(bag.Value.Bytes, password)
		if err != nil {
			return nil, err
		}

		switch key := key.(type) {
		case *rsa.PrivateKey:
			block.Bytes = x509.MarshalPKCS1PrivateKey(key)
		case *ecdsa.PrivateKey:
			block.Bytes, err = x509.MarshalECPrivateKey(key)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("found unknown private key type in PKCS#8 wrapping")
		}
	default:
		return nil, errors.New("don't know how to convert a safe bag of type " + bag.Id.String())
	}
	return block, nil
}

func convertAttribute(attribute *pkcs12Attribute) (key, value string, err error) {
	isString := false

	switch {
	case attribute.Id.Equal(oidFriendlyName):
		key = "friendlyName"
		isString = true
	case attribute.Id.Equal(oidLocalKeyID):
		key = "localKeyId"
	case attribute.Id.Equal(oidMicrosoftCSPName):
		// This key is chosen to match OpenSSL.
		key = "Microsoft CSP Name"
		isString = true
	default:
		return "", "", errors.New("pkcs12: unknown attribute with OID " + attribute.Id.String())
	}

	if isString {
		if err := unmarshal(attribute.Value.Bytes, &attribute.Value); err != nil {
			return "", "", err
		}
		if value, err = decodeBMPString(attribute.Value.Bytes); err != nil {
			return "", "", err
		}
	} else {
		var id []byte
		if err := unmarshal(attribute.Value.Bytes, &id); err != nil {
			return "", "", err
		}
		value = hex.EncodeToString(id)
	}

	return key, value, nil
}

// Decode extracts a certificate and private key from pfxData, which must be a DER-encoded PKCS#12 file. This function
// assumes that there is only one certificate and only one private key in the
// pfxData.  Since PKCS#12 files often contain more than one certificate, you
// probably want to use [DecodeChain] instead.
func Decode(pfxData []byte, password string) (privateKey interface{}, certificate *x509.Certificate, err error) {
	var caCerts []*x509.Certificate
	privateKey, certificate, caCerts, err = DecodeChain(pfxData, password)
	if len(caCerts) != 0 {
		err = errors.New("pkcs12: expected exactly two safe bags in the PFX PDU")
	}
	return
}

// DecodeChain extracts a certificate, a CA certificate chain, and private key
// from pfxData, which must be a DER-encoded PKCS#12 file. This function assumes that there is at least one certificate
// and only one private key in the pfxData.  The first certificate is assumed to
// be the leaf certificate, and subsequent certificates, if any, are assumed to
// comprise the CA certificate chain.
func DecodeChain(pfxData []byte, password string) (privateKey interface{}, certificate *x509.Certificate, caCerts []*x509.Certificate, err error) {
	encodedPassword, err := bmpStringZeroTerminated(password)
	if err != nil {
		return nil, nil, nil, err
	}

	bags, encodedPassword, err := getSafeContents(pfxData, encodedPassword, 1, 2)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, bag := range bags {
		switch {
		case bag.Id.Equal(oidCertBag):
			certsData, err := decodeCertBag(bag.Value.Bytes)
			if err != nil {
				return nil, nil, nil, err
			}
			certs, err := x509.ParseCertificates(certsData)
			if err != nil {
				return nil, nil, nil, err
			}
			if len(certs) != 1 {
				err = errors.New("pkcs12: expected exactly one certificate in the certBag")
				return nil, nil, nil, err
			}
			if certificate == nil {
				certificate = certs[0]
			} else {
				caCerts = append(caCerts, certs[0])
			}

		case bag.Id.Equal(oidKeyBag):
			if privateKey != nil {
				err = errors.New("pkcs12: expected exactly one key bag")
				return nil, nil, nil, err
			}

			if privateKey, err = x509.ParsePKCS8PrivateKey(bag.Value.Bytes); err != nil {
				return nil, nil, nil, err
			}
		case bag.Id.Equal(oidPKCS8ShroundedKeyBag):
			if privateKey != nil {
				err = errors.New("pkcs12: expected exactly one key bag")
				return nil, nil, nil, err
			}

			if privateKey, err = decodePkcs8ShroudedKeyBag(bag.Value.Bytes, encodedPassword); err != nil {
				return nil, nil, nil, err
			}
		}
	}

	if certificate == nil {
		return nil, nil, nil, errors.New("pkcs12: certificate missing")
	}
	if privateKey == nil {
		return nil, nil, nil, errors.New("pkcs12: private key missing")
	}

	return
}

// DecodeTrustStore extracts the certificates from pfxData, which must be a DER-encoded
// PKCS#12 file containing exclusively certificates with attribute 2.16.840.1.113894.746875.1.1,
// which is used by Java to designate a trust anchor.
//
// If the password argument is empty, DecodeTrustStore will decode either password-less
// PKCS#12 files (i.e. those without encryption) or files with a literal empty password.
func DecodeTrustStore(pfxData []byte, password string) (certs []*x509.Certificate, err error) {
	encodedPassword, err := bmpStringZeroTerminated(password)
	if err != nil {
		return nil, err
	}

	bags, encodedPassword, err := getSafeContents(pfxData, encodedPassword, 1, 1)
	if err != nil {
		return nil, err
	}

	for _, bag := range bags {
		switch {
		case bag.Id.Equal(oidCertBag):
			if !bag.hasAttribute(oidJavaTrustStore) {
				return nil, errors.New("pkcs12: trust store contains a certificate that is not marked as trusted")
			}
			certsData, err := decodeCertBag(bag.Value.Bytes)
			if err != nil {
				return nil, err
			}
			parsedCerts, err := x509.ParseCertificates(certsData)
			if err != nil {
				return nil, err
			}

			if len(parsedCerts) != 1 {
				err = errors.New("pkcs12: expected exactly one certificate in the certBag")
				return nil, err
			}

			certs = append(certs, parsedCerts[0])

		default:
			return nil, errors.New("pkcs12: expected only certificate bags")
		}
	}

	return
}

func getSafeContents(p12Data, password []byte, expectedItemsMin int, expectedItemsMax int) (bags []safeBag, updatedPassword []byte, err error) {
	pfx := new(pfxPdu)
	if err := unmarshal(p12Data, pfx); err != nil {
		return nil, nil, errors.New("pkcs12: error reading P12 data: " + err.Error())
	}

	if pfx.Version != 3 {
		return nil, nil, NotImplementedError("can only decode v3 PFX PDU's")
	}

	if !pfx.AuthSafe.ContentType.Equal(oidDataContentType) {
		return nil, nil, NotImplementedError("only password-protected PFX is implemented")
	}

	// unmarshal the explicit bytes in the content for type 'data'
	if err := unmarshal(pfx.AuthSafe.Content.Bytes, &pfx.AuthSafe.Content); err != nil {
		return nil, nil, err
	}

	if len(pfx.MacData.Mac.Algorithm.Algorithm) == 0 {
		if !(len(password) == 2 && password[0] == 0 && password[1] == 0) {
			return nil, nil, errors.New("pkcs12: no MAC in data")
		}
	} else if err := verifyMac(&pfx.MacData, pfx.AuthSafe.Content.Bytes, password); err != nil {
		if err == ErrIncorrectPassword && len(password) == 2 && password[0] == 0 && password[1] == 0 {
			// some implementations use an empty byte array
			// for the empty string password try one more
			// time with empty-empty password
			password = nil
			err = verifyMac(&pfx.MacData, pfx.AuthSafe.Content.Bytes, password)
		}
		if err != nil {
			return nil, nil, err
		}
	}

	var authenticatedSafe []contentInfo
	if err := unmarshal(pfx.AuthSafe.Content.Bytes, &authenticatedSafe); err != nil {
		return nil, nil, err
	}

	if len(authenticatedSafe) < expectedItemsMin || len(authenticatedSafe) > expectedItemsMax {
		if expectedItemsMin == expectedItemsMax {
			return nil, nil, NotImplementedError(fmt.Sprintf("expected exactly %d items in the authenticated safe, but this file has %d", expectedItemsMin, len(authenticatedSafe)))
		}
		return nil, nil, NotImplementedError(fmt.Sprintf("expected between %d and %d items in the authenticated safe, but this file has %d", expectedItemsMin, expectedItemsMax, len(authenticatedSafe)))
	}

	for _, ci := range authenticatedSafe {
		var data []byte

		switch {
		case ci.ContentType.Equal(oidDataContentType):
			if err := unmarshal(ci.Content.Bytes, &data); err != nil {
				return nil, nil, err
			}
		case ci.ContentType.Equal(oidEncryptedDataContentType):
			var encryptedData encryptedData
			if err := unmarshal(ci.Content.Bytes, &encryptedData); err != nil {
				return nil, nil, err
			}
			if encryptedData.Version != 0 {
				return nil, nil, NotImplementedError("only version 0 of EncryptedData is supported")
			}
			if data, err = pbDecrypt(encryptedData.EncryptedContentInfo, password); err != nil {
				return nil, nil, err
			}
		default:
			return nil, nil, NotImplementedError("only data and encryptedData content types are supported in authenticated safe")
		}

		var safeContents []safeBag
		if err := unmarshal(data, &safeContents); err != nil {
			return nil, nil, err
		}
		bags = append(bags, safeContents...)
	}

	return bags, password, nil
}

// Encode is equivalent to LegacyRC2.WithRand(rand).Encode.
// See [Encoder.Encode] and [LegacyRC2] for details.
//
// Deprecated: for the same behavior, use LegacyRC2.Encode; for
// better compatibility, use Legacy.Encode; for better
// security, use Modern.Encode.
func Encode(rand io.Reader, privateKey interface{}, certificate *x509.Certificate, caCerts []*x509.Certificate, password string) (pfxData []byte, err error) {
	return LegacyRC2.WithRand(rand).Encode(privateKey, certificate, caCerts, password)
}

// Encode produces pfxData containing one private key (privateKey), an
// end-entity certificate (certificate), and any number of CA certificates
// (caCerts).
//
// The pfxData is encrypted and authenticated with keys derived from
// the provided password.
//
// Encode emulates the behavior of OpenSSL's PKCS12_create: it creates two
// SafeContents: one that's encrypted with the certificate encryption algorithm
// and contains the certificates, and another that is unencrypted and contains the
// private key shrouded with the key encryption algorithm.  The private key bag and
// the end-entity certificate bag have the LocalKeyId attribute set to the SHA-1
// fingerprint of the end-entity certificate.
func (enc *Encoder) Encode(privateKey interface{}, certificate *x509.Certificate, caCerts []*x509.Certificate, password string) (pfxData []byte, err error) {
	if enc.macAlgorithm == nil && enc.certAlgorithm == nil && enc.keyAlgorithm == nil && password != "" {
		return nil, errors.New("password must be empty")
	}

	encodedPassword, err := bmpStringZeroTerminated(password)
	if err != nil {
		return nil, err
	}

	var pfx pfxPdu
	pfx.Version = 3

	var certFingerprint = sha1.Sum(certificate.Raw)
	var localKeyIdAttr pkcs12Attribute
	localKeyIdAttr.Id = oidLocalKeyID
	localKeyIdAttr.Value.Class = 0
	localKeyIdAttr.Value.Tag = 17
	localKeyIdAttr.Value.IsCompound = true
	if localKeyIdAttr.Value.Bytes, err = asn1.Marshal(certFingerprint[:]); err != nil {
		return nil, err
	}

	var certBags []safeBag
	if certBag, err := makeCertBag(certificate.Raw, []pkcs12Attribute{localKeyIdAttr}); err != nil {
		return nil, err
	} else {
		certBags = append(certBags, *certBag)
	}

	for _, cert := range caCerts {
		if certBag, err := makeCertBag(cert.Raw, []pkcs12Attribute{}); err != nil {
			return nil, err
		} else {
			certBags = append(certBags, *certBag)
		}
	}

	var keyBag safeBag
	if enc.keyAlgorithm == nil {
		keyBag.Id = oidKeyBag
		keyBag.Value.Class = 2
		keyBag.Value.Tag = 0
		keyBag.Value.IsCompound = true
		if keyBag.Value.Bytes, err = x509.MarshalPKCS8PrivateKey(privateKey); err != nil {
			return nil, err
		}
	} else {
		keyBag.Id = oidPKCS8ShroundedKeyBag
		keyBag.Value.Class = 2
		keyBag.Value.Tag = 0
		keyBag.Value.IsCompound = true
		if keyBag.Value.Bytes, err = encodePkcs8ShroudedKeyBag(enc.rand, privateKey, enc.keyAlgorithm, encodedPassword, enc.encryptionIterations, enc.saltLen); err != nil {
			return nil, err
		}
	}
	keyBag.Attributes = append(keyBag.Attributes, localKeyIdAttr)

	// Construct an authenticated safe with two SafeContents.
	// The first SafeContents is encrypted and contains the cert bags.
	// The second SafeContents is unencrypted and contains the shrouded key bag.
	var authenticatedSafe [2]contentInfo
	if authenticatedSafe[0], err = makeSafeContents(enc.rand, certBags, enc.certAlgorithm, encodedPassword, enc.encryptionIterations, enc.saltLen); err != nil {
		return nil, err
	}
	if authenticatedSafe[1], err = makeSafeContents(enc.rand, []safeBag{keyBag}, nil, nil, 0, 0); err != nil {
		return nil, err
	}

	var authenticatedSafeBytes []byte
	if authenticatedSafeBytes, err = asn1.Marshal(authenticatedSafe[:]); err != nil {
		return nil, err
	}

	if enc.macAlgorithm != nil {
		// compute the MAC
		pfx.MacData.Mac.Algorithm.Algorithm = enc.macAlgorithm
		pfx.MacData.MacSalt = make([]byte, enc.saltLen)
		if _, err = enc.rand.Read(pfx.MacData.MacSalt); err != nil {
			return nil, err
		}
		pfx.MacData.Iterations = enc.macIterations
		if err = computeMac(&pfx.MacData, authenticatedSafeBytes, encodedPassword); err != nil {
			return nil, err
		}
	}

	pfx.AuthSafe.ContentType = oidDataContentType
	pfx.AuthSafe.Content.Class = 2
	pfx.AuthSafe.Content.Tag = 0
	pfx.AuthSafe.Content.IsCompound = true
	if pfx.AuthSafe.Content.Bytes, err = asn1.Marshal(authenticatedSafeBytes); err != nil {
		return nil, err
	}

	if pfxData, err = asn1.Marshal(pfx); err != nil {
		return nil, errors.New("pkcs12: error writing P12 data: " + err.Error())
	}
	return
}

// EncodeTrustStore is equivalent to LegacyRC2.WithRand(rand).EncodeTrustStore.
// See [Encoder.EncodeTrustStore] and [LegacyRC2] for details.
//
// Deprecated: for the same behavior, use LegacyRC2.EncodeTrustStore; to generate passwordless trust stores,
// use Passwordless.EncodeTrustStore.
func EncodeTrustStore(rand io.Reader, certs []*x509.Certificate, password string) (pfxData []byte, err error) {
	return LegacyRC2.WithRand(rand).EncodeTrustStore(certs, password)
}

// EncodeTrustStore produces pfxData containing any number of CA certificates
// (certs) to be trusted. The certificates will be marked with a special OID that
// allow it to be used as a Java TrustStore in Java 1.8 and newer.
//
// EncodeTrustStore creates a single SafeContents that's optionally encrypted
// and contains the certificates.
//
// The Subject of the certificates are used as the Friendly Names (Aliases)
// within the resulting pfxData. If certificates share a Subject, then the
// resulting Friendly Names (Aliases) will be identical, which Java may treat as
// the same entry when used as a Java TrustStore, e.g. with `keytool`.  To
// customize the Friendly Names, use [EncodeTrustStoreEntries].
func (enc *Encoder) EncodeTrustStore(certs []*x509.Certificate, password string) (pfxData []byte, err error) {
	var certsWithFriendlyNames []TrustStoreEntry
	for _, cert := range certs {
		certsWithFriendlyNames = append(certsWithFriendlyNames, TrustStoreEntry{
			Cert:         cert,
			FriendlyName: cert.Subject.String(),
		})
	}
	return enc.EncodeTrustStoreEntries(certsWithFriendlyNames, password)
}

// TrustStoreEntry represents an entry in a Java TrustStore.
type TrustStoreEntry struct {
	Cert         *x509.Certificate
	FriendlyName string
}

// EncodeTrustStoreEntries is equivalent to LegacyRC2.WithRand(rand).EncodeTrustStoreEntries.
// See [Encoder.EncodeTrustStoreEntries] and [LegacyRC2] for details.
//
// Deprecated: for the same behavior, use LegacyRC2.EncodeTrustStoreEntries; to generate passwordless trust stores,
// use Passwordless.EncodeTrustStoreEntries.
func EncodeTrustStoreEntries(rand io.Reader, entries []TrustStoreEntry, password string) (pfxData []byte, err error) {
	return LegacyRC2.WithRand(rand).EncodeTrustStoreEntries(entries, password)
}

// EncodeTrustStoreEntries produces pfxData containing any number of CA
// certificates (entries) to be trusted. The certificates will be marked with a
// special OID that allow it to be used as a Java TrustStore in Java 1.8 and newer.
//
// This is identical to [Encoder.EncodeTrustStore], but also allows for setting specific
// Friendly Names (Aliases) to be used per certificate, by specifying a slice
// of TrustStoreEntry.
//
// If the same Friendly Name is used for more than one certificate, then the
// resulting Friendly Names (Aliases) in the pfxData will be identical, which Java
// may treat as the same entry when used as a Java TrustStore, e.g. with `keytool`.
//
// EncodeTrustStoreEntries creates a single SafeContents that's optionally
// encrypted and contains the certificates.
func (enc *Encoder) EncodeTrustStoreEntries(entries []TrustStoreEntry, password string) (pfxData []byte, err error) {
	if enc.macAlgorithm == nil && enc.certAlgorithm == nil && password != "" {
		return nil, errors.New("password must be empty")
	}

	encodedPassword, err := bmpStringZeroTerminated(password)
	if err != nil {
		return nil, err
	}

	var pfx pfxPdu
	pfx.Version = 3

	var certAttributes []pkcs12Attribute

	extKeyUsageOidBytes, err := asn1.Marshal(oidAnyExtendedKeyUsage)
	if err != nil {
		return nil, err
	}

	// the oidJavaTrustStore attribute contains the EKUs for which
	// this trust anchor will be valid
	certAttributes = append(certAttributes, pkcs12Attribute{
		Id: oidJavaTrustStore,
		Value: asn1.RawValue{
			Class:      0,
			Tag:        17,
			IsCompound: true,
			Bytes:      extKeyUsageOidBytes,
		},
	})

	var certBags []safeBag
	for _, entry := range entries {

		bmpFriendlyName, err := bmpString(entry.FriendlyName)
		if err != nil {
			return nil, err
		}

		encodedFriendlyName, err := asn1.Marshal(asn1.RawValue{
			Class:      0,
			Tag:        30,
			IsCompound: false,
			Bytes:      bmpFriendlyName,
		})
		if err != nil {
			return nil, err
		}

		friendlyName := pkcs12Attribute{
			Id: oidFriendlyName,
			Value: asn1.RawValue{
				Class:      0,
				Tag:        17,
				IsCompound: true,
				Bytes:      encodedFriendlyName,
			},
		}

		certBag, err := makeCertBag(entry.Cert.Raw, append(certAttributes, friendlyName))
		if err != nil {
			return nil, err
		}
		certBags = append(certBags, *certBag)
	}

	// Construct an authenticated safe with one SafeContent.
	// The SafeContents is contains the cert bags.
	var authenticatedSafe [1]contentInfo
	if authenticatedSafe[0], err = makeSafeContents(enc.rand, certBags, enc.certAlgorithm, encodedPassword, enc.encryptionIterations, enc.saltLen); err != nil {
		return nil, err
	}

	var authenticatedSafeBytes []byte
	if authenticatedSafeBytes, err = asn1.Marshal(authenticatedSafe[:]); err != nil {
		return nil, err
	}

	if enc.macAlgorithm != nil {
		// compute the MAC
		pfx.MacData.Mac.Algorithm.Algorithm = enc.macAlgorithm
		pfx.MacData.MacSalt = make([]byte, enc.saltLen)
		if _, err = enc.rand.Read(pfx.MacData.MacSalt); err != nil {
			return nil, err
		}
		pfx.MacData.Iterations = enc.macIterations
		if err = computeMac(&pfx.MacData, authenticatedSafeBytes, encodedPassword); err != nil {
			return nil, err
		}
	}

	pfx.AuthSafe.ContentType = oidDataContentType
	pfx.AuthSafe.Content.Class = 2
	pfx.AuthSafe.Content.Tag = 0
	pfx.AuthSafe.Content.IsCompound = true
	if pfx.AuthSafe.Content.Bytes, err = asn1.Marshal(authenticatedSafeBytes); err != nil {
		return nil, err
	}

	if pfxData, err = asn1.Marshal(pfx); err != nil {
		return nil, errors.New("pkcs12: error writing P12 data: " + err.Error())
	}
	return
}

func makeCertBag(certBytes []byte, attributes []pkcs12Attribute) (certBag *safeBag, err error) {
	certBag = new(safeBag)
	certBag.Id = oidCertBag
	certBag.Value.Class = 2
	certBag.Value.Tag = 0
	certBag.Value.IsCompound = true
	if certBag.Value.Bytes, err = encodeCertBag(certBytes); err != nil {
		return nil, err
	}
	certBag.Attributes = attributes
	return
}

func makeSafeContents(rand io.Reader, bags []safeBag, algoID asn1.ObjectIdentifier, password []byte, iterations int, saltLen int) (ci contentInfo, err error) {
	var data []byte
	if data, err = asn1.Marshal(bags); err != nil {
		return
	}

	if algoID == nil {
		ci.ContentType = oidDataContentType
		ci.Content.Class = 2
		ci.Content.Tag = 0
		ci.Content.IsCompound = true
		if ci.Content.Bytes, err = asn1.Marshal(data); err != nil {
			return
		}
	} else {
		randomSalt := make([]byte, saltLen)
		if _, err = rand.Read(randomSalt); err != nil {
			return
		}

		var algo pkix.AlgorithmIdentifier
		algo.Algorithm = algoID
		if algoID.Equal(oidPBES2) {
			if algo.Parameters.FullBytes, err = makePBES2Parameters(rand, randomSalt, iterations); err != nil {
				return
			}
		} else {
			if algo.Parameters.FullBytes, err = asn1.Marshal(pbeParams{Salt: randomSalt, Iterations: iterations}); err != nil {
				return
			}
		}

		var encryptedData encryptedData
		encryptedData.Version = 0
		encryptedData.EncryptedContentInfo.ContentType = oidDataContentType
		encryptedData.EncryptedContentInfo.ContentEncryptionAlgorithm = algo
		if err = pbEncrypt(&encryptedData.EncryptedContentInfo, data, password); err != nil {
			return
		}

		ci.ContentType = oidEncryptedDataContentType
		ci.Content.Class = 2
		ci.Content.Tag = 0
		ci.Content.IsCompound = true
		if ci.Content.Bytes, err = asn1.Marshal(encryptedData); err != nil {
			return
		}
	}
	return
}
