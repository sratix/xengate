package util

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// HashString returns SHA256 hash of a string
func HashString(input string) string {
	hash := sha256.New()
	hash.Write([]byte(input))
	return hex.EncodeToString(hash.Sum(nil))
}

// HashFile returns SHA256 hash of a file
func HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// HashBytes returns SHA256 hash of a byte slice
func HashBytes(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

// HashReader returns SHA256 hash of a reader
func HashReader(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// VerifyHash verifies if a string matches a given hash
func VerifyHash(input, hash string) bool {
	return HashString(input) == hash
}

// VerifyFileHash verifies if a file matches a given hash
func VerifyFileHash(filePath, hash string) (bool, error) {
	fileHash, err := HashFile(filePath)
	if err != nil {
		return false, err
	}
	return fileHash == hash, nil
}
