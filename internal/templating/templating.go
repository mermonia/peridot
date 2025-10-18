package templating

import (
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

func CreateRenderedFile(path, renderedFilePath string, variables map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(renderedFilePath), 0755); err != nil {
		return fmt.Errorf("could not create parent dirs: %w", err)
	}

	out, err := os.Create(renderedFilePath)
	if err != nil {
		return fmt.Errorf("could not create rendered file path: %w", err)
	}
	defer out.Close()

	if err := RenderFile(path, variables, out); err != nil {
		return fmt.Errorf("could not render template: %w", err)
	}

	return nil
}
