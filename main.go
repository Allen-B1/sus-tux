package main

import (
	_ "embed"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/allen-b1/sus-tux/nui"
)

//go:embed maps/research_facility.txt
var researchFacilityData string
var researchFacility *Map

func init() {
	researchFacility = NewMap(researchFacilityData)
}

type Player struct {
	name string
}

type State struct {
	// Map from client ID => player index.
	clients map[int]int
	players []Player

	game *Game

	// This field should be locked whenever
	// any other fields are being read or written to.
	sync.RWMutex
}

func makeGameScreen(state *State, playerIdx int) *nui.Screen {
	g := state.game
	player := &g.Players[playerIdx]
	return &nui.Screen{
		Focus: 0,
		Widgets: []nui.Widget{
			&MapWidget{
				X: 0, Y: 4, PlayerColor: nui.Color(playerIdx + 1), Map: g.Map,
				Player:  player,
				Players: state.game.Players,
				KillHandler: func() {
					if player.Dead || !player.Imposter {
						return
					}

					for i, target := range state.game.Players {
						if i == playerIdx {
							continue
						}
						if (target.X-player.X)*(target.X-player.X)+(target.Y-player.Y)*(target.Y-player.Y) <= KILL_RADIUS*KILL_RADIUS {
							g.Kill(i)
						}
					}
				},
			},
			&nui.Label{
				X: 2, Y: 1, Format: nui.Format{Fg: nui.Color(playerIdx + 1), Bg: nui.Black},
				Text: state.players[playerIdx].name,
			},
			&nui.Label{
				X: 2, Y: 2, Format: nui.Format{Fg: nui.White, Bg: nui.Black},
				Text: "Role:",
			},
			&nui.Label{
				X: 2 + 6, Y: 2, Format: nui.Format{Fg: ternaryColor(player.Imposter, nui.LightRed, nui.LightBlue), Bg: nui.Black, Bold: true},
				Text: ternaryString(player.Imposter, "Impostor", "Crewmate"),
			},
		},
	}
}

// Update all players' screens after changing a player's name.
// Memory safety: Locks the given state, as well as all screens except for the one corresponding
// to targetClientID.
func updateLobbyScreens(srv *nui.Server, state *State, targetClientID int, newName string) {
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

func startGame(srv *nui.Server, state *State) {
	state.Lock()
	defer state.Unlock()
	state.game = NewGame(len(state.players), researchFacility)

	go func() {
		var next <-chan time.Time
		for i := uint(0); true; i++ {
			next = time.After(time.Millisecond * 50)

			state.Lock()
			state.game.Update(i)
			state.Unlock()

			for clientID, playerIdx := range state.clients {
				srv.SetScreen(clientID, makeGameScreen(state, playerIdx))
			}

			<-next
		}
	}()
}

func makeLobbyScreen(srv *nui.Server, state *State, clientID int) *nui.Screen {
	screen := &nui.Screen{}
	for playerIdx, player := range state.players {
		if state.clients[clientID] != playerIdx {
			format := nui.Format{Fg: nui.Color(playerIdx + 61), Bg: nui.Black}
			label := &nui.Label{X: 8, Y: 5 + uint16(playerIdx), Format: format, Text: player.name}
			screen.Widgets = append(screen.Widgets, label)
		} else {
			playerIdx := playerIdx
			format := nui.Format{Fg: nui.Color(playerIdx + 61), Bg: nui.Black, Bold: true}
			entry := &nui.Entry{
				X: 8, Y: 5 + uint16(playerIdx), Format: format, Text: player.name, Max: 16,

				HandleInput: func(name string) {
					state.players[playerIdx].name = name
					updateLobbyScreens(srv, state, clientID, name)
				},
			}
			screen.Widgets = append(screen.Widgets, entry)
			screen.Focus = playerIdx
		}
	}

	headerFormat := nui.Format{Fg: nui.LightWhite, Bg: nui.Black, Underline: true}
	screen.Widgets = append(screen.Widgets, &nui.Label{X: 8, Y: 4, Format: headerFormat, Text: fmt.Sprintf("Players: %d", len(state.players))})

	if state.clients[clientID] == 0 { // host
		screen.Widgets = append(screen.Widgets, &nui.Label{
			X: 64, Y: 4, Format: headerFormat, Text: "Host",
		})
		screen.Widgets = append(screen.Widgets, &nui.Button{
			X: 62, Y: 6, Format: nui.Format{Bg: nui.Blue, Fg: nui.LightWhite}, Text: "Start",

			HandleClick: func() {
				fmt.Println("game starting")
				startGame(srv, state)
			},
		})
	}

	return screen
}

func main() {
	var state = State{
		clients: make(map[int]int),
	}

	ln, err := net.Listen("tcp", ":6567")
	if err != nil {
		panic(err)
	}
	log.Println("localhost:6567")

	srv := nui.NewServer(ln)
	srv.TermWidth = 128
	srv.TermHeight = 32 + 4
	srv.HandleConnect = func(clientID int) {
		log.Printf("event: connect [%d]\n", clientID)

		state.Lock()
		defer state.Unlock()

		if state.game == nil {
			state.clients[clientID] = len(state.players)
			state.players = append(state.players, Player{})

			screen := makeLobbyScreen(srv, &state, clientID)
			srv.SetScreen(clientID, screen)

			// create new screens for everyone
			for clientID, _ := range state.clients {
				screen := makeLobbyScreen(srv, &state, clientID)
				srv.SetScreen(clientID, screen)
			}
		} else {
			screen := &nui.Screen{
				Widgets: []nui.Widget{&nui.Label{X: 0, Y: 0, Format: nui.Format{Fg: nui.LightWhite, Bg: nui.Red}, Text: "Game has begun. Please join later."}},
			}
			srv.SetScreen(clientID, screen)

			// TODO: Print error message and close the connection
		}
	}
	srv.HandleDisconnect = func(clientID int) {
		state.Lock()
		defer state.Unlock()

		log.Printf("event: disconnect [%d]\n", clientID)

		idx, ok := state.clients[clientID]
		if !ok {
			log.Println("warning: non-existant player disconnected")
			return
		}

		delete(state.clients, clientID)
		if state.game == nil {
			if len(state.players) > 1 {
				state.players[idx] = state.players[len(state.clients)-1]
			}
			state.players = state.players[0 : len(state.players)-1]

			// create new screens for everyone
			for clientID, _ := range state.clients {
				screen := makeLobbyScreen(srv, &state, clientID)
				srv.SetScreen(clientID, screen)
			}
		} else {
			state.game.Players[idx].Disconnected = true
			state.game.Players[idx].Dead = true
		}
	}
	srv.Run()
}
