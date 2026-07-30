package templating

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"

	"github.com/mermonia/peridot/internal/files"
	"github.com/mermonia/peridot/internal/utils"
)

func RenderFile(path string, variables map[string]string, out io.Writer) error {
	if isTextFile, err := files.IsTextFile(path); err != nil {
		return fmt.Errorf("could not check if file is text file: %w", err)
	} else if !isTextFile {
		return utils.CopyToWriter(path, out)
	}

	t, err := template.ParseFiles(path)
	if err != nil {
		return fmt.Errorf("could not parse file for templating: %w", err)
	}

	return t.ExecuteTemplate(out, filepath.Base(path), variables)
}

func RenderedFileContent(path string, variables map[string]string) ([]byte, error) {
	var out bytes.Buffer

	if err := RenderFile(path, variables, &out); err != nil {
		return nil, fmt.Errorf("could not render template: %w", err)
	}

	return out.Bytes(), nil
}

func IsRenderedFileUpToDate(renderedFilePath string, content []byte) bool {
	current, err := os.ReadFile(renderedFilePath)
	return err == nil && bytes.Equal(current, content)
}

func WriteRenderedFile(renderedFilePath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(renderedFilePath), 0755); err != nil {
		return fmt.Errorf("could not create parent dirs: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(renderedFilePath), "."+filepath.Base(renderedFilePath)+".tmp")
	if err != nil {
		return fmt.Errorf("could not create temporary rendered file: %w", err)
	}
	tmpPath := tmp.Name()

	defer os.Remove(tmpPath)

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("could not write temporary rendered file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not close temporary rendered file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("could not set rendered file permissions: %w", err)
	}

	if err := os.Rename(tmpPath, renderedFilePath); err != nil {
		return fmt.Errorf("could not replace rendered file: %w", err)
	}

	return nil
}
