package generator

import (
	_ "embed"
	"reflect"
	"testing"
)

var testingKatenaryPrefix = Prefix()

func TestPrefix(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "TestPrefix",
			want: "katenary.v3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Prefix(); got != tt.want {
				t.Errorf("Prefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_labelName(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want Label
	}{
		{
			name: "Test_labelName",
			args: args{
				name: "main-app",
			},
			want: testingKatenaryPrefix + "/main-app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := labelName(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("labelName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLabelHelp(t *testing.T) {
	help := GetLabelHelp(false)
	if help == "" {
		t.Errorf("GetLabelHelp() = %v, want %v", help, "Help")
	}
	help = GetLabelHelp(true)
	if help == "" {
		t.Errorf("GetLabelHelp() = %v, want %v", help, "Help")
	}
}

func TestGetLabelHelpFor(t *testing.T) {
	help := GetLabelHelpFor("main-app", false)
	if help == "" {
		t.Errorf("GetLabelHelpFor() = %v, want %v", help, "Help")
	}
	help = GetLabelHelpFor("main-app", true)
	if help == "" {
		t.Errorf("GetLabelHelpFor() = %v, want %v", help, "Help")
	}
}
