package generator

import (
	"katenary/utils"
	"strings"

	"github.com/compose-spec/compose-go/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const persistenceKey = "persistence"

var _ Yaml = (*VolumeClaim)(nil)

// VolumeClaim is a kubernetes VolumeClaim. This is a PersistentVolumeClaim.
type VolumeClaim struct {
	*v1.PersistentVolumeClaim
	service      *types.ServiceConfig `yaml:"-"`
	volumeName   string
	nameOverride string
}

// NewVolumeClaim creates a new VolumeClaim from a compose service.
func NewVolumeClaim(service types.ServiceConfig, volumeName, appName string) *VolumeClaim {
	fixedName := utils.FixedResourceName(volumeName)
	return &VolumeClaim{
		volumeName: volumeName,
		service:    &service,
		PersistentVolumeClaim: &v1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        utils.TplName(service.Name, appName) + "-" + fixedName,
				Labels:      GetLabels(service.Name, appName),
				Annotations: Annotations,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{
					v1.ReadWriteOnce,
				},
				StorageClassName: utils.StrPtr(
					`{{ .Values.` +
						service.Name +
						"." + persistenceKey +
						"." + volumeName + `.storageClass }}`,
				),
				Resources: v1.VolumeResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		},
	}
}

// Filename returns the suggested filename for a VolumeClaim.
func (v *VolumeClaim) Filename() string {
	return v.service.Name + "." + v.volumeName + ".volumeclaim.yaml"
}

// Yaml marshals a VolumeClaim into yaml.
func (v *VolumeClaim) Yaml() ([]byte, error) {
	serviceName := v.service.Name
	if v.nameOverride != "" {
		serviceName = v.nameOverride
	}
	volumeName := v.volumeName
	out, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}

	out = UnWrapTPL(out)

	// replace 1Gi to {{ .Values.serviceName.volume.size }}
	out = []byte(
		strings.Replace(
			string(out),
			"1Gi",
			utils.TplValue(serviceName, persistenceKey+"."+volumeName+".size"),
			1,
		),
	)

	out = []byte(
		strings.Replace(
			string(out),
			"- ReadWriteOnce",
			"{{- .Values."+
				serviceName+
				"."+persistenceKey+
				"."+volumeName+
				".accessMode | toYaml | nindent __indent__ }}",
			1,
		),
	)

	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if strings.Contains(line, "storageClass") {
			lines[i] = utils.Wrap(
				line,
				"{{- if ne .Values."+
					serviceName+
					"."+persistenceKey+
					"."+volumeName+".storageClass \"-\" }}",
				"{{- end }}",
			)
		}
	}
	out = []byte(strings.Join(lines, "\n"))

	// add condition
	out = []byte(
		"{{- if .Values." +
			serviceName +
			"." + persistenceKey +
			"." + volumeName +
			".enabled }}\n" +
			string(out) +
			"\n{{- end }}",
	)

	return out, nil
}
