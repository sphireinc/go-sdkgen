package generator_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/sphireinc/go-sdkgen/internal/generator"
)

func TestGolden_Examples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		lang     string
		sdkName  string
		authMode string
		tokenFn  string

		// invariants per example
		expectRouteKeys []string
	}{
		{
			name:     "telephone-ts",
			input:    relToRepo(t, "examples/swagger_telephone.json"),
			lang:     "ts",
			sdkName:  "TelephoneSDK",
			authMode: "bearer",
			tokenFn:  "getToken",
			expectRouteKeys: []string{
				"createPhoneNumber",
				"deletePhoneNumber",
				"getPhoneNumber",
				"listPhoneNumbers",
			},
		},
		{
			name:     "telephone-js",
			input:    relToRepo(t, "examples/swagger_telephone.json"),
			lang:     "js",
			sdkName:  "TelephoneSDK",
			authMode: "bearer",
			tokenFn:  "getToken",
			expectRouteKeys: []string{
				"createPhoneNumber",
				"deletePhoneNumber",
				"getPhoneNumber",
				"listPhoneNumbers",
			},
		},
		{
			name:     "dog-parlor-ts",
			input:    relToRepo(t, "examples/swagger_dog_parlor.json"),
			lang:     "ts",
			sdkName:  "DogParlorSDK",
			authMode: "bearer",
			tokenFn:  "getToken",
			// derived names (no operationId) – keep these stable via your naming logic
			expectRouteKeys: []string{
				"getDogs",
				"postDogs",
				"getDogsById",
				"getDogsByIdAppointments",
			},
		},
		{
			name:     "dog-parlor-js",
			input:    relToRepo(t, "examples/swagger_dog_parlor.json"),
			lang:     "js",
			sdkName:  "DogParlorSDK",
			authMode: "bearer",
			tokenFn:  "getToken",
			expectRouteKeys: []string{
				"getDogs",
				"postDogs",
				"getDogsById",
				"getDogsByIdAppointments",
			},
		},
		{
			name:     "customer-booking-ts",
			input:    relToRepo(t, "examples/swagger_customer_booking.json"),
			lang:     "ts",
			sdkName:  "CustomerBookingSDK",
			authMode: "bearer",
			tokenFn:  "getToken",
			expectRouteKeys: []string{
				"listCustomers",
				"getCustomer",
				"listBookings",
				"createBooking",
			},
		},
		{
			name:     "customer-booking-js",
			input:    relToRepo(t, "examples/swagger_customer_booking.json"),
			lang:     "js",
			sdkName:  "CustomerBookingSDK",
			authMode: "bearer",
			tokenFn:  "getToken",
			expectRouteKeys: []string{
				"listCustomers",
				"getCustomer",
				"listBookings",
				"createBooking",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := generator.Config{
				InputPath:  tc.input,
				Lang:       tc.lang,
				SDKName:    tc.sdkName,
				BaseURLVar: "baseApiUrl",
				AuthMode:   tc.authMode,
				TokenFn:    tc.tokenFn,
			}

			// Generate twice; must be byte-identical (determinism)
			dir1 := filepath.Join(t.TempDir(), "gen1")
			dir2 := filepath.Join(t.TempDir(), "gen2")

			cfg.OutputDir = dir1
			if err := generator.Generate(cfg); err != nil {
				t.Fatalf("Generate(gen1) failed: %v", err)
			}

			cfg.OutputDir = dir2
			if err := generator.Generate(cfg); err != nil {
				t.Fatalf("Generate(gen2) failed: %v", err)
			}

			// Expected file set
			expectFiles := expectedFilesForLang(tc.lang)
			assertContainsFiles(t, dir1, expectFiles)
			assertContainsFiles(t, dir2, expectFiles)

			// Deterministic directory contents
			compareDirsExact(t, dir1, dir2)

			// Invariants:
			// 1) No literal placeholder leakage like "$baseApiUrl"
			assertNoSubstringInDir(t, dir1, "$baseApiUrl")
			// 2) routes contains stable keys
			assertRoutesContainKeys(t, dir1, tc.lang, tc.expectRouteKeys)
			// 3) SDK functions expose stable signature markers (path/query/body/config)
			assertSDKHasStableSignature(t, dir1, tc.lang)
		})
	}
}

func relToRepo(t *testing.T, rel string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// this file is internal/generator/golden_test.go
	// repo root is 2 levels up from internal/generator
	pkgDir := filepath.Dir(thisFile)
	repoRoot := filepath.Clean(filepath.Join(pkgDir, "..", ".."))
	return filepath.Join(repoRoot, filepath.FromSlash(rel))
}

func expectedFilesForLang(lang string) []string {
	switch strings.ToLower(lang) {
	case "js":
		return []string{"index.js", "requests.js", "routes.js", "sdk.js"}
	case "ts":
		return []string{"index.ts", "requests.ts", "routes.ts", "sdk.ts"}
	default:
		return nil
	}
}

func assertContainsFiles(t *testing.T, dir string, want []string) {
	t.Helper()
	for _, f := range want {
		p := filepath.Join(dir, f)
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file missing: %s (%v)", p, err)
		}
	}
}

func compareDirsExact(t *testing.T, a, b string) {
	t.Helper()

	af := listFiles(t, a)
	bf := listFiles(t, b)

	if strings.Join(af, "\n") != strings.Join(bf, "\n") {
		t.Fatalf("file list mismatch\n--- a\n%s\n--- b\n%s", strings.Join(af, "\n"), strings.Join(bf, "\n"))
	}

	for _, rel := range af {
		ab := normalizeNewlines(readFile(t, filepath.Join(a, rel)))
		bb := normalizeNewlines(readFile(t, filepath.Join(b, rel)))
		if !bytes.Equal(ab, bb) {
			t.Fatalf("content mismatch: %s", rel)
		}
	}
}

func assertNoSubstringInDir(t *testing.T, dir, bad string) {
	t.Helper()
	files := listFiles(t, dir)
	for _, rel := range files {
		b := readFile(t, filepath.Join(dir, rel))
		if bytes.Contains(b, []byte(bad)) {
			t.Fatalf("found forbidden substring %q in %s", bad, rel)
		}
	}
}

func assertRoutesContainKeys(t *testing.T, outDir, lang string, keys []string) {
	t.Helper()

	routesFile := "routes." + strings.ToLower(lang)
	b := string(readFile(t, filepath.Join(outDir, routesFile)))

	// Very lightweight check: ensure each key appears as an object member.
	// JS:  foo: { ... }
	// TS:  foo: {
	for _, k := range keys {
		needle := k + ":"
		if !strings.Contains(b, needle) {
			t.Fatalf("routes missing expected key %q in %s", k, routesFile)
		}
	}

	// Also ensure keys are emitted in sorted order (stable output).
	// We find all top-level keys using a regex and compare to sorted copy.
	// This assumes your template prints routes as `const routes = { ... }` or `const routes: ... = { ... }`.
	re := regexp.MustCompile(`(?m)^\s*([A-Za-z_][A-Za-z0-9_]*)\s*:\s*\{`)
	m := re.FindAllStringSubmatch(b, -1)

	var got []string
	for _, mm := range m {
		got = append(got, mm[1])
	}
	if len(got) == 0 {
		t.Fatalf("failed to parse route keys from %s", routesFile)
	}

	sorted := append([]string(nil), got...)
	sort.Strings(sorted)
	if strings.Join(got, ",") != strings.Join(sorted, ",") {
		t.Fatalf("routes keys are not sorted (output not stable)\n--- got\n%v\n--- sorted\n%v", got, sorted)
	}
}

func assertSDKHasStableSignature(t *testing.T, outDir, lang string) {
	t.Helper()

	sdkFile := "sdk." + strings.ToLower(lang)
	b := string(readFile(t, filepath.Join(outDir, sdkFile)))

	// We expect our stable signature to appear at least once.
	// JS template: (path = undefined, query = undefined, body = undefined, config = {})
	// TS template: (path?: ..., query?: ..., body?: ..., config: ... = {})
	if strings.ToLower(lang) == "js" {
		if !strings.Contains(b, "path = undefined") ||
			!strings.Contains(b, "query = undefined") ||
			!strings.Contains(b, "body = undefined") ||
			!strings.Contains(b, "config = {}") {
			t.Fatalf("sdk does not appear to use stable signature in %s", sdkFile)
		}
	} else {
		// TS: just verify the parameter names appear in the exported function signature region.
		if !strings.Contains(b, "path?") ||
			!strings.Contains(b, "query?") ||
			!strings.Contains(b, "body?") ||
			!strings.Contains(b, "config:") {
			t.Fatalf("sdk does not appear to use stable signature in %s", sdkFile)
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
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
