package resolve

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/vc2402/utils"
)

type Player struct {
	ID       string   `json:"id,omitempty"`
	Name     string   `json:"name,omitempty"`
	Login    string   `json:"login,omitempty"`
	Password string   `json:"-"`
	Created  int64    `json:"created,omitempty"`
	Activity int64    `json:"activity,omitempty"`
	Roles    []string `json:"roles,omitempty"`
}

var players = make(map[string]*Player)
var playersMux sync.RWMutex

// func (r *Player) ID() graphql.ID { return r.ID }
// func (r *Player) Name() string   { return r.name }

func FindPlayer(login string) *Player {
	users := make([]*Player, 0)
	store := getStorage()
	arr, err := store.ListRecords(users)
	var ok bool
	if err == nil {
		users, ok = arr.([]*Player)
		if ok {
			for _, usr := range users {
				if usr.Login == login {
					return usr
				}
			}
		}
	}
	return nil
}

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

func newPlayer(name string, id string, login string) (*Player, error) {
	if id == "" {
		id = utils.RandString(32)
	}
	log.Debugf("newPlayer: %s with id %s", name, id)
	p := GetPlayer(id)
	if p == nil {
		playersMux.Lock()
		defer playersMux.Unlock()
		p = &Player{ID: id, Name: name, Created: time.Now().Unix(), Login: login, Password: "qq38ww12", Roles: []string{}}
		players[id] = p
	}
	p.save()
	return p, nil
}

func GetPlayer(id string) *Player {
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

func deletePlayer(id string) (*Player, error) {
	log.Tracef("deletePlayer: deleting the player %s", id)
	pl := GetPlayer(id)
	if pl == nil {
		return nil, errors.New("player not found")
	}
	playersMux.Lock()
	defer playersMux.Unlock()
	delete(players, id)
	store := getStorage()
	return pl, store.DeleteRecord("Player", id)
}

type PlayersSorter struct {
	players []*Player
	sortBy  string
	desc    bool
}

// RoomsSorter
func (rs PlayersSorter) Len() int      { return len(rs.players) }
func (rs PlayersSorter) Swap(i, j int) { rs.players[i], rs.players[j] = rs.players[j], rs.players[i] }
func (rs PlayersSorter) Less(i, j int) bool {
	var ret bool = false
	switch rs.sortBy {
	case "Name":
		ret = strings.Compare(rs.players[i].Name, rs.players[j].Name) == -1
	case "Login":
		ret = strings.Compare(rs.players[i].Login, rs.players[j].Login) == -1
	default:
		ret = (rs.players[i].Created < rs.players[j].Created)
	}
	if rs.desc {
		ret = !ret
	}
	return ret
}

func listPlayers() ([]*Player, error) {
	ret := make([]*Player, 0)
	store := getStorage()
	arr, err := store.ListRecords(ret)
	var ok bool
	if err == nil {
		ret, ok = arr.([]*Player)
		if !ok {
			log.Warnf("listPlayers: invalid return type: %+v", arr)
		}
		sorter := PlayersSorter{ret, "Name", true}
		sort.Sort(sorter)
	}
	return ret, err
}
