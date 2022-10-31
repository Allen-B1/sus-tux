package main

import (
	"math/rand"

	"github.com/allen-b1/sus-tux/nui"
)

type TaskState interface {
	Widgets() []nui.Widget
}

type Task interface {
	CreateState() TaskState
	Description() string
}

// State relating to a player
// during a game.
type GamePlayer struct {
	X, Y         uint32
	Dead         bool
	Disconnected bool
	Imposter     bool

	// Horizontal direction corresponds to index 0.
	// Vertical direction corresponds to index 1.
	// Direction[x] in {-1, 0, +1}.
	Direction [2]int8

	OpenTask TaskState
	Tasks    []Task
}

func (p *GamePlayer) UpdatePosition() {
	x := int32(p.X) + int32(p.Direction[0])
	y := int32(p.Y) + int32(p.Direction[1])

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	p.X = uint32(x)
	p.Y = uint32(y)
}

type Game struct {
	Map     *Map
	Players []GamePlayer
}

func NewGame(nplayers int, map_ *Map) *Game {
	imposter := rand.Intn(nplayers)

	players := make([]GamePlayer, nplayers)
	for i := range players {
		// TODO: Randomize & initialize tasks

		players[i] = GamePlayer{
			X: map_.Width / 2,
			Y: map_.Height() / 2,
		}
	}

	players[imposter].Imposter = true

	return &Game{
		Map:     map_,
		Players: players,
	}
}

func (g *Game) makeScreen(playerIdx int) *nui.Screen {
	player := &g.Players[playerIdx]
	return &nui.Screen{
		Focus: 0,
		Widgets: []nui.Widget{
			&MapWidget{
				X: 4, Y: 4, PlayerColor: nui.Color(playerIdx + 1), Map: g.Map,
				PlayerX: &player.X, PlayerY: &player.Y, Direction: &player.Direction,
				PlayerPositions: func() [][2]uint32 {
					positions := make([][2]uint32, len(g.Players))
					for playerIdx, player := range g.Players {
						positions[playerIdx] = [2]uint32{player.X, player.Y}
					}
					return positions
				},
			},
		},
	}
}

func (g *Game) Update() {
	for i := range g.Players {
		g.Players[i].UpdatePosition()
	}
}
