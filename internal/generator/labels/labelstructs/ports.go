package labelstructs

import "gopkg.in/yaml.v3"

type Ports []uint32

// PortsFrom returns a Ports from the given string.
func PortsFrom(data string) (Ports, error) {
	var mapping Ports
	if err := yaml.Unmarshal([]byte(data), &mapping); err != nil {
		return nil, err
	}
	return mapping, nil
}
