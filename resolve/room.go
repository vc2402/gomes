package resolve

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/vc2402/gomes/store"

	"github.com/vc2402/utils"

	log "github.com/cihub/seelog"
	graphql "github.com/graph-gophers/graphql-go"
)

const (
	CROOMMEMBER = "RoomMember"
)

type Room struct {
	ID       string           `json:"id,omitempty" store:"id"`
	Game     *Game            `json:"game,omitempty" store:"helper"`
	impl     GameImpl         `store:"ignore"`
	Name     string           `json:"name,omitempty"`
	State    string           `json:"state,omitempty"`
	Players  []*RoomMember    `json:"players,omitempty"`
	Owner    *Player          `json:"owner,omitempty" store:"helper,index"`
	Created  int64            `json:"created,omitempty"`
	Activity int64            `json:"activity,omitempty"`
	Round    int32            `json:"round,omitempty"`
	History  [][]*KVPair      `json:"history,omitempty" store:"ignore"`
	mux      sync.Mutex       `store:"ignore"`
	channel  chan interface{} `store:"ignore"`
}

type RoomMember struct {
	// RoomID string  `json:"room_id,omitempty" bson:"room_id"`
	Index  int32
	Player *Player `store:"helper,index"`
	Status string
}

type RoomUpdate struct {
	event *RoomEvent
}

type RoomEvent struct {
	name  string
	value string
}

type RoomsSorter struct {
	rooms  []*Room
	sortBy string
	desc   bool
}

var rooms map[string]*Room = make(map[string]*Room)
var roomsLock sync.RWMutex

// func (r *Room) ID() graphql.ID                       { return r.id }
// func (r *Room) Game() *Game                          { return r.game }
// func (r *Room) Name() string                         { return r.name }
// func (r *Room) State() string                        { return r.state }
// func (r *Room) Round() int32                         { return r.round }
// func (r *Room) Players() []*RoomMember               { return r.players }
func (r *Room) SetState(c context.Context, s string) { r.State = s; r.NotifyOnChange(c) }
func (r *Room) Phase(c context.Context) string       { return r.impl.Phase(r.fillMember(c)) }
func (r *Room) Params(c context.Context) []*KVPair   { return r.impl.Params(r.fillMember(c)) }
func (r *Room) Actions(c context.Context) []string   { return r.impl.Actions(r.fillMember(c)) }
func (r *Room) You(ctx context.Context) *int32 {
	id := ctx.Value(cSESSION_ID).(string)
	// log.Tracef("You: looking player with id %s", id)
	if id != "" {
		for _, p := range r.Players {
			if string(p.Player.ID) == id {
				return &p.Index
			}
		}
	}
	return nil
}
func (r *Room) NotifyOnChange(ctx context.Context) {
	ch := ctx.Value("UpdateChannel")
	if ch == nil {
		log.Warnf("NotifyOnChange: UpdateChannel not found")
	} else {
		go func() {
			// msg := map[string]interface{}{"data": map[string]interface{}{"roomUpdates": "changed"}}
			msg := "changed"
			log.Tracef("NotifyOnChange: going to send message: %+v,", msg)
			ch.(chan string) <- msg
		}()
	}
}
func (r *Room) NextRound() {
	r.Round++
	r.History = append(r.History, []*KVPair{})
}
func (r *Room) AddHistory(h *KVPair) {
	log.Tracef("AddHistory: %s: %s", h.Key, h.Val)
	r.History[r.Round] = append(r.History[r.Round], h)
}
func (r *Room) getHistory(round int32) []*KVPair {
	log.Tracef("getHistory: for round: %d; current round is %d", round, r.Round)
	if r.Round < round {
		return nil
	} else {
		log.Tracef("getHistory: returning %d values (%v)", len(r.History[round]), r.History[round])
		return r.History[round]
	}
}

// store.Helper interface
func (r *Room) GetValue(attr string) (interface{}, error) {
	switch attr {
	case "Game":
		res := map[string]interface{}{}
		if r.Game != nil {
			res["gameID"] = r.Game.ID()
		}
		if r.impl != nil {
			res["gameState"] = r.impl.SaveState()
		}
		return res, nil
	case "Owner":
		res := ""
		if r.Owner != nil {
			res = r.Owner.ID
		}
		return res, nil
	default:
		return nil, errors.New("unknown attr for Room.GetValue: " + attr)
	}
	return nil, errors.New("Undefined Room.Helper field " + attr)
}

func (r *Room) SetValue(attr string, from interface{}) error {
	switch attr {
	case "Game":
		m, ok := from.(map[string]interface{})
		if ok {
			game := getGame(m["gameID"].(string))
			if game != nil {
				r.Game = game
				r.impl = game.implGenerator(r)
				return r.impl.LoadState(m["gameState"])
			} else {
				return errors.New("Invalid game id when loading: " + m["gameID"].(string))
			}
		}
		return errors.New("Invalid Room store format")
	case "Owner":
		if id, ok := from.(string); ok && id != "" {
			r.Owner = GetPlayer(id)
		}
		return nil
	default:
		return errors.New("unknown attr for Room.SetValue: " + attr)
	}
}

func (room *Room) Save(ctx context.Context) {
	store := getStorage()
	if store != nil {
		create := room.Activity == 0
		room.Activity = time.Now().Unix()
		if create {
			log.Tracef("room: going to create new record in db %s", room.ID)
			store.CreateRecord(room.ID, room)
		} else {
			log.Tracef("room: going to update record in db %s", room.ID)
			store.UpdateRecord(room.ID, room)
		}
		log.Tracef("room: record was saved successfully. Exiting (%s)", room.ID)

	} else {
		log.Tracef("store not found. skipping saving")
	}
}

// func (rm *RoomMember) Player() *Player     { return rm.player }
// func (rm *RoomMember) Status() string      { return rm.status }
func (rm *RoomMember) SetStatus(st string) { rm.Status = st }

// func (rm *RoomMember) Index() int32        { return rm.index }

// store.Helper interface
func (r *RoomMember) GetValue(attr string) (interface{}, error) {
	switch attr {
	case "Player":
		var res string
		if r.Player != nil {
			res = r.Player.ID
		}
		return res, nil
	}
	return nil, errors.New("Undefined RoomMember.Helper field " + attr)
}

func (r *RoomMember) SetValue(attr string, from interface{}) error {
	m, ok := from.(string)
	if ok {
		player := GetPlayer(m)
		if player != nil {
			r.Player = player
			return nil
		} else {
			return errors.New("Invalid player id when loading RoomMember: " + m)
		}
	}
	return errors.New("Invalid RoomMember store format")
}

func (ru *RoomUpdate) Event() *RoomEvent { return ru.event }

func (re *RoomEvent) Name() string    { return re.name }
func (re *RoomEvent) Value() string   { return re.value }
func (re *RoomEvent) ID() *graphql.ID { return (*graphql.ID)(&re.value) }

// RoomsSorter
func (rs RoomsSorter) Len() int      { return len(rs.rooms) }
func (rs RoomsSorter) Swap(i, j int) { rs.rooms[i], rs.rooms[j] = rs.rooms[j], rs.rooms[i] }
func (rs RoomsSorter) Less(i, j int) bool {
	var ret bool = false
	switch rs.sortBy {
	default:
		ret = (rs.rooms[i].Created < rs.rooms[j].Created)
	}
	if rs.desc {
		ret = !ret
	}
	return ret
}

func newRoom(ctx context.Context, gameID string, name string) (*Room, error) {
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
	room := &Room{
		ID:      id,
		Name:    name,
		Game:    game,
		State:   "COLLECTING",
		Players: make([]*RoomMember, 0),
		Owner:   GetPlayer(ctx.Value(cSESSION_ID).(string)),
		History: make([][]*KVPair, 1),
		Round:   0,
		Created: time.Now().Unix(),
		channel: make(chan interface{})}
	room.impl = game.implGenerator(room)
	room.History[0] = []*KVPair{}
	rooms[id] = room
	log.Tracef("context.store: %v", ctx.Value("store"))
	room.Save(ctx)
	log.Debugf("newRoom: returning room %s", id)
	return room, nil
}

func getRoom(ctx context.Context, id string) (*Room, error) {
	roomsLock.RLock()
	defer roomsLock.RUnlock()
	room, ok := rooms[id]
	err := errors.New("room does not exist")
	if !ok {
		// store := ctx.Value("store").(*store.DB)
		// if store != nil {
		// 	ok, err = store.GetRecord("room", id, &room)
		// 	if ok {
		// 		rooms[id] = room
		// 	}
		// }
		store := getStorage()
		if store != nil {
			room = &Room{}
			store.GetRecord(id, room)
			if room.ID == id {
				rooms[id] = room
				ok = true
			}
		}
	}
	if ok {
		log.Tracef("getRoom: returning room: %v", *room)
		room.Activity = time.Now().Unix()
		return room, nil
	}
	log.Warnf("getRoom: room does not exist: %s", id)
	return nil, err
}

func joinRoom(ctx context.Context, roomID string, playerName string) (*RoomMember, error) {
	log.Tracef("joinRoom: %s joining %s", playerName, roomID)
	room, err := getRoom(ctx, roomID)
	if err != nil {
		log.Warnf("joinRoom: room not foud: %s: %v", roomID, err)
		return nil, err
	}
	playerID := ctx.Value(cSESSION_ID).(string)
	if playerID == "" {
		return nil, errors.New("session is undefined")
	}
	log.Tracef("joinRoom: %s (%s/%s)", roomID, playerName, playerID)
	room.mux.Lock()
	defer room.mux.Unlock()

	var rm *RoomMember
	for _, p := range room.Players {
		if p.Player.Name == playerName {
			log.Infof("joinRoom: player name %s is duplicate for room %s", playerName, roomID)
			return nil, errors.New("name duplicate")
		} else if string(p.Player.ID) == playerID {
			rm = p
			log.Tracef("joinRoom: %s: player already in the list", roomID)
			break
		}
	}
	var pl *Player
	if rm == nil {
		pl = GetPlayer(playerID)
		if pl == nil {
			return nil, errors.New("not logged in")
		}
		rm = &RoomMember{Player: pl, Index: int32(len(room.Players))}
		room.impl.NewMember(ctx, rm)
		room.Players = append(room.Players, rm)
		room.Save(ctx)
	}

	room.NotifyOnChange(ctx)
	return rm, nil
}

func listRooms(ctx context.Context, all bool) ([]*Room, error) {
	ret := make([]*Room, 0, len(rooms))
	storage := getStorage()
	filter := store.Filter{Field: "Owner", Mask: ctx.Value(cSESSION_ID).(string)}
	if all && isAdmin(ctx) {
		filter = store.Filter{}
	}
	arr, err := storage.ListRecords(filter, ret)
	var ok bool
	if err == nil {
		ret, ok = arr.([]*Room)
		if !ok {
			log.Warnf("listRooms: invalid return type: %+v", arr)
		}
		sorter := RoomsSorter{ret, "Created", true}
		sort.Sort(sorter)
	}
	return ret, err
}

func deleteRoom(ctx context.Context, id string) (*Room, error) {
	log.Tracef("deleteRoom: deleting the room %s", id)
	room, err := getRoom(ctx, id)
	if err != nil {
		return nil, err
	}
	roomsLock.Lock()
	defer roomsLock.Unlock()
	delete(rooms, id)
	store := getStorage()
	return room, store.DeleteRecord("Room", id)
}

func play(ctx context.Context, id string, act *Action) (*ActionResult, error) {
	log.Tracef("play: for room %s: %+v", id, *act)
	room, err := getRoom(ctx, id)
	if err != nil {
		return nil, err
	}
	res := room.impl.Play(room.fillMember(ctx), act)
	room.Save(ctx)
	log.Tracef("play: for room %s: returning: %+v", id, *res)
	return res, nil
}

func history(ctx context.Context, id string, round *int32) []*KVPair {
	log.Tracef("history: for round %d of room %s %+v", round, id)
	room, err := getRoom(ctx, id)
	if err == nil {
		r := room.Round
		if round != nil {
			r = *round
		}
		res := room.getHistory(r)
		if res != nil {
			return res
		}
	}
	return []*KVPair{}
}

func roomUpdates(ctx context.Context, id string) (chan *RoomUpdate, error) {
	log.Tracef("roomUpdates: %s", id)
	// room, err := getRoom(id)
	// if err != nil {
	// 	return nil, err
	// }
	c := make(chan *RoomUpdate)
	log.Tracef("roomUpdates: starting the gorouting")
	go func() {
		for {
			log.Tracef("roomUpdates: gorouting: next loop")
			select {
			case <-ctx.Done():
				close(c)
				log.Tracef("roomUpdates: closing")
				return
			case <-time.After(1 * time.Second):
			}
			log.Tracef("roomUpdates: gorouting: sending test update")
			c <- &RoomUpdate{&RoomEvent{name: "test", value: "test"}}
		}
		log.Tracef("roomUpdates: gorouting: exit")
	}()
	log.Tracef("roomUpdates: exiting")
	log.Flush()
	return c, nil
}

func setUpdatesClient(ctx context.Context, id string, ch chan interface{}) {
	room, err := getRoom(ctx, id)
	if err == nil {
		log.Tracef("setUpdatesClient: setting channel for room %s", id)
		room.channel = ch
	}
	go func() {
		for {
			log.Tracef("roomUpdates: gorouting: next loop")
			select {
			case <-time.After(1 * time.Second):
			}
			log.Tracef("roomUpdates: gorouting: sending test update")
			room.NotifyOnChange(ctx)
		}
		log.Tracef("roomUpdates: gorouting: exit")
	}()
}
func (r *Room) fillMember(ctx context.Context) context.Context {
	log.Tracef("fillMemeber: looking for member")
	if s := ctx.Value(cSESSION_ID); s != nil {
		log.Tracef("fillMemeber: looking for member; session id: %+v", s)
		if p := GetPlayer(s.(string)); p != nil {
			log.Tracef("fillMemeber: looking for member; player: %+v", p)
			for _, m := range r.Players {
				log.Tracef("fillMemeber: looking for member; comparing with member: %+v", m)
				if m.Player.ID == p.ID {
					log.Tracef("fillMemeber: looking for member; returning: %+v", *m)
					return context.WithValue(ctx, CROOMMEMBER, m)
				}
			}
		} else {
			log.Debugf("fillMemeber: no player found for session id: %+v", s)
		}
	}
	return ctx
}
