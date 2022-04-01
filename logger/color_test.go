package logger

import "testing"

func TestColor(t *testing.T) {
	NOLOG = false
	Red("Red text")
	Grey("Grey text")
}
