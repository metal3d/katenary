package generator

import (
	"encoding/base64"
	"katenary/generator/labels"
	"katenary/utils"
	"strings"

	"github.com/compose-spec/compose-go/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	_ DataMap = (*Secret)(nil)
	_ Yaml    = (*Secret)(nil)
)

// Secret is a kubernetes Secret.
//
// Implements the DataMap interface.
type Secret struct {
	*corev1.Secret
	service types.ServiceConfig `yaml:"-"`
}

// NewSecret creates a new Secret from a compose service
func NewSecret(service types.ServiceConfig, appName string) *Secret {
	secret := &Secret{
		service: service,
		Secret: &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        utils.TplName(service.Name, appName),
				Labels:      GetLabels(service.Name, appName),
				Annotations: Annotations,
			},
			Data: make(map[string][]byte),
		},
	}

	// check if the value should be in values.yaml
	valueList := []string{}
	varDescriptons := utils.GetValuesFromLabel(service, labels.LabelValues)
	for value := range varDescriptons {
		valueList = append(valueList, value)
	}

	for _, value := range valueList {
		if val, ok := service.Environment[value]; ok {
			value = strings.TrimPrefix(value, `"`)
			*val = `.Values.` + service.Name + `.environment.` + value
		}
	}

	for key, value := range service.Environment {
		if value == nil {
			continue
		}
		secret.AddData(key, *value)
	}

	return secret
}

// AddData adds a key value pair to the secret.
func (s *Secret) AddData(key, value string) {
	if value == "" {
		return
	}
	valuesLabels := utils.GetValuesFromLabel(s.service, labels.LabelValues)
	if _, ok := valuesLabels[key]; ok {
		// the value should be in values.yaml
		s.Data[key] = []byte(`{{ tpl .Values.` + s.service.Name + `.environment.` + key + ` $ | b64enc }}`)
	} else {
		encoded := base64.StdEncoding.EncodeToString([]byte(value))
		s.Data[key] = []byte(encoded)
	}
	// s.Data[key] = []byte(`{{ tpl ` + value + ` $ | b64enc }}`)
}

// Filename returns the filename of the secret.
func (s *Secret) Filename() string {
	return s.service.Name + ".secret.yaml"
}

// SetData sets the data of the secret.
func (s *Secret) SetData(data map[string]string) {
	for key, value := range data {
		s.AddData(key, value)
	}
}

// Yaml returns the yaml representation of the secret.
func (s *Secret) Yaml() ([]byte, error) {
	var y []byte
	var err error
	if y, err = ToK8SYaml(s); err != nil {
		return nil, err
	}

	// replace the b64 value by the real value
	for _, value := range s.Data {
		encoded := base64.StdEncoding.EncodeToString([]byte(value))
		y = []byte(strings.ReplaceAll(string(y), encoded, string(value)))
	}

	return y, nil
}
