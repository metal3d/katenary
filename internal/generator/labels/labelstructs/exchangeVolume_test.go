package labelstructs

import "testing"

func TestExchangeVolumeLabel(t *testing.T) {
	ts := "- name: exchange-volume\n  mountPath: /exchange\n  readOnly: true"
	tc, _ := NewExchangeVolumes(ts)
	if len(tc) != 1 {
		t.Errorf("Expected ExchangeVolumeLabel to have 1 item, got %d", len(tc))
	}
	if tc[0].Name != "exchange-volume" {
		t.Errorf("Expected ExchangeVolumeLabel to contain 'exchange-volume', got %s", tc[0].Name)
	}
	if tc[0].MountPath != "/exchange" {
		t.Errorf("Expected MountPath to be '/exchange', got %s", tc[0].MountPath)
	}
}
