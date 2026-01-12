//go:build !embedui

package backend

import (
	"io/fs"
	"os"
)

func uiFileSystem() (fs.FS, bool) {
	if _, err := os.Stat("web/dist/index.html"); err != nil {
		return nil, false
	}
	return os.DirFS("web/dist"), true
}
