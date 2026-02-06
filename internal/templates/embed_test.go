package templates

import (
	"io/fs"
	"testing"
)

func TestEmbeddedTemplatesPresent(t *testing.T) {
	var files []string
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded FS: %v", err)
	}

	t.Logf("embedded files:\n- %s", join(files, "\n- "))
}

func join(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}
