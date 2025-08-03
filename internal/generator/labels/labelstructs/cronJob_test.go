package labelstructs

import "testing"

func TestCronJobFrom(t *testing.T) {
	ts := `
image: fooimage
command: thecommand
schedule: "0/3 0 * * *"
Rbac: false
`
	tc, _ := CronJobFrom(ts)
	if tc.Image != "fooimage" {
		t.Errorf("Expected CronJob image to be 'fooimage', got %s", tc.Image)
	}
	if tc.Command != "thecommand" {
		t.Errorf("Expected CronJob command to be 'thecommand', got %s", tc.Command)
	}
	if tc.Schedule != "0/3 0 * * *" {
		t.Errorf("Expected CronJob schedule to be '0/3 0 * * *', got %s", tc.Schedule)
	}
	if tc.Rbac != false {
		t.Errorf("Expected CronJob rbac to be false, got %t", tc.Rbac)
	}
}
