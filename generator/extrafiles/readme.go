package extrafiles

import (
	"bytes"
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type chart struct {
	Name        string
	Description string
	Values      []string
}

//go:embed readme.tpl
var readmeTemplate string

// ReadMeFile returns the content of the README.md file.
func ReadMeFile(charname, description string, values map[string]any) string {
	// values is a yaml structure with keys and structured values...
	// we want to make list of dot separated keys and their values

	vv := map[string]any{}
	out, _ := yaml.Marshal(values)
	yaml.Unmarshal(out, &vv)

	result := make(map[string]string)
	parseValues("", vv, result)

	funcMap := template.FuncMap{
		"repeat": func(s string, count int) string {
			return strings.Repeat(s, count)
		},
	}
	tpl, err := template.New("readme").Funcs(funcMap).Parse(readmeTemplate)
	if err != nil {
		panic(err)
	}

	valuesLines := []string{}
	maxParamLen := 0
	maxDefaultLen := 0
	for key, value := range result {
		if len(key) > maxParamLen {
			maxParamLen = len(key)
		}
		if len(value) > maxDefaultLen {
			maxDefaultLen = len(value)
		}
	}
	for key, value := range result {
		valuesLines = append(valuesLines, fmt.Sprintf("| %-*s | %-*s |", maxParamLen, key, maxDefaultLen, value))
	}
	sort.Strings(valuesLines)

	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, map[string]any{
		"DescrptionPadding": maxParamLen,
		"DefaultPadding":    maxDefaultLen,
		"Chart": chart{
			Name:        charname,
			Description: description,
			Values:      valuesLines,
		},
	})
	if err != nil {
		panic(err)
	}

	return buf.String()
}

func parseValues(prefix string, values map[string]interface{}, result map[string]string) {
	for key, value := range values {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		switch v := value.(type) {
		case []interface{}:
			for i, u := range v {
				parseValues(fmt.Sprintf("%s[%d]", path, i), map[string]interface{}{"value": u}, result)
			}
		case map[string]interface{}:
			parseValues(path, v, result)
		default:
			strValue := fmt.Sprintf("`%v`", value)
			result["`"+path+"`"] = strValue
		}
	}
}
