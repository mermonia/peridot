package appcontext

import "github.com/mermonia/peridot/internal/paths"

type Context struct {
	DotfilesDir string
}

func New() *Context {
	return &Context{
		DotfilesDir: paths.DotfilesDir(),
	}
}
