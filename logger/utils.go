package logger

import (
	"fmt"
	"os"
	"sync"
)

type Color int

var ActivateColors = false
var NOLOG = false

const (
	GREY Color = 30 + iota
	RED
	GREEN
	YELLOW
	BLUE
	MAGENTA
	CYAN
)

var waiter = sync.Mutex{}

func color(c Color, args ...interface{}) {
	if NOLOG {
		return
	}
	if !ActivateColors {
		fmt.Println(args...)
		return
	}
	waiter.Lock()
	fmt.Fprintf(os.Stdout, "\x1b[%dm", c)
	fmt.Fprint(os.Stdout, args...)
	fmt.Fprintf(os.Stdout, "\x1b[0m\n")
	waiter.Unlock()
}

func colorf(c Color, format string, args ...interface{}) {
	if NOLOG {
		return
	}
	if !ActivateColors {
		fmt.Printf(format, args...)
		return
	}
	waiter.Lock()
	fmt.Fprintf(os.Stdout, "\x1b[%dm", c)
	fmt.Fprintf(os.Stdout, format, args...)
	fmt.Fprintf(os.Stdout, "\x1b[0m")
	waiter.Unlock()
}

func Grey(args ...interface{}) {
	color(GREY, args...)
}

func Red(args ...interface{}) {
	color(RED, args...)
}
func Green(args ...interface{}) {
	color(GREEN, args...)
}
func Yellow(args ...interface{}) {
	color(YELLOW, args...)
}
func Blue(args ...interface{}) {
	color(BLUE, args...)
}
func Magenta(args ...interface{}) {
	color(MAGENTA, args...)
}
func Greyf(format string, args ...interface{}) {
	colorf(GREY, format, args...)
}

func Redf(format string, args ...interface{}) {
	colorf(RED, format, args...)
}
func Greenf(format string, args ...interface{}) {
	colorf(GREEN, format, args...)
}
func Yellowf(format string, args ...interface{}) {
	colorf(YELLOW, format, args...)
}
func Bluef(format string, args ...interface{}) {
	colorf(BLUE, format, args...)
}
func Magentaf(format string, args ...interface{}) {
	colorf(MAGENTA, format, args...)
}
func Cyanf(format string, args ...interface{}) {
	colorf(CYAN, format, args...)
}
