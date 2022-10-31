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
	Corpse       [2]uint32 // undefined <-> !dead
	Disconnected bool
	Imposter     bool

	// Horizontal direction corresponds to index 0.
	// Vertical direction corresponds to index 1.
	// Direction[x] in {-1, 0, +1}.
	Direction [2]int8

	OpenTask TaskState
	Tasks    []Task
}

func (p *GamePlayer) UpdatePositionX() uint32 {
	x := int32(p.X) + int32(p.Direction[0])
	if x < 0 {
		x = 0
	}
	return uint32(x)
}
func (p *GamePlayer) UpdatePositionY() uint32 {
	y := int32(p.Y) + int32(p.Direction[1])
	if y < 0 {
		y = 0
	}
	return uint32(y)
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

func (g *Game) Update(step uint) {
	for i, player := range g.Players {
		x := g.Players[i].UpdatePositionX()
		y := g.Players[i].Y
		if step%2 == 0 {
			y = g.Players[i].UpdatePositionY()
		}
		c := g.Map.Data[y*g.Map.Width+x]
		if c == ' ' || player.Dead {
			g.Players[i].X = x
			g.Players[i].Y = y
		}
	}
}

func (g *Game) Kill(playerIdx int) {
	g.Players[playerIdx].Dead = true
	g.Players[playerIdx].Corpse = [2]uint32{
		g.Players[playerIdx].X,
		g.Players[playerIdx].Y,
	}
}
