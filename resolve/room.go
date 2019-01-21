package resolve

import (
	"context"
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
	impl     GameImpl
	name     string
	state    string
	players  []*RoomMember
	created  int64
	activity int64
	mux      sync.Mutex
}

type RoomMember struct {
	player *Player
	status string
}

var rooms map[string]*Room = make(map[string]*Room)
var roomsLock sync.RWMutex

func (r *Room) ID() graphql.ID                     { return r.id }
func (r *Room) Game() *Game                        { return r.game }
func (r *Room) Name() string                       { return r.name }
func (r *Room) State() string                      { return r.state }
func (r *Room) SetState(s string)                  { r.state = s; r.NotifyOnChange() }
func (r *Room) Phase(c context.Context) string     { return r.impl.Phase(c) }
func (r *Room) Players() []*RoomMember             { return r.players }
func (r *Room) Params(c context.Context) []*KVPair { return r.impl.Params(c) }
func (r *Room) Actions(c context.Context) []string { return r.impl.Actions(c) }
func (r *Room) You(ctx context.Context) *graphql.ID {
	id := ctx.Value(cSESSION_ID).(string)
	if id != "" {
		for _, p := range r.players {
			if string(p.player.id) == id {
				return &p.player.id
			}
		}
	}
	return nil
}
func (r *Room) NotifyOnChange() {

}

func (rm *RoomMember) Player() *Player     { return rm.player }
func (rm *RoomMember) Status() string      { return rm.status }
func (rm *RoomMember) SetStatus(st string) { rm.status = st }

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
	room := &Room{id: graphql.ID(id), name: name, game: game, state: "COLLECTING", players: make([]*RoomMember, 0), created: time.Now().Unix()}
	room.impl = game.implGenerator(room)
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

func joinRoom(roomID string, playerName string, playerID string) (*RoomMember, error) {
	room, err := getRoom(roomID)
	if err != nil {
		return nil, err
	}
	room.mux.Lock()
	defer room.mux.Unlock()

	var rm *RoomMember
	for _, p := range room.players {
		if p.player.name == playerName {
			return nil, errors.New("name duplicate")
		} else if string(p.player.id) == playerID {
			rm = p
			break
		}
	}
	var pl *Player
	if rm == nil {
		pl = getPlayer(playerID)
		if pl == nil {
			pl, err = newPlayer(playerName, playerID)
			if err != nil {
				return nil, err
			}
			rm = &RoomMember{player: pl}
			room.impl.NewMember(rm)
			room.players = append(room.players, rm)
		}
	}

	return rm, nil
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

func play(ctx context.Context, id string, act *Action) (*ActionResult, error) {
	room, err := getRoom(id)
	if err != nil {
		return nil, err
	}
	return room.impl.Play(ctx, act), nil
}
