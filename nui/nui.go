package nui

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func clear(conn net.Conn) {
	fmt.Fprint(conn, "\x1bc\x1b[49m\x1b[H\x1b[2J\x1b[3J")
}

// Type Buffer represents information about
// the output of a terminal screen.
type Buffer struct {
	Chars   []byte
	Formats []Format
	Width   uint16

	CursorX, CursorY uint16
	CursorFormat     Format
}

// Buffer initialized to all spaces, with the given background color
// and a Default foreground.
func emptyBuffer(width uint16, height uint16, bg Color) *Buffer {
	buffer := &Buffer{
		Width:   width,
		Formats: make([]Format, width*height),
		Chars:   make([]byte, width*height),
	}

	for i, _ := range buffer.Formats {
		buffer.Formats[i] = Format{Fg: Default, Bg: bg}
	}
	for i, _ := range buffer.Chars {
		buffer.Chars[i] = ' '
	}
	buffer.CursorFormat = Format{Fg: Default, Bg: bg}
	return buffer
}

func (b Buffer) Index(x uint16, y uint16) int {
	return int(y)*int(b.Width) + int(x)
}

type Widget interface {
	// Draws the widget.
	// If the widget is focusable, should also
	// set the correct cursor position.
	Draw(buf *Buffer)
}

// Represents a widget that can have focus, during
// which events are delivered. When of these methods are called,
// the screen belonging to the widget is write-locked.
type FocusableWidget interface {
	Widget
	Focus(focus bool)

	// Called when a key is pressed.
	Keypress(byte)
}

type Screen struct {
	Widgets []Widget
	Focus   int

	// This field should be locked whenever
	// any of the other fields of the Screen
	// are being read or written to.
	sync.RWMutex
}

func (s *Screen) Draw(conn net.Conn, oldBuffer *Buffer) *Buffer {
	newBuffer := emptyBuffer(oldBuffer.Width, uint16(len(oldBuffer.Chars))/oldBuffer.Width, Black)

	for idx, widget := range s.Widgets {
		if idx == s.Focus {
			continue
		}

		widget.Draw(newBuffer)
	}

	if s.Focus >= 0 {
		s.Widgets[s.Focus].Draw(newBuffer)
	}

	// diff
	var prevFormat Format
	//var prevX, prevY uint16
	firstDraw := true
	msg := new(strings.Builder)
	for idx := 0; idx < len(oldBuffer.Chars); idx++ {
		if oldBuffer.Chars[idx] != newBuffer.Chars[idx] || oldBuffer.Formats[idx] != newBuffer.Formats[idx] {
			x := uint16(idx % int(oldBuffer.Width))
			y := uint16(idx / int(oldBuffer.Width))

			//if firstDraw || !(prevX+1 == x && prevY == y) {
			//	fmt.Fprintf(msg, "\x1b[%d;%dH", y+1, x+1)
			//}
			if firstDraw || prevFormat != newBuffer.Formats[idx] {
				newBuffer.Formats[idx].Apply(msg)
			}

			fmt.Fprintf(msg, "\x1b[%d;%dH", y+1, x+1)
			//newBuffer.Formats[idx].Apply(msg)
			fmt.Fprintf(msg, "%c", newBuffer.Chars[idx])

			//prevX = x
			//prevY = y
			prevFormat = newBuffer.Formats[idx]
			//
			firstDraw = false
		}
	}

	newBuffer.CursorFormat.Apply(msg)
	fmt.Fprintf(msg, "\x1b[%d;%dH", newBuffer.CursorY+1, newBuffer.CursorX+1)
	fmt.Fprint(conn, msg.String())

	return newBuffer
}

type Server struct {
	ln      net.Listener
	screens sync.Map /* int => Screen */

	TermWidth  uint16
	TermHeight uint16

	// Called when a new client connects.
	// This function should call SetScreen
	// and set the screen of the given client.
	HandleConnect    func(clientID int)
	HandleDisconnect func(clientID int)
}

func NewServer(ln net.Listener) *Server {
	return &Server{
		ln:         ln,
		TermWidth:  64,
		TermHeight: 48,
	}
}

// Set the screen of a particular client ID
func (s *Server) SetScreen(clientID int, screen *Screen) {
	s.screens.Store(clientID, screen)
}

// Get the screen of a particular client ID
func (s *Server) GetScreen(clientID int) (*Screen, bool) {
	v, ok := s.screens.Load(clientID)
	if !ok {
		return nil, false
	}
	return v.(*Screen), true
}

func (s *Server) Run() {
	clients := 0
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
		}
		clients += 1
		go s.connThread(conn, clients-1)
	}
}

// Run in a different thread. Mem safety: This function does not
// write to s.screens and does not read or write from s.ln or s.HandleConnect.
// This function only accesses s.screens[clientID] and not any other key-value pair.
func (s *Server) connThread(conn net.Conn, clientID int) {
	s.HandleConnect(clientID)

	// Send clear escape codes & codes to listen for mouse events
	clear(conn)
	//	fmt.Fprint(conn, "\x1b[?1000h") // mouse events

	screenI, ok := s.screens.Load(clientID)
	if !ok {
		log.Println("no screen found for client ID: ", clientID)
	}
	screen := screenI.(*Screen)

	buffer := emptyBuffer(s.TermWidth, s.TermHeight, Default)
	buffer = screen.Draw(conn, buffer)

	buf := make([]byte, 1)
	for {
		conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		_, err := conn.Read(buf)
		if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
			break
		}

		screenI, ok := s.screens.Load(clientID)
		if !ok {
			continue
		}
		screen := screenI.(*Screen)

		if err == nil {
			screen.Lock()
			c := buf[0]
			if c == '\t' { // TAB: set focus to next widget
				if screen.Focus >= 0 {
					if widget, focusable := screen.Widgets[screen.Focus].(FocusableWidget); focusable {
						screen.Unlock()
						widget.Focus(true)
						screen.Lock()
					}
				}

				endIdx := (screen.Focus + 1) % len(screen.Widgets)
				if endIdx < 0 {
					endIdx = 0
				}

				first := true
				for i := endIdx; first || i != endIdx; i = (i + 1) % len(screen.Widgets) {
					if widget, focusable := screen.Widgets[i].(FocusableWidget); focusable {
						widget.Focus(true)

						screen.Focus = i
						break
					}
				}
			} else if c == '\x1b' {
				// TODO
			} else {
				if screen.Focus >= 0 {
					if widget, focusable := screen.Widgets[screen.Focus].(FocusableWidget); focusable {
						widget.Keypress(c)

						//						log.Printf("delivering keypress to widget: %d", screen.Focus)
					} else {
						log.Println("warning: Focus for client", clientID, "is set to a non-focusable widget", screen.Focus)
					}
				}
			}
			screen.Unlock()
		}

		screen.RLock()
		buffer = screen.Draw(conn, buffer)
		screen.RUnlock()
	}

	s.HandleDisconnect(clientID)
}
