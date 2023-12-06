package generator

// DataMap is a kubernetes ConfigMap or Secret. It can be used to add data to the ConfigMap or Secret.
type DataMap interface {
	SetData(map[string]string)
	AddData(string, string)
}

// Yaml is a kubernetes object that can be converted to yaml.
type Yaml interface {
	Yaml() ([]byte, error)
	Filename() string
}
