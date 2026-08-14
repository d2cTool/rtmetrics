package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignMatchesHMAC(t *testing.T) {
	t.Parallel()

	data := []byte(`{"id":"Alloc","type":"gauge","value":1.5}`)
	const key = "secret"

	mac := hmac.New(sha256.New, []byte(key))
	_, err := mac.Write(data)
	require.NoError(t, err)

	assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), Sign(data, key))
}

func TestValid(t *testing.T) {
	t.Parallel()

	data := []byte("payload")
	const key = "secret"
	signature := Sign(data, key)

	assert.True(t, Valid(data, key, signature))
	assert.False(t, Valid([]byte("other payload"), key, signature), "подпись не должна подходить к другим данным")
	assert.False(t, Valid(data, "another key", signature), "подпись не должна подходить к другому ключу")
	assert.False(t, Valid(data, key, "не hex"), "битая строка не должна проходить проверку")
	assert.False(t, Valid(data, key, ""), "пустая подпись не должна проходить проверку")
}
