package labelstructs

import "testing"

func TestProbesLabel(t *testing.T) {
	readiness := "readinessProbe:\n    httpGet:\n      path: /healthz\n      port: 8080\n    initialDelaySeconds: 5\n    periodSeconds: 10"
	tc, err := ProbeFrom(readiness)
	if err != nil {
		t.Errorf("Error parsing ProbesLabel: %v %v", err, tc)
	}
	liveness := "livenessProbe:\n    httpGet:\n      path: /healthz\n      port: 8080\n    initialDelaySeconds: 5\n    periodSeconds: 10"
	tc2, err := ProbeFrom(liveness)
	if err != nil {
		t.Errorf("Error parsing ProbesLabel: %v %v", err, tc2)
	}
}
