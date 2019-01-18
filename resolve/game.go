package resolve

import graphql "github.com/graph-gophers/graphql-go"

type Game struct {
	id   graphql.ID
	name string
}

func (r *Game) ID() graphql.ID { return r.id }
func (r *Game) Name() string   { return r.name }

var games []*Game

func initGame() {
	games = make([]*Game, 1)
	games[0] = &Game{id: "profs", name: "Professions"}
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
