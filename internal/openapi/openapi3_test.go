package openapi_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/go-sdkgen/internal/generator"
	"github.com/sphireinc/go-sdkgen/internal/openapi"
)

func fixture(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..", "examples", "openapi31_claims.yaml")
}

func TestOpenAPI31LoadsJSONAndRejectsMalformedDocuments(t *testing.T) {
	wd, _ := os.Getwd()
	jsonPath := filepath.Join(wd, "..", "..", "examples", "openapi31_minimal.json")
	if _, err := openapi.LoadOpenAPI3(jsonPath, false); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(bad, []byte("openapi: 3.1.0\ninfo: {}\npaths: [not-an-object]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openapi.LoadOpenAPI3(bad, false); err == nil {
		t.Fatal("expected malformed document to fail")
	}
}

func TestOpenAPI31LoadsAndBuildsTypedModel(t *testing.T) {
	spec, err := openapi.LoadOpenAPI3(fixture(t), false)
	if err != nil {
		t.Fatal(err)
	}
	m, err := openapi.BuildModelV3(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !m.Typed || len(m.Types) == 0 {
		t.Fatal("expected typed model and schemas")
	}
	var found bool
	for _, op := range m.Operations {
		if op.Name == "listMyClaims" {
			found = true
			if op.SuccessType == "" || op.ParamsType == "" || !op.Auth {
				t.Fatalf("incomplete listMyClaims model: %+v", op)
			}
		}
	}
	if !found {
		t.Fatal("listMyClaims was not generated")
	}
	for _, op := range m.Operations {
		if op.Name == "submitClaim" && op.Auth {
			t.Fatal("operation-level security override was not honored")
		}
	}
}

func TestOpenAPI31GeneratesTypedDeterministicOutput(t *testing.T) {
	d1, d2 := t.TempDir(), t.TempDir()
	c := generator.Config{InputPath: fixture(t), Lang: "ts", SDKName: "ClaimsSDK", OutputDir: d1, BaseURLVar: "baseApiUrl", AuthMode: "bearer", EmitSchemas: true, EmitOperations: true}
	if err := generator.Generate(c); err != nil {
		t.Fatal(err)
	}
	c.OutputDir = d2
	if err := generator.Generate(c); err != nil {
		t.Fatal(err)
	}
	a, _ := os.ReadFile(filepath.Join(d1, "types.ts"))
	b, _ := os.ReadFile(filepath.Join(d2, "types.ts"))
	if string(a) != string(b) {
		t.Fatal("types output is not deterministic")
	}
	sdk, _ := os.ReadFile(filepath.Join(d1, "sdk.ts"))
	out := string(sdk)
	for _, needle := range []string{"listMyClaims", "ListMyClaimsResponse", "ListMyClaimsParams", "ClaimInput"} {
		if !strings.Contains(out, needle) {
			t.Fatalf("sdk missing %q", needle)
		}
	}
	types := string(a)
	for _, needle := range []string{"export type Claim =", "provenance", "pending", "Array<Claim>"} {
		if !strings.Contains(types, needle) {
			t.Fatalf("types missing %q", needle)
		}
	}
}
