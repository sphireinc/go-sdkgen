package templates

import "embed"

//go:embed ts/*.tmpl js/*.tmpl
var FS embed.FS
