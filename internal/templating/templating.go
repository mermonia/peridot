package templating

import (
	"fmt"
	"io"
	"path/filepath"
	"text/template"
)

func ProcessFile(path string, variables map[string]string, out io.Writer) error {
	t, err := template.ParseFiles(path)
	if err != nil {
		return fmt.Errorf("could not parse file for templating: %w", err)
	}

	return t.ExecuteTemplate(out, filepath.Base(path), variables)
}
