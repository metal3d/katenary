package labelstructs

import "testing"

func TestSecretLabel(t *testing.T) {
	data := "- foo\n- bar"
	tc, err := SecretsFrom(data)
	if err != nil {
		t.Errorf("Error parsing SecretLabel: %v %v", err, tc)
	}
	items := []string{"foo", "bar"}
	for i, item := range tc {
		if item != items[i] {
			t.Errorf("Expected SecretLabel to contain '%s', got '%s'", items[i], item)
		}
	}
}
