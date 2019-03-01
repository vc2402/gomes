package games

import (
	"context"
	"errors"
	"math/rand"
	"time"

	log "github.com/cihub/seelog"

	"github.com/vc2402/gomes/resolve"
)

type RPSMember struct {
	// *resolve.RoomMember
	selection string
	winner    bool
}

type RPSGame struct {
	room    *resolve.Room
	phase   string
	members []*RPSMember
}

var RPSGameObj *resolve.Game

const (
	cActionSelect = "Select"

	cDraw = "DRAW"

	cRock     = "rock"
	cPaper    = "paper"
	cScissors = "scissors"
)

func (g *RPSGame) Play(ctx context.Context, act *resolve.Action) *resolve.ActionResult {
	r := &resolve.ActionResult{ActionStatus: "invalid"}
	m, idx := g.getMember(ctx)
	if m != nil {
		log.Tracef("Play: action %s for %v", act.Name, *m)
		actions := g.getActions(m)
		found := false
		for _, a := range actions {
			if a == act.Name {
				found = true
				break
			}
		}
		if found {
			switch act.Name {
			case cActionSelect:
				if len(act.Bits) > 0 {
					sel := act.Bits[0].Value
					switch sel {
					case cRock, cPaper, cScissors:
						m.selection = sel
						r.ActionStatus = "ok"
						g.room.Players[idx].SetStatus("COMPLETE")
						g.phase = cActionSelect
						g.promote(ctx)
					default:
						r.ActionStatus = "error"
						log.Debugf("Play: invalid selection: %s", sel)
					}
				}
			}
		}

	} else {
		log.Debugf("Play: member was not found!")
		r.ActionStatus = "invalid session"
	}
	log.Tracef("Play: returning %+v", *r)
	return r
}
func (g *RPSGame) Phase(context.Context) string {
	return g.phase
}
func (g *RPSGame) Params(ctx context.Context) []*resolve.KVPair {
	log.Tracef("Params invoked")
	ret := make([]*resolve.KVPair, 0)
	m, _ := g.getMember(ctx)
	if m != nil {
		if m.selection != "" {
			ret = append(ret, &resolve.KVPair{Key: "selection", Val: m.selection})
		}
		switch g.room.State {
		case "COLLECTING":

		case "PREPARING":

		case "ACTIVE":

			if g.phase == cDraw {
				ret = append(ret, &resolve.KVPair{Key: "result", Val: cDraw})
			}
		case "FINISHED":
			res := "lose"
			if m.winner {
				res = "win"
			}
			ret = append(ret, &resolve.KVPair{Key: "result", Val: res})
		}
	}
	log.Tracef("Params: returnign: %+v", ret)
	return ret
}
func (*RPSGame) Game() *resolve.Game {
	return gameObj
}
func (g *RPSGame) Actions(ctx context.Context) []string {
	log.Tracef("Actions invoked")
	m, _ := g.getMember(ctx)
	return g.getActions(m)
}

func (g *RPSGame) NewMember(ctx context.Context, rm *resolve.RoomMember) {
	log.Tracef("NewMember: %v", *rm)
	m := &RPSMember{selection: ""}
	g.members = append(g.members, m)
	log.Tracef("NewMember: adding member with id %s %v", rm.Player.ID, *m)
	rm.SetStatus("NEW")
	g.promote(ctx)
}

func (g *RPSGame) getMember(ctx context.Context) (*RPSMember, int32) {
	m, ok := ctx.Value(resolve.CROOMMEMBER).(*resolve.RoomMember)
	log.Tracef("getMember: %v", ok)
	if !ok {
		return nil, -1
	}
	return g.members[m.Index], m.Index
}

func (g *RPSGame) getActions(m *RPSMember) []string {
	ret := make([]string, 0)
	if m != nil {
		log.Debugf("getActions: invoked on status %s for memeber: %v", g.room.State, *m)
		switch g.room.State {
		case "COLLECTING":
		case "ACTIVE":
			if m.selection == "" {
				ret = append(ret, cActionSelect)
			}
		case "FINISHED":
		}
	} else {
		log.Debugf("getActions: invoked for memeber: nil")

	}
	return ret
}

func (g *RPSGame) promote(ctx context.Context) {
	log.Tracef("going to promote from state %s", g.room.State)
	switch g.room.State {
	case "COLLECTING":
		if len(g.members) == 2 {
			log.Tracef("  promote: setting state ACTIVE")
			g.room.SetState(ctx, "ACTIVE")
			log.Tracef("  promote:room: %+v", *g.room)
		}
	case "ACTIVE":
		p0 := g.members[0].selection
		p1 := g.members[1].selection
		if p0 == "" || p1 == "" {
			return
		}
		if p0 == p1 {
			g.phase = cDraw
			for _, m := range g.members {
				m.selection = ""
			}
			g.room.NotifyOnChange(ctx)
			return
		}
		if p0 == cScissors && p1 == cRock ||
			p0 == cRock && p1 == cPaper ||
			p0 == cPaper && p1 == cScissors {
			log.Debugf("promote: setting player 1 as winner: %s vs %s", p1, p0)
			g.members[1].winner = true
		} else {
			g.members[0].winner = true
			log.Debugf("promote: setting player 0 as winner: %s vs %s", p0, p1)
		}
		g.room.SetState(ctx, "FINISHED")
	case "FINISHED":
	}

}

func (g *RPSGame) SaveState() interface{} {
	members := make([]map[string]interface{}, len(g.members), len(g.members))
	for i, mb := range g.members {
		mem := map[string]interface{}{"idx": i, "selection": mb.selection, "winner": mb.winner}
		members[i] = mem
	}
	ret := map[string]interface{}{"phase": g.phase, "members": members}
	log.Tracef("SaveState: returning: %+v", ret)
	return ret
}

func (g *RPSGame) LoadState(saved interface{}) error {
	in, ok := saved.(map[string]interface{})
	if ok {
		g.phase = in["phase"].(string)
		mems := in["members"].([]interface{})
		g.members = make([]*RPSMember, len(mems), len(mems))
		log.Tracef("LoadState: loading members; len: %d", len(g.members))
		for _, memb := range mems {
			mb := memb.(map[string]interface{})
			idx := int(mb["idx"].(float64))
			log.Tracef("LoadState: next member: %d: %+v", idx, mb)
			g.members[idx] = &RPSMember{ //RoomMember: g.room.Players[idx],
				selection: mb["selection"].(string),
				winner:    mb["winner"].(bool),
			}
		}
		log.Tracef("LoadState: exiting")
		return nil
	}
	return errors.New("Invalid game format")
}

func NewRPSGame(r *resolve.Room) resolve.GameImpl {
	return &RPSGame{room: r, members: make([]*RPSMember, 0), phase: "NEW"}
}

func InitRPSGame() {
	gameObj = resolve.AddGame("rps", "Камень Ножницы Бумага", NewRPSGame)
	// fmt.Printf("Professions: init: game object was created: %v\n", gameObj)
	rand.Seed(time.Now().UnixNano())
}
