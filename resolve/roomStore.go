package resolve

// import (
// 	"encoding/json"

// 	log "github.com/cihub/seelog"
// 	"github.com/vc2402/gomes/store"
// )

// type roomStorer struct {
// 	room *Room
// }

// func (r *roomStorer) Name() string { return "room" }

// func (r *roomStorer) Describe(d store.StorableDescriptor) {
// 	d.AddField("id", store.FTString)
// 	d.AddField("name", store.FTString)
// 	d.AddField("game", store.FTString)
// 	d.AddField("state", store.FTString)
// 	d.AddField("game", store.FTString)
// 	d.AddField("round", store.FTInt)
// 	d.AddField("created", store.FTLong)
// 	d.AddField("activity", store.FTLong)
// 	d.AddField("players", store.FTByteArray)
// 	d.AddField("history", store.FTByteArray)
// }

// func (r *roomStorer) ToStore(s store.StorableBuffer) {
// 	s.SetField("id", r.room.ID)
// 	s.SetField("name", r.room.Name)
// 	s.SetField("game", r.room.Game.ID)
// 	s.SetField("state", r.room.State)
// 	s.SetField("round", r.room.Round)
// 	s.SetField("created", r.room.Created)
// 	s.SetField("activity", r.room.Activity)
// 	plArr := make([]map[string]interface{}, len(r.room.Players), len(r.room.Players))
// 	for i, p := range r.room.Players {
// 		plArr[i] = map[string]interface{}{"id": p.Player.ID, "index": p.Index, "status": p.Status}
// 	}
// 	pl, _ := json.Marshal(plArr)
// 	s.SetField("players", pl)
// 	hs, _ := json.Marshal(r.room.History)
// 	s.SetField("history", hs)
// }

// func (r *roomStorer) FromStore(s store.StorableBuffer) {
// 	r.room = &Room{
// 		ID:       s.GetField("id").(string),
// 		Name:     s.GetField("name").(string),
// 		State:    s.GetField("state").(string),
// 		Round:    s.GetField("round").(int32),
// 		Created:  s.GetField("created").(int64),
// 		Activity: s.GetField("activity").(int64),
// 	}
// 	gameID := s.GetField("game").(string)
// 	r.room.Game = getGame(gameID)
// 	r.room.impl = r.room.Game.implGenerator(r.room)
// 	plArr := make([]map[string]interface{}, 0)
// 	err := json.Unmarshal(s.GetField("players").([]byte), plArr)
// 	if err != nil {
// 		r.room.Players = make([]*RoomMember, 0)
// 	} else {
// 		r.room.Players = make([]*RoomMember, len(plArr), len(plArr))
// 		for i, p := range plArr {
// 			pl := getPlayer(p["id"].(string))
// 			if pl == nil {
// 				pl, err = newPlayer("invalid", p["id"].(string))
// 			}
// 			rm := &RoomMember{Index: int32(p["index"].(int)), Status: p["status"].(string), Player: pl}
// 			r.room.Players[i] = rm
// 		}
// 	}
// 	r.room.History = make([][]*KVPair, 0)
// 	json.Unmarshal(s.GetField("players").([]byte), r.room.History)
// 	log.Tracef("FromStore: returning room %+v", *r.room)
// }
