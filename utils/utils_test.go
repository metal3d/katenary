package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestTplName(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		serviceName string
		appname     string
		suffix      []string
		want        string
	}{
		{"simple test without suffix", "foosvc", "myapp", nil, `{{ include "myapp.fullname" . }}-foosvc`},
		{"simple test with suffix", "foosvc", "myapp", []string{"bar"}, `{{ include "myapp.fullname" . }}-foosvc-bar`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TplName(tt.serviceName, tt.appname, tt.suffix...)
			if got != tt.want {
				t.Errorf("TplName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountStartingSpaces(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		line string
		want int
	}{
		{
			"test no spaces",
			"the line is here",
			0,
		},
		{
			"test with 4 spaces",
			"    line with 4 spaces",
			4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountStartingSpaces(tt.line)
			if got != tt.want {
				t.Errorf("CountStartingSpaces() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetKind(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		path string
		want string
	}{
		{
			"test get kind from file path",
			"my.deployment.yaml",
			"deployment",
		},
		{
			"test with 2 parts",
			"service.yaml",
			"service",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetKind(tt.path)
			if got != tt.want {
				t.Errorf("GetKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		src   string
		above string
		below string
		want  string
	}{
		{
			"test a simple wrap",
			"    - foo: bar",
			"line above",
			"line below",
			"    line above\n    - foo: bar\n    line below",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Wrap(tt.src, tt.above, tt.below)
			if got != tt.want {
				t.Errorf("Wrap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetServiceNameByPort(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		port int
		want string
	}{
		{
			"test http port by service number 80",
			80,
			"http",
		},
		{
			"test with a port that has no service name",
			8745,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetServiceNameByPort(tt.port)
			if got != tt.want {
				t.Errorf("GetServiceNameByPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetContainerByName(t *testing.T) {
	httpContainer := &corev1.Container{
		Name: "http",
	}
	mariadbContainer := &corev1.Container{
		Name: "mariadb",
	}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		containerName string
		containers    []corev1.Container
		want          *corev1.Container
		want2         int
	}{
		{
			"get container from by name",
			"http",
			[]corev1.Container{
				*httpContainer,
				*mariadbContainer,
			},
			httpContainer, 0,
		},
		{
			"get container from by name",
			"mariadb",
			[]corev1.Container{
				*httpContainer,
				*mariadbContainer,
			},
			mariadbContainer, 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got2 := GetContainerByName(tt.containerName, tt.containers)
			if got.Name != tt.want.Name {
				t.Errorf("GetContainerByName() = %v, want %v", got.Name, tt.want.Name)
			}
			if got2 != tt.want2 {
				t.Errorf("GetContainerByName() = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestTplValue(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		serviceName string
		variable    string
		pipes       []string
		want        string
	}{
		{
			"check simple template value",
			"foosvc",
			"variableFoo",
			nil,
			"{{ tpl .Values.foosvc.variableFoo $ }}",
		},
		{
			"check with pipes",
			"foosvc",
			"bar",
			[]string{"toYaml", "nindent 2"},
			"{{ tpl .Values.foosvc.bar $ | toYaml | nindent 2 }}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TplValue(tt.serviceName, tt.variable, tt.pipes...)
			if got != tt.want {
				t.Errorf("TplValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathToName(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		path string
		want string
	}{
		{
			"check complete path with various characters",
			"./foo/bar.test/and_bad_name",
			"foo-bar-test-and-bad-name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PathToName(tt.path)
			if got != tt.want {
				t.Errorf("PathToName() = %v, want %v", got, tt.want)
			}
		})
	}
}
