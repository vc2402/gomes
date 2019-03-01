package resolve

import (
	"context"

	graphql "github.com/graph-gophers/graphql-go"
)

type GameImpl interface {
	Game() *Game
	Play(context.Context, *Action) *ActionResult
	Params(context.Context) []*KVPair
	Phase(context.Context) string
	Actions(context.Context) []string
	NewMember(context.Context, *RoomMember)

	SaveState() interface{}
	LoadState(interface{}) error
}

type ImplGenerator func(*Room) GameImpl

type Game struct {
	id            graphql.ID
	name          string
	implGenerator ImplGenerator
}

func (r *Game) ID() graphql.ID { return r.id }
func (r *Game) Name() string   { return r.name }

var games []*Game = make([]*Game, 0)

func initGame() {
	// games = make([]*Game, 1)
	// games[0] = &Game{id: "profs", name: "Professions", implGenerator: g.NewProfessionsGame}
}

func AddGame(id string, name string, gen ImplGenerator) *Game {
	g := &Game{id: graphql.ID(id), name: name, implGenerator: gen}
	games = append(games, g)
	return g
}

func listGames() []*Game {
	return games
}

func getGame(id string) *Game {
	for _, g := range games {
		if string(g.id) == id {
			return g
		}
	}
	return nil
}
