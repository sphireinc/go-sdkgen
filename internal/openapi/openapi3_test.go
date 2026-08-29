package openapi_test

import (
	"bytes"
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

func constFixture(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	return filepath.Join(wd, "..", "..", "examples", "openapi31_const.yaml")
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
		if op.Name == "submitClaim" && op.Auth {
			t.Fatal("operation-level security override was not honored")
		}
	}
	if !found {
		t.Fatal("listMyClaims was not generated")
	}
}

func TestOpenAPI31ConstKeywordsBecomeLiteralTypes(t *testing.T) {
	spec, err := openapi.LoadOpenAPI3(constFixture(t), false)
	if err != nil {
		t.Fatal(err)
	}
	m, err := openapi.BuildModelV3(spec)
	if err != nil {
		t.Fatal(err)
	}
	find := func(name string) string {
		for _, typ := range m.Types {
			if typ.Name == name {
				return typ.Type
			}
		}
		return ""
	}
	if got := find("ApiInfo"); !strings.Contains(got, "\"Example API\"") || !strings.Contains(got, "\"v1\"") {
		t.Fatalf("ApiInfo constants missing: %s", got)
	}
	holder := find("ConstHolder")
	for _, literal := range []string{"publishedStatus: \"published\"", "reportable: true", "revision: 1", "nullableValue: null", "kind: \"nested\"", "Array<\"item\">", "Record<string, false>"} {
		if !strings.Contains(holder, literal) {
			t.Fatalf("ConstHolder missing literal %q: %s", literal, holder)
		}
	}
	if got := find("ComposedInfo"); !strings.Contains(got, "ApiInfo") || !strings.Contains(got, `kind: "composed"`) {
		t.Fatalf("composed constant schema missing: %s", got)
	}
}

func TestOpenAPI31ConstGenerationIsDeterministic(t *testing.T) {
	d1, d2 := t.TempDir(), t.TempDir()
	c := generator.Config{InputPath: constFixture(t), OutputDir: d1, Lang: "ts", SDKName: "ConstantsSDK", BaseURLVar: "baseApiUrl", AuthMode: "none", EmitSchemas: true, EmitOperations: true}
	if err := generator.Generate(c); err != nil {
		t.Fatal(err)
	}
	c.OutputDir = d2
	if err := generator.Generate(c); err != nil {
		t.Fatal(err)
	}
	a, err := os.ReadFile(filepath.Join(d1, "types.ts"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(d2, "types.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("const generation is not deterministic")
	}
	if !bytes.Contains(a, []byte("export type GetConstantsResponse = ConstHolder;")) {
		t.Fatal("typed operation response missing")
	}
	for _, literal := range []string{"name: \"Example API\"", "version: \"v1\"", "publishedStatus: \"published\"", "nullableValue: null"} {
		if !bytes.Contains(a, []byte(literal)) {
			t.Fatalf("generated output missing literal %q", literal)
		}
	}
}

func TestOpenAPI31RejectsInvalidConstType(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad-const.yaml")
	doc := []byte("openapi: 3.1.0\ninfo: {title: Bad, version: 1}\npaths: {}\ncomponents:\n  schemas:\n    Bad:\n      type: string\n      const: 42\n")
	if err := os.WriteFile(bad, doc, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openapi.LoadOpenAPI3(bad, false); err == nil {
		t.Fatal("expected invalid const type to be rejected")
	}
}
