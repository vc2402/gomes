package resolve

import (
	"sync"
	"time"

	log "github.com/cihub/seelog"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/vc2402/utils"
)

type Player struct {
	id       graphql.ID
	name     string
	created  int64
	activity int64
}

var players = make(map[string]*Player)
var playersMux sync.RWMutex

func (r *Player) ID() graphql.ID { return r.id }
func (r *Player) Name() string   { return r.name }

func newPlayer(name string, id string) (*Player, error) {
	if id == "" {
		id = utils.RandString(32)
	}
	log.Debugf("newPlayer: %s with id %s", name, id)
	p := getPlayer(id)
	if p == nil {
		playersMux.Lock()
		defer playersMux.Unlock()
		p = &Player{id: graphql.ID(id), name: name, created: time.Now().Unix()}
		players[id] = p
	}

	return p, nil
}

func getPlayer(id string) *Player {
	playersMux.RLock()
	defer playersMux.RUnlock()
	p, ok := players[id]
	if ok {
		p.activity = time.Now().Unix()
	}
	return p
}
