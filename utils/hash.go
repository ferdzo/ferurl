package utils

import (
	"crypto/sha256"
)

const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func GenerateUrlHash(url string) string {
	hash := sha256.Sum256([]byte(url))
	return EncodeToBase62(string(hash[:]))
}

func EncodeToBase62(url string) string {
	hash := sha256.Sum256([]byte(url))

	encoded := ""
	for _, b := range hash[:] {
		encoded += string(base62[int(b)%62])
	}
	if len(encoded) > 7 {
		return encoded[:7]
	}
	return encoded
}
