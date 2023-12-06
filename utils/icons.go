package utils

import "fmt"

// Icon is a unicode icon
type Icon string

// Icons used in katenary.
const (
	IconSuccess    Icon = "✅"
	IconFailure         = "❌"
	IconWarning         = "⚠️'"
	IconNote            = "📝"
	IconWorld           = "🌐"
	IconPlug            = "🔌"
	IconPackage         = "📦"
	IconCabinet         = "🗄️"
	IconInfo            = "❕"
	IconSecret          = "🔒"
	IconConfig          = "🔧"
	IconDependency      = "🔗"
)

// Warn prints a warning message
func Warn(msg ...interface{}) {
	orange := "\033[38;5;214m"
	reset := "\033[0m"
	fmt.Print(IconWarning, orange, " ")
	fmt.Print(msg...)
	fmt.Println(reset)
}
