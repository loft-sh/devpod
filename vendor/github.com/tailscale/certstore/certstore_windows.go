package certstore

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"syscall"
	"unsafe"

	"golang.org/x/sys/cpu"
	"golang.org/x/sys/windows"
)

const myStoreName = "MY"

var (
	// winAPIFlag specifies the flags that should be passed to
	// CryptAcquireCertificatePrivateKey. This impacts whether the CryptoAPI or CNG
	// API will be used.
	//
	// Possible values are:
	//
	//	0                                            — Only use CryptoAPI.
	//	windows.CRYPT_ACQUIRE_ALLOW_NCRYPT_KEY_FLAG  — Prefer CryptoAPI.
	//	windows.CRYPT_ACQUIRE_PREFER_NCRYPT_KEY_FLAG — Prefer CNG.
	//	windows.CRYPT_ACQUIRE_ONLY_NCRYPT_KEY_FLAG   — Only use CNG.
	winAPIFlag uint32 = windows.CRYPT_ACQUIRE_PREFER_NCRYPT_KEY_FLAG
)

// winStore is a wrapper around a certStoreHandle.
type winStore struct {
	store certStoreHandle
}

// openStore opens the current user's personal cert store.
func openStore(location StoreLocation, permissions ...StorePermission) (*winStore, error) {
	storeName, err := windows.UTF16PtrFromString(myStoreName)
	if err != nil {
		return nil, err
	}

	var flags uint32
	switch location {
	case User:
		flags |= windows.CERT_SYSTEM_STORE_CURRENT_USER
	case System:
		flags |= windows.CERT_SYSTEM_STORE_LOCAL_MACHINE
	}

	for _, p := range permissions {
		switch p {
		case ReadOnly:
			flags |= windows.CERT_STORE_READONLY_FLAG
		}
	}

	store, err := windows.CertOpenStore(windows.CERT_STORE_PROV_SYSTEM_W, 0, 0, flags, uintptr(unsafe.Pointer(storeName)))
	if err != nil {
		return nil, fmt.Errorf("failed to open system cert store: %w", err)
	}

	return &winStore{certStoreHandle(store)}, nil
}

// Identities implements the Store interface.
func (s *winStore) Identities() ([]Identity, error) {
	var (
		err    error
		idents []Identity

		// CertFindChainInStore parameters
		encoding = uint32(windows.X509_ASN_ENCODING)
		flags    = uint32(windows.CERT_CHAIN_FIND_BY_ISSUER_CACHE_ONLY_FLAG | windows.CERT_CHAIN_FIND_BY_ISSUER_CACHE_ONLY_URL_FLAG)
		findType = uint32(windows.CERT_CHAIN_FIND_BY_ISSUER)
		params   = &windows.CertChainFindByIssuerPara{Size: uint32(unsafe.Sizeof(windows.CertChainFindByIssuerPara{}))}
		chainCtx *windows.CertChainContext
	)

	for {
		chainCtx, err = windows.CertFindChainInStore(windows.Handle(s.store), encoding, flags, findType, unsafe.Pointer(params), chainCtx)
		if errors.Is(err, syscall.Errno(windows.CRYPT_E_NOT_FOUND)) {
			return idents, nil
		}
		if err != nil {
			err = fmt.Errorf("failed to iterate certs in store: %w", err)
			break
		}
		if chainCtx.ChainCount < 1 {
			err = errors.New("bad chain")
			break
		}

		// not sure why this isn't 1 << 29
		const maxPointerArray = 1 << 28

		// rgpChain is actually an array, but we only care about the first one.
		simpleChain := *chainCtx.Chains
		if simpleChain.NumElements < 1 || simpleChain.NumElements > maxPointerArray {
			err = errors.New("bad chain")
			break
		}

		chainElts := unsafe.Slice(simpleChain.Elements, simpleChain.NumElements)

		// Build chain of certificates from each elt's certificate context.
		chain := make([]*windows.CertContext, len(chainElts))
		for j := range chainElts {
			chain[j] = chainElts[j].CertContext
		}

		idents = append(idents, newWinIdentity(chain))
	}

	for _, ident := range idents {
		ident.Close()
	}

	return nil, err
}

// Import implements the Store interface.
func (s *winStore) Import(data []byte, password string) error {
	cpw, err := windows.UTF16PtrFromString(password)
	if err != nil {
		return fmt.Errorf("converting password: %w", err)
	}

	pfx := &windows.CryptDataBlob{
		Size: uint32(len(data)),
		Data: unsafe.SliceData(data),
	}

	var flags uint32
	flags = windows.CRYPT_USER_KEYSET

	// import into preferred KSP
	if winAPIFlag&windows.CRYPT_ACQUIRE_PREFER_NCRYPT_KEY_FLAG > 0 {
		flags |= windows.PKCS12_PREFER_CNG_KSP
	} else if winAPIFlag&windows.CRYPT_ACQUIRE_ONLY_NCRYPT_KEY_FLAG > 0 {
		flags |= windows.PKCS12_ALWAYS_CNG_KSP
	}

	store, err := windows.PFXImportCertStore(pfx, cpw, flags)
	if err != nil {
		return fmt.Errorf("failed to import PFX cert store: %w", err)
	}
	defer windows.CertCloseStore(store, windows.CERT_CLOSE_STORE_FORCE_FLAG)

	var (
		ctx      *windows.CertContext
		encoding = uint32(windows.X509_ASN_ENCODING | windows.PKCS_7_ASN_ENCODING)
	)

	for {
		// iterate through certs in temporary store
		ctx, err = windows.CertFindCertificateInStore(store, encoding, 0, windows.CERT_FIND_ANY, nil, ctx)
		if errors.Is(err, syscall.Errno(windows.CRYPT_E_NOT_FOUND)) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate certs in store: %w", err)
		}

		// Copy the cert to the system store.
		if err := windows.CertAddCertificateContextToStore(windows.Handle(s.store), ctx, windows.CERT_STORE_ADD_REPLACE_EXISTING, nil); err != nil {
			return fmt.Errorf("failed to add imported certificate to %s store: %w", myStoreName, err)
		}
	}

	return nil
}

// Close implements the Store interface.
func (s *winStore) Close() {
	windows.CertCloseStore(windows.Handle(s.store), 0)
	s.store = 0
}

// winIdentity implements the Identity interface.
type winIdentity struct {
	chain  []*windows.CertContext
	signer *winPrivateKey
}

func newWinIdentity(chain []*windows.CertContext) *winIdentity {
	for _, ctx := range chain {
		windows.CertDuplicateCertificateContext(ctx)
	}

	return &winIdentity{chain: chain}
}

// Certificate implements the Identity interface.
func (i *winIdentity) Certificate() (*x509.Certificate, error) {
	return exportCertCtx(i.chain[0])
}

// CertificateChain implements the Identity interface.
func (i *winIdentity) CertificateChain() ([]*x509.Certificate, error) {
	var (
		certs = make([]*x509.Certificate, len(i.chain))
		err   error
	)

	for j := range i.chain {
		if certs[j], err = exportCertCtx(i.chain[j]); err != nil {
			return nil, err
		}
	}

	return certs, nil
}

// Signer implements the Identity interface.
func (i *winIdentity) Signer() (crypto.Signer, error) {
	return i.getPrivateKey()
}

// getPrivateKey gets this identity's private *winPrivateKey.
func (i *winIdentity) getPrivateKey() (*winPrivateKey, error) {
	if i.signer != nil {
		return i.signer, nil
	}

	cert, err := i.Certificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get identity certificate: %w", err)
	}

	signer, err := newWinPrivateKey(i.chain[0], cert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load identity private key: %w", err)
	}

	i.signer = signer

	return i.signer, nil
}

// Delete implements the Identity interface.
func (i *winIdentity) Delete() error {
	// duplicate cert context, since CertDeleteCertificateFromStore will free it.
	deleteCtx := windows.CertDuplicateCertificateContext(i.chain[0])

	// try deleting cert
	if err := windows.CertDeleteCertificateFromStore(deleteCtx); err != nil {
		return fmt.Errorf("failed to delete certificate from store: %w", err)
	}

	// try deleting private key
	wpk, err := i.getPrivateKey()
	if err != nil {
		return fmt.Errorf("failed to get identity private key: %w", err)
	}

	if err := wpk.Delete(); err != nil {
		return fmt.Errorf("failed to delete identity private key: %w", err)
	}

	return nil
}

// Close implements the Identity interface.
func (i *winIdentity) Close() {
	if i.signer != nil {
		i.signer.Close()
		i.signer = nil
	}

	for _, ctx := range i.chain {
		_ = windows.CertFreeCertificateContext(ctx)
		i.chain = nil
	}
}

// winPrivateKey is a wrapper around a HCRYPTPROV_OR_NCRYPT_KEY_HANDLE.
type winPrivateKey struct {
	publicKey crypto.PublicKey

	// CryptoAPI fields
	capiProv cryptProviderHandle

	// CNG fields
	cngHandle nCryptKeyHandle
	keySpec   uint32
}

// newWinPrivateKey gets a *winPrivateKey for the given certificate.
func newWinPrivateKey(certCtx *windows.CertContext, publicKey crypto.PublicKey) (*winPrivateKey, error) {
	var (
		provOrKey windows.Handle // HCRYPTPROV_OR_NCRYPT_KEY_HANDLE
		keySpec   uint32
		mustFree  bool
	)

	if publicKey == nil {
		return nil, errors.New("nil public key")
	}

	// Get a handle for the found private key.
	if err := windows.CryptAcquireCertificatePrivateKey(certCtx, winAPIFlag, nil, &provOrKey, &keySpec, &mustFree); err != nil {
		return nil, fmt.Errorf("failed to get private key for certificate: %w", err)
	}

	if !mustFree {
		// This shouldn't happen since we're not asking for cached keys.
		return nil, errors.New("CryptAcquireCertificatePrivateKey set mustFree")
	}

	if keySpec == windows.CERT_NCRYPT_KEY_SPEC {
		return &winPrivateKey{
			publicKey: publicKey,
			cngHandle: nCryptKeyHandle(provOrKey),
		}, nil
	} else {
		return &winPrivateKey{
			publicKey: publicKey,
			capiProv:  cryptProviderHandle(provOrKey),
			keySpec:   keySpec,
		}, nil
	}
}

// Public implements the crypto.Signer interface.
func (wpk *winPrivateKey) Public() crypto.PublicKey {
	return wpk.publicKey
}

// Sign implements the crypto.Signer interface.
func (wpk *winPrivateKey) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if wpk.capiProv != 0 {
		return wpk.capiSignHash(opts, digest)
	} else if wpk.cngHandle != 0 {
		return wpk.cngSignHash(opts, digest)
	} else {
		return nil, errors.New("bad private key")
	}
}

type bCryptPKCS1PaddingInfo struct {
	// Algorithm is the name of the hashing algorithm to use, in UTF16. Use one
	// of the BCRYPT_*_ALGORITHM constants.
	Algorithm *uint16
}

type bCryptPSSPaddingInfo struct {
	// Algorithm is the name of the hashing algorithm to use, in UTF16. Use one
	// of the BCRYPT_*_ALGORITHM constants.
	Algorithm *uint16
	// SaltLength is the size, in bytes, of the random salt to use for the
	// padding.
	SaltLength uint32
}

// cngSignHash signs a digest using the CNG APIs.
func (wpk *winPrivateKey) cngSignHash(opts crypto.SignerOpts, digest []byte) ([]byte, error) {
	hash := opts.HashFunc()
	if len(digest) != hash.Size() {
		return nil, errors.New("cngSignHash: bad digest for hash")
	}

	var (
		// input
		padPtr    unsafe.Pointer
		digestPtr = unsafe.SliceData(digest)
		digestLen = uint32(len(digest))
		flags     uint32

		// output
		sigLen uint32
	)

	// setup pkcs1v1.5 padding for RSA
	if _, isRSA := wpk.publicKey.(*rsa.PublicKey); isRSA {
		var alg string
		switch hash {
		case crypto.SHA256:
			alg = BCRYPT_SHA256_ALGORITHM
		case crypto.SHA384:
			alg = BCRYPT_SHA384_ALGORITHM
		case crypto.SHA512:
			alg = BCRYPT_SHA512_ALGORITHM
		default:
			return nil, fmt.Errorf("cngSignHash: converting from Go algorithm: %w", ErrUnsupportedHash)
		}

		algName, err := windows.UTF16PtrFromString(alg)
		if err != nil {
			return nil, err
		}

		if pssOpts, ok := opts.(*rsa.PSSOptions); ok {
			saltLen := pssOpts.SaltLength
			if saltLen == rsa.PSSSaltLengthEqualsHash {
				saltLen = len(digest)
			} else if saltLen == rsa.PSSSaltLengthAuto {
				saltLen = hash.Size()
			}
			padPtr = unsafe.Pointer(&bCryptPSSPaddingInfo{
				Algorithm:  algName,
				SaltLength: uint32(saltLen),
			})
			flags |= BCRYPT_PAD_PSS
		} else {
			padPtr = unsafe.Pointer(&bCryptPKCS1PaddingInfo{
				Algorithm: algName,
			})
			flags |= BCRYPT_PAD_PKCS1
		}
	}

	// get signature length
	if err := nCryptSignHash(wpk.cngHandle, padPtr, digestPtr, digestLen, nil, 0, &sigLen, flags); err != nil {
		return nil, fmt.Errorf("cngSignHash: failed to get signature length: %w", err)
	}

	// get signature
	sig := make([]byte, sigLen)
	if err := nCryptSignHash(wpk.cngHandle, padPtr, digestPtr, digestLen, unsafe.SliceData(sig), sigLen, &sigLen, flags); err != nil {
		return nil, fmt.Errorf("cngSignHash: failed to sign digest: %w", err)
	}

	// CNG returns a raw ECDSA signature, but we wan't ASN.1 DER encoding.
	if _, isEC := wpk.publicKey.(*ecdsa.PublicKey); isEC {
		if len(sig)%2 != 0 {
			return nil, errors.New("cngSignHash: bad ecdsa signature from CNG")
		}

		type ecdsaSignature struct {
			R, S *big.Int
		}

		r := new(big.Int).SetBytes(sig[:len(sig)/2])
		s := new(big.Int).SetBytes(sig[len(sig)/2:])

		encoded, err := asn1.Marshal(ecdsaSignature{r, s})
		if err != nil {
			return nil, fmt.Errorf("cngSignHash: failed to ASN.1 encode EC signature: %w", err)
		}

		return encoded, nil
	}

	return sig, nil
}

// capiSignHash signs a digest using the CryptoAPI APIs.
func (wpk *winPrivateKey) capiSignHash(opts crypto.SignerOpts, digest []byte) ([]byte, error) {
	if _, ok := opts.(*rsa.PSSOptions); ok {
		return nil, fmt.Errorf("capiSignHash: CAPI does not support PSS padding, %w", ErrUnsupportedHash)
	}

	hash := opts.HashFunc()
	if len(digest) != hash.Size() {
		return nil, errors.New("capiSignHash: bad digest for hash")
	}

	// Figure out which CryptoAPI hash algorithm we're using.
	var hash_alg cryptAlgorithm

	switch hash {
	case crypto.SHA256:
		hash_alg = CALG_SHA_256
	case crypto.SHA384:
		hash_alg = CALG_SHA_384
	case crypto.SHA512:
		hash_alg = CALG_SHA_512
	default:
		return nil, fmt.Errorf("capiSignHash: converting from Go algorithm: %w", ErrUnsupportedHash)
	}

	// Instantiate a CryptoAPI hash object.
	var chash cryptHashHandle

	if err := cryptCreateHash(wpk.capiProv, hash_alg, 0, 0, &chash); err != nil {
		if errors.Is(err, syscall.Errno(windows.NTE_BAD_ALGID)) {
			err = ErrUnsupportedHash
		}
		return nil, fmt.Errorf("capiSignHash: cryptCreateHash: %w", err)
	}
	defer cryptDestroyHash(chash)

	// Make sure the hash size matches.
	var (
		hashSize    uint32
		hashSizeLen = uint32(unsafe.Sizeof(hashSize))
	)

	if err := cryptGetHashParam(chash, HP_HASHSIZE, unsafe.Pointer(&hashSize), &hashSizeLen, 0); err != nil {
		return nil, fmt.Errorf("capiSignHash: failed to get hash size: %w", err)
	}

	if hash.Size() != int(hashSize) {
		return nil, errors.New("capiSignHash: invalid CryptoAPI hash")
	}

	// Put our digest into the hash object.
	if err := cryptSetHashParam(chash, HP_HASHVAL, unsafe.Pointer(unsafe.SliceData(digest)), 0); err != nil {
		return nil, fmt.Errorf("capiSignHash: failed to set hash digest: %w", err)
	}

	// Get signature length.
	var sigLen uint32

	if err := cryptSignHash(chash, wpk.keySpec, nil, 0, nil, &sigLen); err != nil {
		return nil, fmt.Errorf("capiSignHash: failed to get signature length: %w", err)
	}

	// Get signature
	sig := make([]byte, int(sigLen))

	if err := cryptSignHash(chash, wpk.keySpec, nil, 0, unsafe.SliceData(sig), &sigLen); err != nil {
		return nil, fmt.Errorf("capiSignHash: failed to sign digest: %w", err)
	}

	// Signature is little endian, but we want big endian. Reverse it.
	for i := len(sig)/2 - 1; i >= 0; i-- {
		opp := len(sig) - 1 - i
		sig[i], sig[opp] = sig[opp], sig[i]
	}

	return sig, nil
}

func (wpk *winPrivateKey) Delete() error {
	if wpk.cngHandle != 0 {
		// Delete CNG key
		if err := nCryptDeleteKey(wpk.cngHandle, 0); err != nil {
			return err
		}
	} else if wpk.capiProv != 0 {
		// Delete CryptoAPI key
		var (
			containerName string
			providerName  string
			providerType  uint32
			err           error
		)

		containerName, err = wpk.getProviderStringParam(PP_CONTAINER)
		if err != nil {
			return fmt.Errorf("failed to get PP_CONTAINER: %w", err)
		}

		providerName, err = wpk.getProviderStringParam(PP_NAME)
		if err != nil {
			return fmt.Errorf("failed to get PP_NAME: %w", err)
		}

		providerType, err = wpk.getProviderUint32Param(PP_PROVTYPE)
		if err != nil {
			return fmt.Errorf("failed to get PP_PROVTYPE: %w", err)
		}

		// use CRYPT_SILENT too?
		var prov windows.Handle
		if err := windows.CryptAcquireContext(&prov, windows.StringToUTF16Ptr(containerName), windows.StringToUTF16Ptr(providerName),
			providerType, windows.CRYPT_DELETEKEYSET); err != nil {
			return fmt.Errorf("failed to delete key set (container=%q, name=%q, provtype=%d): %w",
				containerName, providerName, providerType, err)
		}
		wpk.capiProv = 0
	} else {
		return errors.New("bad private key")
	}

	return nil
}

// getProviderParam gets a parameter about a provider.
func (wpk *winPrivateKey) getProviderParam(param uint32) ([]byte, error) {
	var dataLen uint32
	if err := cryptGetProvParam(wpk.capiProv, param, nil, &dataLen, 0); err != nil {
		return nil, fmt.Errorf("failed to get provider parameter size: %w", err)
	}

	data := make([]byte, dataLen)
	if err := cryptGetProvParam(wpk.capiProv, param, unsafe.SliceData(data), &dataLen, 0); err != nil {
		return nil, fmt.Errorf("failed to get provider parameter: %w", err)
	}

	return data, nil
}

// getProviderStringParam gets a parameter about a provider. The parameter is a zero-terminated char string.
func (wpk *winPrivateKey) getProviderStringParam(param uint32) (string, error) {
	val, err := wpk.getProviderParam(param)
	if err != nil {
		return "", err
	}
	return windows.ByteSliceToString(val), nil
}

// getProviderUint32Param gets a parameter about a provider. The parameter is a uint32.
func (wpk *winPrivateKey) getProviderUint32Param(param uint32) (uint32, error) {
	val, err := wpk.getProviderParam(param)
	if err != nil {
		return 0, err
	}
	if len(val) != 4 {
		return 0, errors.New("uint32 data length is not 32 bits")
	}
	if cpu.IsBigEndian {
		return binary.BigEndian.Uint32(val), nil
	}
	return binary.LittleEndian.Uint32(val), nil
}

// Close closes this winPrivateKey.
func (wpk *winPrivateKey) Close() {
	if wpk.cngHandle != 0 {
		_ = nCryptFreeObject(windows.Handle(wpk.cngHandle))
		wpk.cngHandle = 0
	}

	if wpk.capiProv != 0 {
		_ = windows.CryptReleaseContext(windows.Handle(wpk.capiProv), 0)
		wpk.capiProv = 0
	}
}

// exportCertCtx exports a *windows.CertContext as an *x509.Certificate.
func exportCertCtx(ctx *windows.CertContext) (*x509.Certificate, error) {
	der := unsafe.Slice(ctx.EncodedCert, ctx.Length)

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("certificate parsing failed: %w", err)
	}

	return cert, nil
}
