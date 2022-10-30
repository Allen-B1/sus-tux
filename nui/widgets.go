package nui

import (
	"unicode"
)

type Label struct {
	X      uint16
	Y      uint16
	Format Format
	Text   string
}

func (l *Label) Draw(buf *Buffer) {
	idx := buf.Index(l.X, l.Y)
	for i, c := range []byte(l.Text) {
		buf.Chars[idx+i] = c
		buf.Formats[idx+i] = l.Format
	}
}

// Represents an entry.
//
// Note: When the event handlers are called,
// the screen that this widget belongs to is write-locked.
type Entry struct {
	X      uint16
	Y      uint16
	Format Format
	Text   string
	Max    int

	HandleInput func(text string)
	HandleEnter func(text string)
}

func (e *Entry) Draw(buf *Buffer) {
	idx := buf.Index(e.X, e.Y)
	for i := 0; i < e.Max; i++ {
		c := byte(' ')
		if i < len(e.Text) {
			c = e.Text[i]
		}

		buf.Chars[idx+i] = c
		buf.Formats[idx+i] = e.Format
	}

	buf.CursorX = e.X + uint16(len(e.Text))
	buf.CursorY = e.Y
	buf.CursorFormat = e.Format
}

func (e *Entry) Focus(focus bool) {}

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

// Represents a button.
//
// Note: When the event handlers are called,
// the screen that this widget belongs to is write-locked.
type Button struct {
	X      uint16
	Y      uint16
	Format Format
	Text   string

	HandleClick func()
}

func (b *Button) Draw(buf *Buffer) {
	for x := b.X; x < b.X+uint16(len(b.Text))+4; x++ {
		idx := buf.Index(x, b.Y)
		buf.Chars[idx] = ' '
		buf.Formats[idx] = b.Format
	}

	for x := b.X; x < b.X+uint16(len(b.Text))+4; x++ {
		idx := buf.Index(x, b.Y+1)
		buf.Formats[idx] = b.Format

		if x < b.X+2 || x >= b.X+2+uint16(len(b.Text)) {
			buf.Chars[idx] = ' '
		} else {
			buf.Chars[idx] = b.Text[int(x-b.X-2)]
		}
	}

	for x := b.X; x < b.X+uint16(len(b.Text))+4; x++ {
		idx := buf.Index(x, b.Y+2)
		buf.Chars[idx] = ' '
		buf.Formats[idx] = b.Format
	}

	buf.CursorX = b.X + 2 + uint16(len(b.Text))
	buf.CursorY = b.Y + 1
	buf.CursorFormat = b.Format
}

func (b *Button) Focus(focus bool) {}
func (b *Button) Keypress(ch byte) {
	if ch == '\n' && b.HandleClick != nil {
		b.HandleClick()
	}
}
