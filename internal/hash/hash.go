package hash

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open the file %s: %w", path, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("could not copy the file %s: %w", path, err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
