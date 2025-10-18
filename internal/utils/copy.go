package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func CopyToWriter(src string, out io.Writer) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open source: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(out, file)
	return err
}

func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("could not open source: %w", err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("could not create parent dirs: %w", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("could not create destination: %w", err)
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	buf := make([]byte, 32*1024)
	if _, err := io.CopyBuffer(out, in, buf); err != nil {
		return fmt.Errorf("could not copy the contents of src to dst: %w", err)
	}

	if info, err := os.Stat(src); err == nil {
		if err := os.Chmod(dst, info.Mode()); err != nil {
			return fmt.Errorf("could not copy file mode: %w", err)
		}
	}

	return nil
}
