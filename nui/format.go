package nui

import (
	"fmt"
	"io"
)

type Color int8

const (
	Black   Color = 0
	Red     Color = 1
	Green   Color = 2
	Yellow  Color = 3
	Blue    Color = 4
	Magenta Color = 5
	Cyan    Color = 6
	White   Color = 7
	Default Color = 9

	LightBlack   Color = 60
	LightRed     Color = 61
	LightGreen   Color = 62
	LightYellow  Color = 63
	LightBlue    Color = 64
	LightMagenta Color = 65
	LightCyan    Color = 66
	LightWhite   Color = 67
)

type Format struct {
	Fg        Color
	Bg        Color
	Bold      bool
	Underline bool
}

func (f Format) Apply(w io.Writer) {
	if f.Bold {
		fmt.Fprint(w, "\x1b[1m")
	} else {
		fmt.Fprint(w, "\x1b[22m")
	}
	if f.Underline {
		fmt.Fprint(w, "\x1b[4m")
	} else {
		fmt.Fprint(w, "\x1b[24m")
	}
	fmt.Fprintf(w, "\x1b[%dm", int(f.Fg)+30)
	fmt.Fprintf(w, "\x1b[%dm", int(f.Bg)+40)
}
