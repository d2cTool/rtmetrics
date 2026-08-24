// Package rsaenc — гибридное шифрование тел запросов: RSA-OAEP + AES-256-GCM.
package rsaenc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// Header помечает зашифрованное тело, чтобы сервер не пытался расшифровать
// обычный JSON от автотестов и curl.
const (
	Header      = "X-Encrypted"
	HeaderValue = "1"
)

var (
	errNoPEM        = errors.New("pem block not found")
	errShortCipher  = errors.New("ciphertext too short")
	errNotRSAPub    = errors.New("not an rsa public key")
	errNotRSAPriv   = errors.New("not an rsa private key")
	errBadAESKeyLen = errors.New("unexpected aes key length")
)

// LoadPublicKey читает PEM с публичным ключом (PKIX или PKCS#1).
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key %s: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("public key %s: %w", path, errNoPEM)
	}

	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errNotRSAPub
		}
		return rsaPub, nil
	}
	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key %s: %w", path, err)
	}
	return pub, nil
}

// LoadPrivateKey читает PEM с приватным ключом (PKCS#1 или PKCS#8).
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("private key %s: %w", path, errNoPEM)
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key %s: %w", path, err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errNotRSAPriv
	}
	return key, nil
}

// Encrypt шифрует plaintext: случайный AES-256 ключ закрывается RSA-OAEP,
// тело — AES-GCM. Формат: rsa(aesKey) || nonce || seal(plaintext).
func Encrypt(pub *rsa.PublicKey, plaintext []byte) ([]byte, error) {
	if pub == nil {
		return plaintext, nil
	}

	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("generate aes key: %w", err)
	}

	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("rsa-oaep: %w", err)
	}

	nonce, sealed, err := sealAESGCM(aesKey, plaintext)
	if err != nil {
		return nil, err
	}

	out := make([]byte, 0, len(encKey)+len(nonce)+len(sealed))
	out = append(out, encKey...)
	out = append(out, nonce...)
	out = append(out, sealed...)
	return out, nil
}

// Decrypt восстанавливает plaintext, зашифрованный Encrypt.
func Decrypt(priv *rsa.PrivateKey, ciphertext []byte) ([]byte, error) {
	if priv == nil {
		return ciphertext, nil
	}

	rsaLen := priv.Size()
	if len(ciphertext) < rsaLen {
		return nil, errShortCipher
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext[:rsaLen], nil)
	if err != nil {
		return nil, fmt.Errorf("rsa-oaep: %w", err)
	}
	if len(aesKey) != 32 {
		return nil, errBadAESKeyLen
	}

	return openAESGCM(aesKey, ciphertext[rsaLen:])
}

func sealAESGCM(aesKey, plaintext []byte) (nonce, sealed []byte, err error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("gcm: %w", err)
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("nonce: %w", err)
	}
	return nonce, gcm.Seal(nil, nonce, plaintext, nil), nil
}

func openAESGCM(aesKey, rest []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	ns := gcm.NonceSize()
	if len(rest) < ns {
		return nil, errShortCipher
	}
	plain, err := gcm.Open(nil, rest[:ns], rest[ns:], nil)
	if err != nil {
		return nil, fmt.Errorf("gcm open: %w", err)
	}
	return plain, nil
}
