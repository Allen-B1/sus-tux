package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/allen-b1/sus-tux/nui"
)

type Player struct {
	name string
}

type State struct {
	// Map from client ID => player index.
	clients map[int]int
	players []Player

	// This field should be locked whenever
	// any other fields are being read or written to.
	sync.RWMutex
}

// Update all players' screens after changing a player's name.
// Memory safety: Locks the given state, as well as all screens except for the one corresponding
// to targetClientID.
func updateScreens(srv *nui.Server, state *State, targetClientID int, newName string) {
	state.RLock()
	defer state.RUnlock()

	targetIdx := state.clients[targetClientID]
	for clientID, _ := range state.clients {
		screen, ok := srv.GetScreen(clientID)
		if !ok {
			log.Println("warning: impossible situation")
			continue
		}

		if clientID != targetClientID {
			screen.Lock()
		}
		widget := screen.Widgets[targetIdx]
		switch w := widget.(type) {
		case *nui.Label:
			w.Text = newName
		case *nui.Entry:
			w.Text = newName
		}
		if clientID != targetClientID {
			screen.Unlock()
		}
	}
}

func makeScreen(srv *nui.Server, state *State, clientID int) *nui.Screen {
	screen := &nui.Screen{}
	for playerIdx, player := range state.players {
		if state.clients[clientID] != playerIdx {
			format := nui.Format{Fg: nui.Color(playerIdx + 61), Bg: nui.Black}
			label := &nui.Label{X: 8, Y: 5 + playerIdx, Format: format, Text: player.name, Max: 16}
			screen.Widgets = append(screen.Widgets, label)
		} else {
			playerIdx := playerIdx
			format := nui.Format{Fg: nui.Color(playerIdx + 61), Bg: nui.Black, Bold: true}
			entry := &nui.Entry{
				X: 8, Y: 5 + playerIdx, Format: format, Text: player.name, Max: 16,

				HandleInput: func(name string) {
					state.players[playerIdx].name = name
					updateScreens(srv, state, clientID, name)
				},
			}
			screen.Widgets = append(screen.Widgets, entry)
			screen.Focus = playerIdx
		}
	}

	format := nui.Format{Fg: nui.LightWhite, Bg: nui.Black, Underline: true}
	screen.Widgets = append(screen.Widgets, &nui.Label{X: 8, Y: 4, Format: format, Text: fmt.Sprintf("Players: %d", len(state.players))})
	return screen
}

func main() {
	var state = State{
		clients: make(map[int]int),
	}

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	srv := nui.NewServer(ln)
	srv.HandleConnect = func(clientID int) {
		log.Printf("event: connect [%d]\n", clientID)

		state.Lock()
		defer state.Unlock()

		state.clients[clientID] = len(state.players)
		state.players = append(state.players, Player{})

		screen := makeScreen(srv, &state, clientID)
		srv.SetScreen(clientID, screen)

		// create new screens for everyone
		for clientID, _ := range state.clients {
			screen := makeScreen(srv, &state, clientID)
			srv.SetScreen(clientID, screen)
		}
	}
	srv.HandleDisconnect = func(clientID int) {
		log.Printf("event: disconnect [%d]\n", clientID)

		idx, ok := state.clients[clientID]
		if !ok {
			log.Println("warning: non-existant player disconnected")
			return
		}

		delete(state.clients, clientID)
		if len(state.players) > 1 {
			state.players[idx] = state.players[len(state.clients)-1]
		}
		state.players = state.players[0 : len(state.players)-1]

		// create new screens for everyone
		for clientID, _ := range state.clients {
			screen := makeScreen(srv, &state, clientID)
			srv.SetScreen(clientID, screen)
		}
	}
	srv.Run()
}
