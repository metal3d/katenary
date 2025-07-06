package labelstructs

import (
	"testing"
)

func TestValueFromLabel(t *testing.T) {
	data := "data: foo\ndata2: bar"
	tc, err := GetValueFrom(data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tc == nil {
		t.Fatalf("expected non-nil map, got nil")
	}
	if len(*tc) != 2 {
		t.Errorf("expected 2 items, got %d", len(*tc))
	}
	if (*tc)["data"] != "foo" {
		t.Errorf("expected 'data' to be 'foo', got %s", (*tc)["data"])
	}
	if (*tc)["data2"] != "bar" {
		t.Errorf("expected 'data2' to be 'bar', got %s", (*tc)["data2"])
	}
}
