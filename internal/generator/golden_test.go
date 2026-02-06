package generator_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sphireinc/go-sdkgen/internal/generator"
)

func TestGolden_Examples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		outSubdir string
		lang      string
		sdkName   string
		authMode  string
		tokenFn   string
	}{
		{
			name:      "telephone-ts",
			input:     filepath.FromSlash("../../examples/swagger_telephone.json"),
			outSubdir: "telephone-ts",
			lang:      "ts",
			sdkName:   "TelephoneSDK",
			authMode:  "bearer",
			tokenFn:   "getToken",
		},
		{
			name:      "telephone-js",
			input:     filepath.FromSlash("../../examples/swagger_telephone.json"),
			outSubdir: "telephone-js",
			lang:      "js",
			sdkName:   "TelephoneSDK",
			authMode:  "bearer",
			tokenFn:   "getToken",
		},
		{
			name:      "dog-parlor-ts",
			input:     filepath.FromSlash("../../examples/swagger_dog_parlor.json"),
			outSubdir: "dog-parlor-ts",
			lang:      "ts",
			sdkName:   "DogParlorSDK",
			authMode:  "bearer",
			tokenFn:   "getToken",
		},
		{
			name:      "dog-parlor-js",
			input:     filepath.FromSlash("../../examples/swagger_dog_parlor.json"),
			outSubdir: "dog-parlor-js",
			lang:      "js",
			sdkName:   "DogParlorSDK",
			authMode:  "bearer",
			tokenFn:   "getToken",
		},
		{
			name:      "customer-booking-ts",
			input:     filepath.FromSlash("../../examples/swagger_customer_booking.json"),
			outSubdir: "customer-booking-ts",
			lang:      "ts",
			sdkName:   "CustomerBookingSDK",
			authMode:  "bearer",
			tokenFn:   "getToken",
		},
		{
			name:      "customer-booking-js",
			input:     filepath.FromSlash("../../examples/swagger_customer_booking.json"),
			outSubdir: "customer-booking-js",
			lang:      "js",
			sdkName:   "CustomerBookingSDK",
			authMode:  "bearer",
			tokenFn:   "getToken",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tmp := t.TempDir()
			outDir := filepath.Join(tmp, tc.outSubdir)

			cfg := generator.Config{
				InputPath:  tc.input,
				OutputDir:  outDir,
				Lang:       tc.lang,
				SDKName:    tc.sdkName,
				BaseURLVar: "baseApiUrl",
				AuthMode:   tc.authMode,
				TokenFn:    tc.tokenFn,
			}

			if err := generator.Generate(cfg); err != nil {
				t.Fatalf("Generate() failed: %v", err)
			}

			goldenDir := filepath.Join("testdata", "golden", tc.outSubdir)
			compareDirs(t, goldenDir, outDir)
		})
	}
}

func compareDirs(t *testing.T, goldenDir, gotDir string) {
	t.Helper()

	goldenFiles := listFiles(t, goldenDir)
	gotFiles := listFiles(t, gotDir)

	if strings.Join(goldenFiles, "\n") != strings.Join(gotFiles, "\n") {
		t.Fatalf("file list mismatch\n--- golden\n%s\n--- got\n%s\n\nIf change is intended: run `make golden`",
			strings.Join(goldenFiles, "\n"), strings.Join(gotFiles, "\n"))
	}

	for _, rel := range goldenFiles {
		gb := readFile(t, filepath.Join(goldenDir, rel))
		ob := readFile(t, filepath.Join(gotDir, rel))

		gb = normalizeNewlines(gb)
		ob = normalizeNewlines(ob)

		if !bytes.Equal(gb, ob) {
			t.Fatalf("content mismatch: %s\n\nIf change is intended: run `make golden`", rel)
		}
	}
}

func listFiles(t *testing.T, dir string) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk dir %s: %v", dir, err)
	}
	sort.Strings(files)
	return files
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

func normalizeNewlines(b []byte) []byte {
	// Normalize CRLF to LF for cross-platform stability.
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
