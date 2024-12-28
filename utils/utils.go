package utils

import (
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/mitchellh/go-wordwrap"
	"github.com/thediveo/netdb"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

// TplName returns the name of the kubernetes resource as a template string.
// It is used in the templates and defined in _helper.tpl file.
func TplName(serviceName, appname string, suffix ...string) string {
	if len(suffix) > 0 {
		suffix[0] = "-" + suffix[0]
	}
	for i, s := range suffix {
		// replae all "_" with "-"
		suffix[i] = strings.ReplaceAll(s, "_", "-")
	}
	serviceName = strings.ReplaceAll(serviceName, "_", "-")
	return `{{ include "` + appname + `.fullname" . }}-` + serviceName + strings.Join(suffix, "-")
}

// Int32Ptr returns a pointer to an int32.
func Int32Ptr(i int32) *int32 { return &i }

// StrPtr returns a pointer to a string.
func StrPtr(s string) *string { return &s }

// CountStartingSpaces counts the number of spaces at the beginning of a string.
func CountStartingSpaces(line string) int {
	count := 0
	for _, char := range line {
		if char == ' ' {
			count++
		} else {
			break
		}
	}
	return count
}

// GetKind returns the kind of the resource from the file path.
func GetKind(path string) (kind string) {
	defer func() {
		if r := recover(); r != nil {
			kind = ""
		}
	}()
	filename := filepath.Base(path)
	parts := strings.Split(filename, ".")
	if len(parts) == 2 {
		kind = parts[0]
	} else {
		kind = strings.Split(path, ".")[1]
	}
	return
}

// Wrap wraps a string with a string above and below. It will respect the indentation of the src string.
func Wrap(src, above, below string) string {
	spaces := strings.Repeat(" ", CountStartingSpaces(src))
	return spaces + above + "\n" + src + "\n" + spaces + below
}

// GetServiceNameByPort returns the service name for a port. It the service name is not found, it returns an empty string.
func GetServiceNameByPort(port int) string {
	name := ""
	info := netdb.ServiceByPort(port, "tcp")
	if info != nil {
		name = info.Name
	}
	return name
}

// GetContainerByName returns a container by name and its index in the array. It returns nil, -1 if not found.
func GetContainerByName(name string, containers []corev1.Container) (*corev1.Container, int) {
	for index, c := range containers {
		if c.Name == name {
			return &c, index
		}
	}
	return nil, -1
}

// GetContainerByName returns a container by name and its index in the array.
func TplValue(serviceName, variable string, pipes ...string) string {
	if len(pipes) == 0 {
		return `{{ tpl .Values.` + serviceName + `.` + variable + ` $ }}`
	} else {
		return `{{ tpl .Values.` + serviceName + `.` + variable + ` $ | ` + strings.Join(pipes, " | ") + ` }}`
	}
}

// PathToName converts a path to a kubernetes complient name.
func PathToName(path string) string {
	if len(path) == 0 {
		return ""
	}

	path = filepath.Clean(path)
	if path[0] == '/' || path[0] == '.' {
		path = path[1:]
	}
	path = strings.ReplaceAll(path, "_", "-")
	path = strings.ReplaceAll(path, "/", "-")
	path = strings.ReplaceAll(path, ".", "-")
	path = strings.ToLower(path)
	return path
}

// EnvConfig is a struct to hold the description of an environment variable.
type EnvConfig struct {
	Service     types.ServiceConfig
	Description string
}

// GetValuesFromLabel returns a map of values from a label.
func GetValuesFromLabel(service types.ServiceConfig, LabelValues string) map[string]*EnvConfig {
	descriptions := make(map[string]*EnvConfig)
	if v, ok := service.Labels[LabelValues]; ok {
		labelContent := []any{}
		err := yaml.Unmarshal([]byte(v), &labelContent)
		if err != nil {
			log.Printf("Error parsing label %s: %s", v, err)
			log.Fatal(err)
		}
		for _, value := range labelContent {
			switch val := value.(type) {
			case string:
				descriptions[val] = nil
			case map[string]interface{}:
				for k, v := range value.(map[string]interface{}) {
					descriptions[k] = &EnvConfig{Service: service, Description: v.(string)}
				}
			case map[interface{}]interface{}:
				for k, v := range value.(map[interface{}]interface{}) {
					descriptions[k.(string)] = &EnvConfig{Service: service, Description: v.(string)}
				}
			default:
				log.Fatalf("Unknown type in label: %s %T", LabelValues, value)
			}
		}
	}
	return descriptions
}

// WordWrap wraps a string to a given line width. Warning: it may break the string. You need to check the result.
func WordWrap(text string, lineWidth int) string {
	return wordwrap.WrapString(text, uint(lineWidth))
}

// Confirm asks a question and returns true if the answer is y.
func Confirm(question string, icon ...Icon) bool {
	if len(icon) > 0 {
		fmt.Printf("%s %s [y/N] ", icon[0], question)
	} else {
		fmt.Print(question + " [y/N] ")
	}
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y"
}

// EncodeBasicYaml encodes a basic yaml from an interface.
func EncodeBasicYaml(data any) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FixedResourceName returns a resource name without underscores to respect the kubernetes naming convention.
func FixedResourceName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

// AsResourceName returns a resource name with underscores to respect the kubernetes naming convention.
// It's the opposite of FixedResourceName.
func AsResourceName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}
