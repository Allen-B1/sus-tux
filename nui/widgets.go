package nui

import (
	"fmt"
	"net"
	"strings"
	"unicode"
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

func (f Format) Apply(conn net.Conn) {
	if f.Bold {
		fmt.Fprint(conn, "\x1b[1m")
	} else {
		fmt.Fprint(conn, "\x1b[22m")
	}
	if f.Underline {
		fmt.Fprint(conn, "\x1b[4m")
	} else {
		fmt.Fprint(conn, "\x1b[24m")
	}
	fmt.Fprintf(conn, "\x1b[%dm", int(f.Fg)+30)
	fmt.Fprintf(conn, "\x1b[%dm", int(f.Bg)+40)

}

type Label struct {
	X      int
	Y      int
	Format Format
	Text   string

	// If Max > 0, then the remaining
	// Max - len(Text) spots will be
	// filled with spaces
	Max int
}

func (l *Label) Draw(conn net.Conn) {
	fmt.Fprintf(conn, "\x1b[%d;%dH", l.Y, l.X)
	l.Format.Apply(conn)
	fmt.Fprint(conn, l.Text)
	if l.Max > 0 {
		fmt.Fprint(conn, strings.Repeat(" ", l.Max-len(l.Text)))
	}
}

// Represents an entry.
//
// Note: When the event handlers are called,
// the screen that this widget belongs to is write-locked.
type Entry struct {
	X       int
	Y       int
	Format  Format
	Text    string
	Focused bool
	Max     int

	HandleInput func(text string)
	HandleEnter func(text string)
}

func (e *Entry) Draw(conn net.Conn) {
	fmt.Fprintf(conn, "\x1b[%d;%dH", e.Y, e.X)
	e.Format.Apply(conn)
	fmt.Fprint(conn, e.Text)
	fmt.Fprint(conn, strings.Repeat(" ", e.Max-len(e.Text)))
	fmt.Fprintf(conn, "\x1b[%d;%dH", e.Y, e.X+len(e.Text))
}

func (e *Entry) Focus(focus bool) {
	if focus {
		e.Focused = true
	}
}

func (e *Entry) Keypress(ch byte) {
	if ch == '\b' || ch == 127 {
		if len(e.Text) != 0 {
			e.Text = e.Text[0 : len(e.Text)-1]
		}
		if e.HandleInput != nil {
			e.HandleInput(e.Text)
		}
	} else if unicode.IsPrint(rune(ch)) && len(e.Text) < e.Max {
		e.Text += string(rune(ch))
		if e.HandleInput != nil {
			e.HandleInput(e.Text)
		}
	} else if ch == '\n' {
		if e.HandleEnter != nil {
			e.HandleEnter(e.Text)
		}
	}
}
