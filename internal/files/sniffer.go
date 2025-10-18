package files

import (
	"fmt"
	"io"
	"os"
)

func IsTextFile(path string) (bool, error) {
	const sniffSize = 4000

	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	buf := make([]byte, sniffSize)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("could not read file: %w", err)
	}

	for i := range n {
		if buf[i] == 0 {
			return false, nil
		}
	}

	return true, nil
}
