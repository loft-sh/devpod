package certstore

import (
	"golang.org/x/sys/windows"
)

//go:generate go run golang.org/x/sys/windows/mkwinsyscall -output zsyscall_windows.go syscall_windows.go

type certStoreHandle windows.Handle

// CryptoAPI APIs from wincrypt.h

type cryptProviderHandle windows.Handle // HCRYPTPROV
type cryptHashHandle windows.Handle     // HCRYPTHASH
type cryptKeyHandle windows.Handle      // HCRYPTKEY
type cryptAlgorithm uint32              // ALG_ID

const (
	// Values for ALG_ID
	CALG_SHA_256 cryptAlgorithm = 0x0000800c
	CALG_SHA_384 cryptAlgorithm = 0x0000800d
	CALG_SHA_512 cryptAlgorithm = 0x0000800e

	// Values of the dwParam for CryptGetHashParam and/or CryptSetHashParam
	HP_ALGID     = 0x1
	HP_HASHVAL   = 0x2
	HP_HASHSIZE  = 0x4
	HP_HMAC_INFO = 0x5

	// Values of the dwParam for CryptGetProvParam and/or CryptSetProvParam
	PP_NAME      = 4
	PP_CONTAINER = 6
	PP_PROVTYPE  = 16
)

// https://learn.microsoft.com/en-ca/windows/win32/api/wincrypt/nf-wincrypt-cryptcreatehash
//sys cryptCreateHash(hProv cryptProviderHandle, Algid cryptAlgorithm, hKey cryptKeyHandle, dwFlags uint32, phHash *cryptHashHandle) (err error) = advapi32.CryptCreateHash
// https://learn.microsoft.com/en-us/windows/win32/api/wincrypt/nf-wincrypt-cryptdestroyhash
//sys cryptDestroyHash(hHash cryptHashHandle) (err error) = advapi32.CryptDestroyHash
// https://learn.microsoft.com/en-ca/windows/win32/api/wincrypt/nf-wincrypt-cryptgethashparam
//sys cryptGetHashParam(hHash cryptHashHandle, dwParam uint32, pbData unsafe.Pointer, pdwDataLen *uint32, dwFlags uint32) (err error) = advapi32.CryptGetHashParam
// https://learn.microsoft.com/en-ca/windows/win32/api/wincrypt/nf-wincrypt-cryptsethashparam
//sys cryptSetHashParam(hHash cryptHashHandle, dwParam uint32, pbData unsafe.Pointer, dwFlags uint32) (err error) = advapi32.CryptSetHashParam
// https://learn.microsoft.com/en-ca/windows/win32/api/wincrypt/nf-wincrypt-cryptsignhashw
//sys cryptSignHash(hHash cryptHashHandle, dwKeySpec uint32, szDescription *uint16, dwFlags uint32, pbSignature *byte, pdwSigLen *uint32) (err error) = advapi32.CryptSignHashW
// https://learn.microsoft.com/en-us/windows/win32/api/wincrypt/nf-wincrypt-cryptgetprovparam
//sys cryptGetProvParam(hProv cryptProviderHandle, dwParam uint32, pbData *byte, pdwDataLen *uint32, dwFlags uint32) (err error) = advapi32.CryptGetProvParam
// https://learn.microsoft.com/en-us/windows/win32/api/wincrypt/nf-wincrypt-cryptsetprovparam
//sys cryptSetProvParam(hProv cryptProviderHandle, dwParam uint32, pbData *byte, dwFlags uint32) (err error) = advapi32.CryptSetProvParam

// CNG APIs from bcrypt.h and ncrypt.h.
const (
	// Flags for NCryptSignHash and others
	BCRYPT_PAD_NONE  = 0x00000001
	BCRYPT_PAD_PKCS1 = 0x00000002
	BCRYPT_PAD_OAEP  = 0x00000004
	BCRYPT_PAD_PSS   = 0x00000008
)

type nCryptKeyHandle windows.Handle

// https://learn.microsoft.com/en-us/windows/win32/api/ncrypt/nf-ncrypt-ncryptsignhash
//sys nCryptSignHash(hKey nCryptKeyHandle, pPaddingInfo unsafe.Pointer, pbHashValue *byte, cbHashValue uint32, pbSignature *byte, cbSignature uint32, pcbResult *uint32, dwFlags uint32) (ret error) = ncrypt.NCryptSignHash
// https://learn.microsoft.com/en-us/windows/win32/api/ncrypt/nf-ncrypt-ncryptdeletekey
//sys nCryptDeleteKey(hKey nCryptKeyHandle, dwFlags uint32) (ret error) = ncrypt.NCryptDeleteKey
// https://learn.microsoft.com/en-us/windows/win32/api/ncrypt/nf-ncrypt-ncryptfreeobject
//sys nCryptFreeObject(hObject windows.Handle) (ret error) = ncrypt.NCryptFreeObject
