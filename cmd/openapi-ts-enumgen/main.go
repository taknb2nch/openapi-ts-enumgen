package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/taknb2nch/openapi-ts-enumgen/templates"
	"gopkg.in/yaml.v3"
)

type EnumSchema struct {
	Name        string
	Description string
	Deprecated  bool
	Since       string
	See         string
	Items       []EnumItem
}

type EnumItem struct {
	Value string
	Key   string
	Label string
}

type TemplateData struct {
	SourceBase string
	Schemas    []EnumSchema
}

func main() {
	var (
		inPath   string
		outPath  string
		quoteOpt string
		noSort   bool
	)

	flag.StringVar(&inPath, "input", "", "Path to input OpenAPI YAML")
	flag.StringVar(&outPath, "output", "", "Path to output .ts file")
	flag.StringVar(&quoteOpt, "quote", "double", `Quote style for string literals: "single" or "double"`)
	flag.BoolVar(&noSort, "no-sort", false, "Disable schema name sorting")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  openapi-ts-enumgen -in openapi.yaml -out enums.ts [options]\n")
		fmt.Fprintf(os.Stderr, "  openapi-ts-enumgen openapi.yaml enums.ts [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if inPath == "" || outPath == "" {
		flag.Usage()
		os.Exit(2)
	}

	// quote validation
	switch quoteOpt {
	case "single", "double":
	default:
		fmt.Fprintln(os.Stderr, `-quote must be "single" or "double"`)
		os.Exit(2)
	}

	root, err := loadYAML(inPath)

	must(err)

	schemas := extractEnums(root)

	if !noSort {
		sort.Slice(schemas, func(i, j int) bool {
			return schemas[i].Name < schemas[j].Name
		})
	}

	sourceBase := filepath.Base(inPath)

	for i := range schemas {
		schemas[i].See = fmt.Sprintf(
			"OpenAPI components/schemas/%s (%s)",
			schemas[i].Name,
			sourceBase,
		)
	}

	data := TemplateData{
		SourceBase: filepath.Base(inPath),
		Schemas:    schemas,
	}

	tpl := template.Must(template.New("ts").
		Funcs(template.FuncMap{
			"computedKey": func(schemaName, itemKey string) string {
				return fmt.Sprintf("[%s.%s]", schemaName, itemKey)
			},
			"jsDocLines": func(s string) []string {
				s = strings.TrimSpace(s)

				if s == "" {
					return nil
				}

				lines := strings.Split(s, "\n")
				out := make([]string, 0, len(lines))

				for _, l := range lines {
					l = strings.TrimSpace(l)

					if l != "" {
						out = append(out, l)
					}
				}

				return out
			},
			"jsDocTitle": func(s string) string {
				s = strings.TrimSpace(s)
				if s == "" {
					return ""
				}

				lines := strings.Split(s, "\n")

				return strings.TrimSpace(lines[0])
			},
			"quote": func(s string) string {
				if quoteOpt == "single" {
					s = strings.ReplaceAll(s, `\`, `\\`)
					s = strings.ReplaceAll(s, `'`, `\'`)

					return "'" + s + "'"
				}

				return strconv.Quote(s)
			},
		}).
		Parse(templates.EnumsTSTemplate),
	)

	var out bytes.Buffer

	must(tpl.Execute(&out, data))

	must(os.MkdirAll(filepath.Dir(outPath), 0o755))
	must(os.WriteFile(outPath, out.Bytes(), 0o644))
}

// ---------------- YAML parsing ----------------

func loadYAML(path string) (*yaml.Node, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var n yaml.Node

	dec := yaml.NewDecoder(bytes.NewReader(b))

	dec.KnownFields(false)

	if err := dec.Decode(&n); err != nil {
		return nil, err
	}

	return &n, nil
}

func extractEnums(root *yaml.Node) []EnumSchema {
	// root: DocumentNode -> MappingNode
	doc := root

	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}

	schemas := mapGet(mapGet(doc, "components"), "schemas")

	if schemas == nil || schemas.Kind != yaml.MappingNode {
		return nil
	}

	var result []EnumSchema

	for i := 0; i+1 < len(schemas.Content); i += 2 {
		nameNode := schemas.Content[i]
		schemaNode := schemas.Content[i+1]

		if nameNode == nil || schemaNode == nil || schemaNode.Kind != yaml.MappingNode {
			continue
		}

		name := nameNode.Value

		if name == "" {
			continue
		}

		typ := scalarValue(mapGet(schemaNode, "type"))
		enumNode := mapGet(schemaNode, "enum")

		// string enum only
		if typ != "string" || enumNode == nil || enumNode.Kind != yaml.SequenceNode || len(enumNode.Content) == 0 {
			continue
		}

		desc := scalarValue(mapGet(schemaNode, "description"))

		items := make([]EnumItem, 0, len(enumNode.Content))
		usedKeys := map[string]int{}

		for _, n := range enumNode.Content {
			if n == nil || n.Kind != yaml.ScalarNode {
				continue
			}

			val := n.Value

			label := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(n.LineComment), "#"))

			if label == "" {
				label = val
			}

			key := toTSMemberKey(val)

			if c, ok := usedKeys[key]; ok {
				c++

				usedKeys[key] = c

				key = fmt.Sprintf("%s_%d", key, c)
			} else {
				usedKeys[key] = 0
			}

			items = append(items, EnumItem{
				Value: val,
				Key:   key,
				Label: label,
			})
		}

		if len(items) == 0 {
			continue
		}

		dep := mapGet(schemaNode, "deprecated")

		deprecated := false

		if dep != nil && dep.Kind == yaml.ScalarNode && dep.Value == "true" {
			deprecated = true
		}

		since := scalarValue(mapGet(schemaNode, "x-since"))

		result = append(result, EnumSchema{
			Name:        name,
			Description: desc,
			Deprecated:  deprecated,
			Since:       since,
			See:         "",
			Items:       items,
		})
	}

	return result
}

func mapGet(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(m.Content); i += 2 {
		k := m.Content[i]
		v := m.Content[i+1]

		if k != nil && k.Value == key {
			return v
		}
	}

	return nil
}

func scalarValue(n *yaml.Node) string {
	if n == nil || n.Kind != yaml.ScalarNode {
		return ""
	}

	return n.Value
}

// ---------------- naming ----------------

func toTSMemberKey(value string) string {
	value = strings.TrimSpace(value)

	if value == "" {
		return "Value"
	}

	re := regexp.MustCompile(`[A-Za-z0-9]+`)
	parts := re.FindAllString(value, -1)

	if len(parts) == 0 {
		return "Value"
	}

	for i, p := range parts {
		parts[i] = upperFirst(strings.ToLower(p))
	}
	key := strings.Join(parts, "")

	// starts with digit
	if key != "" && key[0] >= '0' && key[0] <= '9' {
		key = "_" + key
	}

	// reserved-ish identifiers
	if isReservedTSIdent(key) {
		key += "_"
	}

	return key
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}

	r := []rune(s)
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]

	return string(r)
}

// minimal reserved list (enough to avoid obvious breakage)
func isReservedTSIdent(s string) bool {
	switch s {
	case "Default", "Class", "Function", "Var", "Let", "Const", "Enum",
		"Export", "Import", "Type", "Interface", "Extends", "Implements",
		"Public", "Private", "Protected", "New", "Delete", "Return",
		"Switch", "Case", "For", "While", "If", "Else", "Try", "Catch",
		"Finally", "Throw", "In", "Of", "This", "Super",
		"Null", "True", "False", "Void", "Any", "Never", "Unknown":
		return true
	default:
		return false
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
