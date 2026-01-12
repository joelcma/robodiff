//go:build embedui

package backend

import (
	"io/fs"
)

func uiFileSystem() (fs.FS, bool) {
	// Intentionally disabled: embedding web/dist from this package location is unreliable.
	// Use the default disk-based serving from web/dist.
	return nil, false
}
