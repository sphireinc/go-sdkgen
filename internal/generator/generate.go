package generator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sphireinc/go-sdkgen/internal/openapi"
	"github.com/sphireinc/go-sdkgen/internal/templates"
	"github.com/sphireinc/go-sdkgen/internal/util"
)

func Generate(cfg Config) error {
	cfg.Lang = strings.ToLower(strings.TrimSpace(cfg.Lang))
	if cfg.Lang != "ts" && cfg.Lang != "js" {
		return fmt.Errorf("unsupported –lang: %s (expected ts or js)", cfg.Lang)
	}

	var (
		model openapi.Model
		err   error
	)
	if b, rerr := os.ReadFile(cfg.InputPath); rerr != nil {
		return rerr
	} else if strings.Contains(string(b), `"openapi"`) || strings.Contains(string(b), "openapi:") {
		spec, lerr := openapi.LoadOpenAPI3(cfg.InputPath, cfg.AllowExternalRefs)
		if lerr != nil {
			return lerr
		}
		model, err = openapi.BuildModelV3(spec)
	} else {
		spec, lerr := openapi.LoadSwaggerV2(cfg.InputPath)
		if lerr != nil {
			return lerr
		}
		model, err = openapi.BuildModel(spec)
	}
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return err
	}

	files := []struct {
		template string
		outName  string
	}{
		{"package.tmpl", "package.json"},
		{"routes.tmpl", "routes." + cfg.Lang},
		{"requests.tmpl", "requests." + cfg.Lang},
		{"sdk.tmpl", "sdk." + cfg.Lang},
		{"index.tmpl", "index." + cfg.Lang},
	}
	if model.Typed && cfg.Lang == "ts" {
		// OpenAPI 3 typed operations always need the generated schema module.
		// EmitSchemas remains available for config documentation and future
		// non-operation modes, while typed output cannot safely omit its types.
		files = append(files, struct{ template, outName string }{"types.tmpl", "types.ts"})
	}

	data := map[string]any{
		"Cfg":   cfg,
		"Model": model,
	}

	for _, f := range files {
		tplPath := filepath.ToSlash(filepath.Join(cfg.Lang, f.template))
		outPath := filepath.Join(cfg.OutputDir, f.outName)

		rendered, rerr := util.RenderTemplateFS(templates.FS, tplPath, data)
		if rerr != nil {
			return fmt.Errorf("render %s: %w", tplPath, rerr)
		}
		if err := os.WriteFile(outPath, []byte(rendered), 0o644); err != nil {
			return err
		}
	}

	if len(model.Operations) == 0 && cfg.EmitOperations {
		return errors.New("no operations found in OpenAPI document")
	}

	return nil
}
