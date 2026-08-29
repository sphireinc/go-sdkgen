package openapi

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
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
	if err := spec.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("invalid openapi document: %w", err)
	}
	// The Go representation intentionally uses nil for both an absent const
	// and a JSON null const. Preserve the distinction in Extensions for the
	// emitter without weakening the parser's validation.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid openapi document: %w", err)
	}
	markNullConsts(spec, raw)
	return spec, nil
}

func markNullConsts(doc *openapi3.T, raw map[string]any) {
	var visit func(any, *openapi3.Schema)
	visit = func(value any, parsed *openapi3.Schema) {
		m, ok := value.(map[string]any)
		if !ok || parsed == nil {
			return
		}
		if v, exists := m["const"]; exists && v == nil {
			if parsed.Extensions == nil {
				parsed.Extensions = map[string]any{}
			}
			parsed.Extensions["x-sdkgen-null-const"] = true
		}
		if props, ok := m["properties"].(map[string]any); ok {
			for name, child := range props {
				if ref := parsed.Properties[name]; ref != nil {
					visit(child, ref.Value)
				}
			}
		}
		if child, ok := m["items"]; ok && parsed.Items != nil {
			visit(child, parsed.Items.Value)
		}
		for _, keyword := range []string{"oneOf", "anyOf", "allOf", "prefixItems"} {
			if list, ok := m[keyword].([]any); ok {
				var refs openapi3.SchemaRefs
				switch keyword {
				case "oneOf":
					refs = parsed.OneOf
				case "anyOf":
					refs = parsed.AnyOf
				case "allOf":
					refs = parsed.AllOf
				default:
					refs = parsed.PrefixItems
				}
				for i, child := range list {
					if i < len(refs) && refs[i] != nil {
						visit(child, refs[i].Value)
					}
				}
			}
		}
		if child, ok := m["additionalProperties"]; ok && parsed.AdditionalProperties.Schema != nil {
			visit(child, parsed.AdditionalProperties.Schema.Value)
		}
		if defs, ok := m["$defs"].(map[string]any); ok {
			for name, child := range defs {
				if ref := parsed.Defs[name]; ref != nil {
					visit(child, ref.Value)
				}
			}
		}
	}
	if components, ok := raw["components"].(map[string]any); ok {
		if schemas, ok := components["schemas"].(map[string]any); ok && doc.Components != nil {
			for name, value := range schemas {
				if ref := doc.Components.Schemas[name]; ref != nil {
					visit(value, ref.Value)
				}
			}
		}
	}
	if paths, ok := raw["paths"].(map[string]any); ok && doc.Paths != nil {
		for path, rawItem := range paths {
			item := doc.Paths.Value(path)
			rawItemMap, ok := rawItem.(map[string]any)
			if !ok || item == nil {
				continue
			}
			for method, operation := range item.Operations() {
				rawOperation, ok := rawItemMap[method].(map[string]any)
				if !ok || operation == nil {
					continue
				}
				if rawParams, ok := rawOperation["parameters"].([]any); ok {
					for i, rawParam := range rawParams {
						if i < len(operation.Parameters) && operation.Parameters[i] != nil && operation.Parameters[i].Value != nil && operation.Parameters[i].Value.Schema != nil {
							if p, ok := rawParam.(map[string]any); ok {
								if schema, ok := p["schema"]; ok {
									visit(schema, operation.Parameters[i].Value.Schema.Value)
								}
							}
						}
					}
				}
				if rawBody, ok := rawOperation["requestBody"].(map[string]any); ok && operation.RequestBody != nil && operation.RequestBody.Value != nil {
					if content, ok := rawBody["content"].(map[string]any); ok {
						for media, rawMedia := range content {
							if parsed := operation.RequestBody.Value.Content[media]; parsed != nil && parsed.Schema != nil {
								if mediaMap, ok := rawMedia.(map[string]any); ok {
									if schema, ok := mediaMap["schema"]; ok {
										visit(schema, parsed.Schema.Value)
									}
								}
							}
						}
					}
				}
				if rawResponses, ok := rawOperation["responses"].(map[string]any); ok && operation.Responses != nil {
					for code, rawResponse := range rawResponses {
						parsedResponse := operation.Responses.Value(code)
						if parsedResponse == nil || parsedResponse.Value == nil {
							continue
						}
						if responseMap, ok := rawResponse.(map[string]any); ok {
							if content, ok := responseMap["content"].(map[string]any); ok {
								for media, rawMedia := range content {
									if parsed := parsedResponse.Value.Content[media]; parsed != nil && parsed.Schema != nil {
										if mediaMap, ok := rawMedia.(map[string]any); ok {
											if schema, ok := mediaMap["schema"]; ok {
												visit(schema, parsed.Schema.Value)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
