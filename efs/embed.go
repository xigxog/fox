package efs

import (
	"embed"
)

const (
	HelloWorldPath = "hello-world"
)

// Go will not embed directories containing a go.mod file. To resolve this the
// extension .trim is added. This should be removed when writing the template
// files to disk. Additionally, files starting with . or _ are ignored. Adding
// all: prefix ensures they are included.

//go:embed all:*
var EFS embed.FS
