package labelstructs

import "testing"

func TestEnvFromLabel(t *testing.T) {
	ts := "- foo\n- bar"
	tc, _ := EnvFromFrom(ts)
	if len(tc) != 2 {
		t.Errorf("Expected EnvFrom to have 2 items, got %d", len(tc))
	}
	if tc[0] != "foo" {
		t.Errorf("Expected EnvFrom to contain 'foo', got %s", tc[0])
	}
	if tc[1] != "bar" {
		t.Errorf("Expected EnvFrom to contain 'bar', got %s", tc[1])
	}
}
