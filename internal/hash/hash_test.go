package hash

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestHash(t *testing.T) {
	dir := t.TempDir()

	textFiles := []string{
		"Some flavor text, not very long",
		"Some other flavor text, not really long either",
	}

	for i, textFile := range textFiles {
		if err := os.WriteFile(filepath.Join(dir, "textFile"+strconv.Itoa(i)), []byte(textFile), 0644); err != nil {
			t.Fatalf("Could not write text file %d", i)
		}
	}

	existingHashes := make(map[string]bool)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Could not read temp dir")
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		hash, err := HashFile(path)
		if err != nil {
			t.Fatalf("Could not hash file %s", path)
		}

		if existingHashes[hash] {
			t.Fatalf("Detected repeated hash: %s", hash)
		} else {
			existingHashes[hash] = true
		}
	}
}
