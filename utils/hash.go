package utils

import (
	"crypto/sha256"
	"fmt"
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func GenerateUrlHash(url string) string {
	encoded := EncodeToBase62(url)
	hash := sha256.Sum256([]byte(encoded))

	return fmt.Sprintf("%x", hash)
}

func EncodeToBase62(url string) string {
	hash := sha256.Sum256([]byte(url))

	encoded := ""
	for _, b := range hash[:] {
		encoded += string(base62[int(b)%62])
	}
	if len(encoded) > 6 {
		encoded = encoded[:6]
	}

	return encoded
}
