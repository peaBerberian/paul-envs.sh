package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// FileHash calculates the SHA256 hash of a given file and returns it as a hex string
func FileHash(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to read file '%s': %v", filePath, err)
	}

	hash := sha256.New()
	hash.Write(data)
	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString, nil
}

// BufferHash calculates the SHA256 hash of a buffer and returns it as a hex string
func BufferHash(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)
	return hashString
}
