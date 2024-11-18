package labels

import (
	_ "embed"
	"reflect"
	"testing"
)

var testingKatenaryPrefix = Prefix()

const mainAppLabel = "main-app"

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

func TestLabelName(t *testing.T) {
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
				name: mainAppLabel,
			},
			want: testingKatenaryPrefix + "/" + mainAppLabel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LabelName(tt.args.name); !reflect.DeepEqual(got, tt.want) {
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
	help := GetLabelHelpFor(mainAppLabel, false)
	if help == "" {
		t.Errorf("GetLabelHelpFor() = %v, want %v", help, "Help")
	}
	help = GetLabelHelpFor("main-app", true)
	if help == "" {
		t.Errorf("GetLabelHelpFor() = %v, want %v", help, "Help")
	}
}
