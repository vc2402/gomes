package resolve

import (
	"errors"
	"sync"
	"time"

	"github.com/vc2402/utils"

	log "github.com/cihub/seelog"
	graphql "github.com/graph-gophers/graphql-go"
)

type Room struct {
	id       graphql.ID
	game     *Game
	name     string
	state    string
	players  []*Player
	created  int64
	activity int64
	mux      sync.Mutex
}

var rooms map[string]*Room = make(map[string]*Room)
var roomsLock sync.RWMutex

func (r *Room) ID() graphql.ID     { return r.id }
func (r *Room) Game() *Game        { return r.game }
func (r *Room) Name() string       { return r.name }
func (r *Room) State() string      { return r.state }
func (r *Room) Players() []*Player { return r.players }

func newRoom(gameID string, name string) (*Room, error) {
	var id string
	roomsLock.Lock()
	defer roomsLock.Unlock()
	for {
		id = utils.RandString(32)
		if _, ok := rooms[id]; !ok {
			break
		}
	}
	log.Debugf("newRoom: %s", gameID)
	game := getGame(gameID)
	if game == nil {
		log.Warnf("newRoom for invalid gameID: %s", gameID)
		return nil, errors.New("invalig gameID")
	}
	room := &Room{id: graphql.ID(id), name: name, game: game, state: "NEW", players: make([]*Player, 0), created: time.Now().Unix()}
	rooms[id] = room
	log.Debugf("newRoom: returning room %s", id)
	return room, nil
}

func getRoom(id string) (*Room, error) {
	roomsLock.RLock()
	defer roomsLock.RUnlock()
	room, ok := rooms[id]
	if ok {
		log.Tracef("getRoom: returning room: %v", *room)
		room.activity = time.Now().Unix()
		return room, nil
	}
	log.Warnf("getRoom: room does not exist: %s", id)
	return nil, errors.New("room does not exist")
}

func joinRoom(roomID string, playerName string, playerID string) (*Player, error) {
	room, err := getRoom(roomID)
	if err != nil {
		return nil, err
	}
	room.mux.Lock()
	defer room.mux.Unlock()

	for _, p := range room.players {
		if p.name == playerName {
			return nil, errors.New("name duplicate")
		}
	}
	pl, err := newPlayer(playerName, playerID)
	if err != nil {
		return nil, err
	}
	room.players = append(room.players, pl)
	return pl, nil
}

func listRooms() ([]*Room, error) {
	roomsLock.RLock()
	defer roomsLock.RUnlock()
	ret := make([]*Room, 0, len(rooms))
	for _, r := range rooms {
		ret = append(ret, r)
	}
	return ret, nil
}

func deleteRoom(id string) (*Room, error) {
	room, err := getRoom(id)
	if err != nil {
		return nil, err
	}
	roomsLock.Lock()
	defer roomsLock.Unlock()
	delete(rooms, id)
	return room, nil
}
