package nui

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type Screen struct {
	Widgets []Widget
	Focus   int

	// This field should be locked whenever
	// any of the other fields of the Screen
	// are being read or written to.
	sync.RWMutex
}

func (s *Screen) Draw(conn net.Conn) {
	for idx, widget := range s.Widgets {
		if idx == s.Focus {
			continue
		}

		widget.Draw(conn)
	}

	if s.Focus >= 0 {
		s.Widgets[s.Focus].Draw(conn)
	}
}

type Server struct {
	ln      net.Listener
	screens sync.Map /* int => Screen */
	clients int

	// Called when a new client connects.
	// This function should call SetScreen
	// and set the screen of the given client.
	HandleConnect    func(clientID int)
	HandleDisconnect func(clientID int)
}

func NewServer(ln net.Listener) *Server {
	return &Server{
		ln: ln,
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

// Returns the number of clients, connected & disconnected.
func (s *Server) Clients() int {
	return s.clients
}

func (s *Server) Run() {
	s.clients = 0
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			log.Println("error accepting connection:", err)
		}
		s.clients += 1
		go s.connThread(conn, s.clients-1)
	}
}

// Run in a different thread. Mem safety: This function does not
// write to s.screens and does not read or write from s.ln or s.HandleConnect.
// This function only accesses s.screens[clientID] and not any other key-value pair.
func (s *Server) connThread(conn net.Conn, clientID int) {
	s.HandleConnect(clientID)

	// Send clear escape codes & codes to listen for mouse events
	fmt.Fprint(conn, "\x1bc\x1b[40m\x1b[H\x1b[2J\x1b[3J")
	fmt.Fprint(conn, "\x1b[?1000h")

	screenI, ok := s.screens.Load(clientID)
	if !ok {
		log.Println("no screen found for client ID: ", clientID)
	}
	screen := screenI.(*Screen)
	screen.Draw(conn)

	buf := make([]byte, 1)
	for {
		conn.SetReadDeadline(time.Now().Add(time.Second))
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

				endIdx := screen.Focus
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
		screen.Draw(conn)
		screen.RUnlock()
	}

	s.HandleDisconnect(clientID)
}

type Widget interface {
	// Draws the widget.
	// If the widget is focusable, should also
	// set the correct cursor position.
	Draw(conn net.Conn)
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
