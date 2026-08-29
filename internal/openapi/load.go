package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
)

func LoadSwaggerV2(path string) (*openapi2.T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t openapi2.T
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("invalid swagger json: %w", err)
	}
	return &t, nil
}

// LoadOpenAPI3 loads and validates an OpenAPI 3.x document from JSON or YAML.
// External references are allowed only when explicitly requested by the caller.
func LoadOpenAPI3(path string, allowExternalRefs bool) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = allowExternalRefs
	spec, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("invalid openapi document: %w", err)
	}
	if len(spec.OpenAPI) < 3 || spec.OpenAPI[0] != '3' {
		return nil, fmt.Errorf("unsupported OpenAPI version %q: expected 3.x", spec.OpenAPI)
	}
	// kin-openapi v0.133 models most 3.1 constructs but its validator still
	// rejects the valid JSON Schema 2020-12 `null` type. Temporarily remove
	// only that union member while running the normal validator, then restore
	// the parsed schema. This preserves malformed-document validation.
	if strings.HasPrefix(spec.OpenAPI, "3.1") {
		restore := stripNullableTypes(spec)
		err := spec.Validate(loader.Context)
		restore()
		if err != nil {
			return nil, fmt.Errorf("invalid openapi document: %w", err)
		}
	} else if err := spec.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("invalid openapi document: %w", err)
	}
	return spec, nil
}

func stripNullableTypes(doc *openapi3.T) func() {
	type change struct {
		schema *openapi3.Schema
		types  openapi3.Types
	}
	var changes []change
	seen := map[*openapi3.Schema]bool{}
	var visit func(*openapi3.Schema)
	visit = func(s *openapi3.Schema) {
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		if s.Type != nil {
			kept := make(openapi3.Types, 0, len(*s.Type))
			for _, typ := range *s.Type {
				if typ != "null" {
					kept = append(kept, typ)
				}
			}
			if len(kept) != len(*s.Type) {
				changes = append(changes, change{s, *s.Type})
				s.Type = &kept
			}
		}
		for _, r := range s.OneOf {
			if r != nil {
				visit(r.Value)
			}
		}
		for _, r := range s.AnyOf {
			if r != nil {
				visit(r.Value)
			}
		}
		for _, r := range s.AllOf {
			if r != nil {
				visit(r.Value)
			}
		}
		if s.Items != nil {
			visit(s.Items.Value)
		}
		for _, r := range s.Properties {
			if r != nil {
				visit(r.Value)
			}
		}
		if s.AdditionalProperties.Schema != nil {
			visit(s.AdditionalProperties.Schema.Value)
		}
	}
	visitContent := func(c openapi3.Content) {
		for _, m := range c {
			if m != nil && m.Schema != nil {
				visit(m.Schema.Value)
			}
		}
	}
	if doc.Components != nil {
		for _, r := range doc.Components.Schemas {
			if r != nil {
				visit(r.Value)
			}
		}
	}
	if doc.Paths != nil {
		for _, item := range doc.Paths.Map() {
			if item == nil {
				continue
			}
			for _, op := range item.Operations() {
				if op == nil {
					continue
				}
				for _, p := range op.Parameters {
					if p != nil && p.Value != nil && p.Value.Schema != nil {
						visit(p.Value.Schema.Value)
					}
				}
				if op.RequestBody != nil && op.RequestBody.Value != nil {
					visitContent(op.RequestBody.Value.Content)
				}
				if op.Responses != nil {
					for _, r := range op.Responses.Map() {
						if r != nil && r.Value != nil {
							visitContent(r.Value.Content)
						}
					}
				}
			}
		}
	}
	return func() {
		for _, c := range changes {
			c.schema.Type = &c.types
		}
	}
}
