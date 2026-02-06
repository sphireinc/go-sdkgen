package openapi

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/getkin/kin-openapi/openapi2"
)

type Model struct {
	Title      string
	BasePath   string
	Operations []Operation
}

type Operation struct {
	Name        string
	Method      string
	Path        string
	PathParams  []Param
	QueryParams []Param
	HasBody     bool
	Summary     string
	Description string
}

type Param struct {
	Name     string
	Required bool
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
