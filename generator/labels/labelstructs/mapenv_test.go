package labelstructs

import "testing"

func TestConfigMapLabel(t *testing.T) {
	ts := "foo: bar"
	tc, _ := MapEnvFrom(ts)
	if len(tc) != 1 {
		t.Errorf("Expected ConfigMapFile to have 1 item, got %d", len(tc))
	}
}
