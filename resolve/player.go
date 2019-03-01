package resolve

import (
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/vc2402/utils"
)

type Player struct {
	ID       string
	Name     string
	Created  int64
	Activity int64
}

var players = make(map[string]*Player)
var playersMux sync.RWMutex

// func (r *Player) ID() graphql.ID { return r.ID }
// func (r *Player) Name() string   { return r.name }

func (p *Player) save() {
	store := getStorage()
	if store != nil {
		create := p.Activity == 0
		p.Activity = time.Now().Unix()
		if create {
			log.Tracef("player: going to create new record in db %s", p.ID)
			store.CreateRecord(p.ID, p)
		} else {
			log.Tracef("player: going to update record in db %s", p.ID)
			store.UpdateRecord(p.ID, p)
		}

	} else {
		log.Tracef("store not found. skipping saving player")
	}
}

func newPlayer(name string, id string) (*Player, error) {
	if id == "" {
		id = utils.RandString(32)
	}
	log.Debugf("newPlayer: %s with id %s", name, id)
	p := getPlayer(id)
	if p == nil {
		playersMux.Lock()
		defer playersMux.Unlock()
		p = &Player{ID: id, Name: name, Created: time.Now().Unix()}
		players[id] = p
	}
	p.save()
	return p, nil
}

func getPlayer(id string) *Player {
	log.Tracef("getPlayer: %s", id)
	playersMux.RLock()
	defer playersMux.RUnlock()
	p, ok := players[id]
	if ok {
		p.Activity = time.Now().Unix()
	} else {
		log.Tracef("getPlayer: %s; looking in store", id)
		store := getStorage()
		if store != nil {
			p = &Player{}
			store.GetRecord(id, p)
			if p.ID == id {
				players[id] = p
				ok = true
			} else {
				p = nil
			}
		}
	}
	return p
}
