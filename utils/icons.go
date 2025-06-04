package utils

import "fmt"

// Icon is a unicode icon
type Icon string

// Icons used in katenary.
const (
	IconSuccess    Icon = "✅"
	IconFailure    Icon = "❌"
	IconWarning    Icon = "❕"
	IconNote       Icon = "📝"
	IconWorld      Icon = "🌐"
	IconPlug       Icon = "🔌"
	IconPackage    Icon = "📦"
	IconCabinet    Icon = "🗄️"
	IconInfo       Icon = "🔵"
	IconSecret     Icon = "🔒"
	IconConfig     Icon = "🔧"
	IconDependency Icon = "🔗"
)

// Warn prints a warning message
func Warn(msg ...any) {
	orange := "\033[38;5;214m"
	reset := "\033[0m"
	fmt.Print(IconWarning, orange, " ")
	fmt.Print(msg...)
	fmt.Println(reset)
}
