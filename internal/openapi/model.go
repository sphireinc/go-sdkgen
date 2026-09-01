package openapi

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
)

type Model struct {
	Title      string
	BasePath   string
	Operations []Operation
	Types      []TypeDef
	Typed      bool
}

type Operation struct {
	Name          string
	Method        string
	Path          string
	PathParams    []Param
	QueryParams   []Param
	HasBody       bool
	Multipart     bool
	Summary       string
	Description   string
	ParamsType    string
	BodyType      string
	BodyTypeNamed bool
	SuccessType   string
	ErrorType     string
	Auth          bool
	HeaderParams  []Param
	CookieParams  []Param
}

type Param struct {
	Name     string
	Required bool
	Type     string
}

type TypeDef struct{ Name, Type string }

// BuildModelV3 builds a deterministic, schema-derived model from OpenAPI 3.x.
func BuildModelV3(spec *openapi3.T) (Model, error) {
	if spec == nil || spec.Paths == nil {
		return Model{}, fmt.Errorf("OpenAPI document has no paths")
	}
	title := "API"
	if spec.Info != nil && strings.TrimSpace(spec.Info.Title) != "" {
		title = strings.TrimSpace(spec.Info.Title)
	}
	m := Model{Title: title, Typed: true}
	if len(spec.Servers) > 0 && spec.Servers[0] != nil {
		if u := strings.TrimRight(spec.Servers[0].URL, "/"); u != "" {
			m.BasePath = u
		}
	}
	typeNames := map[*openapi3.Schema]string{}
	if spec.Components != nil {
		names := make([]string, 0, len(spec.Components.Schemas))
		for n := range spec.Components.Schemas {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			if ref := spec.Components.Schemas[n]; ref != nil && ref.Value != nil {
				typeNames[ref.Value] = schemaTypeName(n)
			}
		}
		for _, n := range names {
			ref := spec.Components.Schemas[n]
			if ref != nil && ref.Value != nil {
				// Do not resolve the schema being defined to itself; recursive
				// properties and references to other components still use names.
				name := typeNames[ref.Value]
				delete(typeNames, ref.Value)
				value := schemaTypeScript(ref.Value, typeNames)
				typeNames[ref.Value] = name
				m.Types = append(m.Types, TypeDef{Name: name, Type: value})
			}
		}
	}
	seen := map[string]string{}
	for _, path := range sortedKeys(spec.Paths.Map()) {
		item := spec.Paths.Value(path)
		if item == nil {
			continue
		}
		methods := item.Operations()
		methodNames := make([]string, 0, len(methods))
		for method := range methods {
			methodNames = append(methodNames, method)
		}
		sort.Strings(methodNames)
		for _, method := range methodNames {
			op := methods[method]
			if op == nil {
				continue
			}
			name := toCamelCase(sanitizeIdent(op.OperationID))
			if name == "" {
				name = toCamelCase(deriveBaseName(strings.ToUpper(method), path))
			}
			if _, ok := reservedJS[name]; ok {
				name += "Op"
			}
			key := strings.ToUpper(method) + " " + path
			if old, ok := seen[name]; ok && old != key {
				name += "__" + shortHash(key)
			}
			seen[name] = key
			o := Operation{Name: name, Method: strings.ToUpper(method), Path: path, Summary: strings.TrimSpace(op.Summary), Description: strings.TrimSpace(op.Description), Auth: len(spec.Security) > 0}
			params := append([]*openapi3.ParameterRef{}, item.Parameters...)
			params = append(params, op.Parameters...)
			for _, pr := range params {
				if pr == nil || pr.Value == nil {
					continue
				}
				p := pr.Value
				typ := schemaTypeScript(refSchema(p.Schema), typeNames)
				pp := Param{Name: p.Name, Required: p.Required, Type: typ}
				switch p.In {
				case "path":
					o.PathParams = append(o.PathParams, pp)
				case "query":
					o.QueryParams = append(o.QueryParams, pp)
				case "header":
					o.HeaderParams = append(o.HeaderParams, pp)
				case "cookie":
					o.CookieParams = append(o.CookieParams, pp)
				}
			}
			if op.Security != nil {
				o.Auth = len(*op.Security) > 0
			}
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				o.HasBody = true
				for mediaType := range op.RequestBody.Value.Content {
					if strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
						o.Multipart = true
						break
					}
				}
				o.BodyType = contentType(op.RequestBody.Value.Content, typeNames)
				for _, media := range op.RequestBody.Value.Content {
					if media != nil && media.Schema != nil {
						_, o.BodyTypeNamed = typeNames[media.Schema.Value]
						if o.BodyTypeNamed {
							break
						}
					}
				}
				if o.BodyType == "" {
					o.BodyType = "unknown"
				}
			}
			o.ParamsType = toPascalCase(name) + "Params"
			o.SuccessType = toPascalCase(name) + "Response"
			o.ErrorType = toPascalCase(name) + "Error"
			m.Types = append(m.Types, TypeDef{Name: o.ParamsType, Type: paramsTypeScript(o)})
			m.Types = append(m.Types, TypeDef{Name: o.ErrorType, Type: "{ message: string; code?: string; details?: unknown }"})
			responseTypeName := toPascalCase(name) + "Response"
			responseTypeValue := responseType(op, typeNames)
			if responseTypeValue == "" {
				responseTypeValue = "void"
			}
			o.SuccessType = responseTypeName
			m.Types = append(m.Types, TypeDef{Name: responseTypeName, Type: responseTypeValue})
			sort.Slice(o.PathParams, func(i, j int) bool { return o.PathParams[i].Name < o.PathParams[j].Name })
			sort.Slice(o.QueryParams, func(i, j int) bool { return o.QueryParams[i].Name < o.QueryParams[j].Name })
			sort.Slice(o.HeaderParams, func(i, j int) bool { return o.HeaderParams[i].Name < o.HeaderParams[j].Name })
			sort.Slice(o.CookieParams, func(i, j int) bool { return o.CookieParams[i].Name < o.CookieParams[j].Name })
			m.Operations = append(m.Operations, o)
		}
	}
	sort.Slice(m.Operations, func(i, j int) bool { return m.Operations[i].Name < m.Operations[j].Name })
	sort.Slice(m.Types, func(i, j int) bool { return m.Types[i].Name < m.Types[j].Name })
	return m, nil
}

func schemaTypeName(name string) string {
	typ := toPascalCase(name)
	switch typ {
	case "ApiError", "ApiResult", "RequestConfig", "HttpMethod":
		return typ + "Model"
	default:
		return typ
	}
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func refSchema(r *openapi3.SchemaRef) *openapi3.Schema {
	if r == nil {
		return nil
	}
	return r.Value
}
func contentType(c openapi3.Content, names map[*openapi3.Schema]string) string {
	keys := sortedKeys(c)
	for _, k := range keys {
		if c[k] != nil && c[k].Schema != nil {
			return schemaTypeScript(c[k].Schema.Value, names)
		}
	}
	return ""
}
func responseType(op *openapi3.Operation, names map[*openapi3.Schema]string) string {
	if op.Responses == nil {
		return ""
	}
	keys := sortedKeys(op.Responses.Map())
	for _, k := range keys {
		if strings.HasPrefix(k, "2") {
			r := op.Responses.Value(k)
			if r != nil && r.Value != nil {
				if t := contentType(r.Value.Content, names); t != "" {
					return t
				}
			}
		}
	}
	return ""
}
func paramsTypeScript(o Operation) string {
	fields := []string{}
	add := func(ps []Param) {
		for _, p := range ps {
			n := toCamelCase(sanitizeIdent(p.Name))
			if n == "" {
				n = "value"
			}
			optional := ""
			if !p.Required {
				optional = "?"
			}
			fields = append(fields, fmt.Sprintf("%s%s: %s", n, optional, p.Type))
		}
	}
	add(o.PathParams)
	add(o.QueryParams)
	add(o.HeaderParams)
	add(o.CookieParams)
	if len(fields) == 0 {
		return "Record<string, never>"
	}
	return "{ " + strings.Join(fields, "; ") + " }"
}
func schemaTypeScript(s *openapi3.Schema, names map[*openapi3.Schema]string) string {
	if s == nil {
		return "unknown"
	}
	if s.Extensions != nil {
		if _, ok := s.Extensions["x-sdkgen-null-const"]; ok {
			return "null"
		}
	}
	if s.Const != nil {
		return constTypeScript(s.Const)
	}
	if strings.EqualFold(s.Format, "binary") {
		return "Blob"
	}
	if n := names[s]; n != "" {
		return n
	}
	if len(s.OneOf) > 0 || len(s.AnyOf) > 0 {
		refs := s.OneOf
		if len(refs) == 0 {
			refs = s.AnyOf
		}
		ts := []string{}
		for _, r := range refs {
			ts = append(ts, schemaTypeScript(refSchema(r), names))
		}
		return strings.Join(ts, " | ")
	}
	if len(s.AllOf) > 0 {
		ts := []string{}
		for _, r := range s.AllOf {
			ts = append(ts, schemaTypeScript(refSchema(r), names))
		}
		return strings.Join(ts, " & ")
	}
	if s.Type != nil {
		hasNull := s.Nullable || s.Type.IncludesNull()
		for _, t := range *s.Type {
			if t == "null" {
				hasNull = true
				continue
			}
			switch t {
			case "string":
				if len(s.Enum) > 0 {
					return nullableType(strings.Join(enumValues(s.Enum), " | "), hasNull)
				}
				return nullableType("string", hasNull)
			case "integer", "number":
				return nullableType("number", hasNull)
			case "boolean":
				return nullableType("boolean", hasNull)
			case "array":
				if s.Items != nil {
					return nullableType("Array<"+schemaTypeScript(s.Items.Value, names)+">", hasNull)
				}
				return nullableType("unknown[]", hasNull)
			case "object":
				fields := []string{}
				for _, k := range sortedKeys(s.Properties) {
					p := s.Properties[k]
					if p == nil {
						continue
					}
					opt := "?"
					for _, req := range s.Required {
						if req == k {
							opt = ""
							break
						}
					}
					fields = append(fields, toCamelCase(sanitizeIdent(k))+opt+": "+schemaTypeScript(p.Value, names))
				}
				if len(fields) == 0 {
					if s.AdditionalProperties.Schema != nil {
						return nullableType("Record<string, "+schemaTypeScript(s.AdditionalProperties.Schema.Value, names)+">", hasNull)
					}
					return nullableType("Record<string, unknown>", hasNull)
				}
				return nullableType("{ "+strings.Join(fields, "; ")+" }", hasNull)
			}
		}
		if hasNull {
			return "null"
		}
	}
	return "unknown"
}

func nullableType(base string, nullable bool) string {
	if nullable {
		return base + " | null"
	}
	return base
}

func constTypeScript(value any) string {
	b, err := json.Marshal(value)
	if err != nil {
		return "unknown"
	}
	return string(b)
}
func enumValues(v []any) []string {
	out := []string{}
	for _, x := range v {
		out = append(out, fmt.Sprintf("%q", fmt.Sprint(x)))
	}
	return out
}

var nonIdent = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// JS/TS keywords - treat as reserved/no-op for exporting func names
var reservedJS = map[string]struct{}{
	"break": {}, "case": {}, "catch": {}, "class": {}, "const": {}, "continue": {},
	"debugger": {}, "default": {}, "delete": {}, "do": {}, "else": {}, "export": {},
	"extends": {}, "finally": {}, "for": {}, "function": {}, "if": {}, "import": {},
	"in": {}, "instanceof": {}, "new": {}, "return": {}, "super": {}, "switch": {},
	"this": {}, "throw": {}, "try": {}, "typeof": {}, "var": {}, "void": {},
	"while": {}, "with": {}, "yield": {}, "let": {}, "static": {}, "enum": {},
	"await": {}, "implements": {}, "interface": {}, "package": {}, "private": {},
	"protected": {}, "public": {},
}

// BuildModel builds a deterministic model from swagger.json
func BuildModel(spec *openapi2.T) (Model, error) {
	m := Model{
		Title:    strings.TrimSpace(spec.Info.Title),
		BasePath: strings.TrimSpace(spec.BasePath),
	}
	if m.Title == "" {
		m.Title = "API"
	}

	seen := map[string]string{}

	for path, item := range spec.Paths {
		if item == nil {
			continue
		}
		add := func(method string, op *openapi2.Operation) {
			if op == nil {
				return
			}

			methodUp := strings.ToUpper(method)

			// Candidate raw name
			var rawName string
			if strings.TrimSpace(op.OperationID) != "" {
				rawName = op.OperationID
			} else {
				rawName = deriveBaseName(methodUp, path)
			}

			// Normalize to camelCase and sanitize
			base := toCamelCase(sanitizeIdent(rawName))
			if base == "" {
				base = "op"
			}

			// check against and avoid reserved words
			if _, ok := reservedJS[base]; ok {
				base = base + "Op"
			}

			// Ensure uniqueness deterministically
			key := methodUp + " " + path
			name := base
			if existing, ok := seen[name]; ok && existing != key {
				// stable suffix based on METHOD+path, not on ordering
				name = base + "__" + shortHash(key)
			}
			seen[name] = key

			o := Operation{
				Name:        name,
				Method:      methodUp,
				Path:        path,
				Summary:     strings.TrimSpace(op.Summary),
				Description: strings.TrimSpace(op.Description),
			}

			for _, p := range op.Parameters {
				if p == nil {
					continue
				}

				param := Param{Name: p.Name, Required: p.Required}

				switch strings.ToLower(p.In) {
				case "path":
					o.PathParams = append(o.PathParams, param)
				case "query":
					o.QueryParams = append(o.QueryParams, param)
				case "body", "formdata":
					o.HasBody = true
				}
			}

			sort.Slice(o.PathParams, func(i, j int) bool { return o.PathParams[i].Name < o.PathParams[j].Name })
			sort.Slice(o.QueryParams, func(i, j int) bool { return o.QueryParams[i].Name < o.QueryParams[j].Name })

			m.Operations = append(m.Operations, o)
		}

		add("get", item.Get)
		add("post", item.Post)
		add("put", item.Put)
		add("delete", item.Delete)
		add("patch", item.Patch)
		add("head", item.Head)
		add("options", item.Options)
	}

	// Sort final operations by Name so generated code is stable in diffs
	sort.Slice(m.Operations, func(i, j int) bool { return m.Operations[i].Name < m.Operations[j].Name })

	return m, nil
}

// deriveBaseName is stable and ONLY depends on method + path.
// Example: GET /jobs/runs/{id} -> getJobsRunsById
func deriveBaseName(method, path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	tokens := []string{strings.ToLower(method)}
	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			tokens = append(tokens, "by", strings.Trim(p, "{}"))
			continue
		}
		tokens = append(tokens, p)
	}
	return strings.Join(tokens, "_")
}

func sanitizeIdent(s string) string {
	s = strings.TrimSpace(s)
	s = nonIdent.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}

// toCamelCase converts snake/space/kebab/mixed to lower camelCase.
// "get_jobs_runs_by_id" -> "getJobsRunsById"
// "FetchJobDetails" -> "fetchJobDetails"
func toCamelCase(s string) string {
	if s == "" {
		return ""
	}
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}
	out := words[0]
	for _, w := range words[1:] {
		out += toTitle(w)
	}
	out = ensureValidIdent(out)
	return out
}

// toPascalCase converts to PascalCase (useful later if you generate types).
// "get_jobs_runs_by_id" -> "GetJobsRunsById"
func toPascalCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	for _, w := range words {
		b.WriteString(toTitle(strings.ToLower(w)))
	}
	return ensureValidIdent(b.String())
}

func splitWords(s string) []string {
	// normalize separators into spaces
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.TrimSpace(s)

	// also split camelCase/PascalCase boundaries
	var parts []rune
	for i, r := range s {
		if i > 0 {
			prev := rune(s[i-1])
			if unicode.IsLower(prev) && unicode.IsUpper(r) {
				parts = append(parts, ' ')
			}
		}
		parts = append(parts, r)
	}
	s = string(parts)

	fields := strings.Fields(s)
	var out []string
	for _, f := range fields {
		// strip any remaining junk
		f = nonIdent.ReplaceAllString(f, "")
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// toTilte fuck it, all upper
func toTitle(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func ensureValidIdent(s string) string {
	if s == "" {
		return ""
	}
	// must not start with digit !!important!!
	if s[0] >= '0' && s[0] <= '9' {
		s = "op" + s
	}
	return s
}

func shortHash(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%08x", h.Sum32())
}
