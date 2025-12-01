package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// generateUUIDv4 returns a random RFC4122 UUID v4 using only stdlib.
func GenerateUUIDv4() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// Set version (4) and variant (RFC4122)
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4 (random)
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10xxxxxx

	buf := make([]byte, 36)
	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])

	return string(buf), nil
}
