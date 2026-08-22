package rsaenc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	plain := []byte(`[{"id":"Alloc","type":"gauge","value":1.5}]`)
	enc, err := Encrypt(&priv.PublicKey, plain)
	require.NoError(t, err)
	require.NotEqual(t, plain, enc)

	got, err := Decrypt(priv, enc)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestEncryptNilKeyReturnsPlaintext(t *testing.T) {
	t.Parallel()

	plain := []byte("hello")
	got, err := Encrypt(nil, plain)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestDecryptRejectsGarbage(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	_, err = Decrypt(priv, []byte("not-ciphertext"))
	require.Error(t, err)
}

func TestLoadPEMKeys(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	dir := t.TempDir()
	privPath := filepath.Join(dir, "key.pem")
	pubPath := filepath.Join(dir, "key.pub")

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	require.NoError(t, os.WriteFile(privPath, privPEM, 0o600))
	require.NoError(t, os.WriteFile(pubPath, pubPEM, 0o644))

	loadedPub, err := LoadPublicKey(pubPath)
	require.NoError(t, err)
	loadedPriv, err := LoadPrivateKey(privPath)
	require.NoError(t, err)

	plain := []byte("payload")
	enc, err := Encrypt(loadedPub, plain)
	require.NoError(t, err)
	got, err := Decrypt(loadedPriv, enc)
	require.NoError(t, err)
	assert.Equal(t, plain, got)
}

func TestLoadPKCS8PrivateKey(t *testing.T) {
	t.Parallel()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "pkcs8.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600))

	loaded, err := LoadPrivateKey(path)
	require.NoError(t, err)
	assert.Equal(t, priv.N, loaded.N)
}
