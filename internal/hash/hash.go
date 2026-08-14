// Package hash реализует подпись тела HTTP-запросов и ответов
// алгоритмом HMAC-SHA256 с общим ключом агента и сервера.
package hash

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Header — имя HTTP-заголовка, в котором передаётся подпись.
const Header = "HashSHA256"

// Sign возвращает HMAC-SHA256 от data в шестнадцатеричном виде.
func Sign(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// Valid сообщает, соответствует ли подпись signature данным data.
// Сравнение выполняется через hmac.Equal, чтобы не давать утечки по времени.
func Valid(data []byte, key, signature string) bool {
	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hmac.Equal(got, mac.Sum(nil))
}
