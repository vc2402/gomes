package games

import (
	"context"
	"math/rand"
	"time"

	log "github.com/cihub/seelog"

	"github.com/vc2402/gomes/resolve"
)

type Member struct {
	*resolve.RoomMember
	profession string
	isSpy      bool
}

type ProfessionsGame struct {
	room       *resolve.Room
	phase      string
	profession string
	members    map[string]*Member
}

var gameObj *resolve.Game

const (
	cPERSONID = "session-id"

	cActionSetProfession = "SetProfession"
	cActionStartGame     = "StartGame"
)

func (g *ProfessionsGame) Play(ctx context.Context, act *resolve.Action) *resolve.ActionResult {
	r := &resolve.ActionResult{ActionStatus: "invalid"}
	m := g.getMember(ctx)
	log.Tracef("Play: action %s for %v", act.Name, *m)
	if m != nil {
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
			case cActionStartGame:
				log.Debugf("Play: setting state PREPARING")
				for _, m := range g.members {
					m.RoomMember.SetStatus("PREPARING")
				}
				g.phase = "PROFESSIONS"
				g.room.SetState("PREPARING")
				r.ActionStatus = "ok"
			case cActionSetProfession:
				if len(act.Bits) > 0 {
					prof := act.Bits[0].Value
					m.profession = prof
					m.RoomMember.SetStatus("COMPLETE")
					r.ActionStatus = "ok"
					g.promote()
				}

			}
		}

	}

	return r
}
func (g *ProfessionsGame) Phase(context.Context) string {
	return g.phase
}
func (g *ProfessionsGame) Params(ctx context.Context) []*resolve.KVPair {
	log.Tracef("Params invoked")
	ret := make([]*resolve.KVPair, 0)
	m := g.getMember(ctx)
	if m != nil {
		switch g.room.State() {
		case "COLLECTING":

		case "PREPARING":

		case "ACTIVE":
			if !m.isSpy {
				ret = append(ret, &resolve.KVPair{Key: "profession", Val: g.profession})
			} else {
				ret = append(ret, &resolve.KVPair{Key: "isSpy", Val: "true"})
			}
		case "FINISHED":
		}
	}
	return ret
}
func (*ProfessionsGame) Game() *resolve.Game {
	return gameObj
}
func (g *ProfessionsGame) Actions(ctx context.Context) []string {
	log.Tracef("Actions invoked")
	m := g.getMember(ctx)
	return g.getActions(m)
}

func (g *ProfessionsGame) NewMember(rm *resolve.RoomMember) {
	log.Tracef("NewMember: %v", *rm)
	m := &Member{RoomMember: rm, profession: "", isSpy: false}
	g.members[string(rm.Player().ID())] = m
	log.Tracef("NewMember: adding member with id %s %v", rm.Player().ID(), *m)
	rm.SetStatus("NEW")
}

func (g *ProfessionsGame) getMember(ctx context.Context) *Member {
	id, ok := ctx.Value(cPERSONID).(string)
	log.Tracef("getMember: %s (%v)", id, ok)
	if !ok || id == "" {
		return nil
	}
	return g.members[id]
}

func (g *ProfessionsGame) getActions(m *Member) []string {
	ret := make([]string, 0)
	if m != nil {
		log.Debugf("getActions: invoked on status %s for memeber: %v", g.room.State(), *m)
		switch g.room.State() {
		case "COLLECTING":
			log.Debugf("getActions: %v", g.members)
			if len(g.members) >= 2 {
				ret = append(ret, cActionStartGame)
			}
		case "PREPARING":
			if m.profession == "" {
				ret = append(ret, cActionSetProfession)
			}
		case "ACTIVE":
		case "FINISHED":
		}
	} else {
		log.Debugf("getActions: invoked for memeber: nil")

	}
	return ret
}

func (g *ProfessionsGame) promote() {
	log.Tracef("going to promote from state %s", g.room.State())
	switch g.room.State() {
	case "PREPARING":
		for _, m := range g.members {
			if m.profession == "" {
				return
			}
		}
		limit := int32(len(g.room.Players()))
		idx := rand.Int31n(limit)
		g.profession = g.members[string(g.room.Players()[idx].Player().ID())].profession
		log.Tracef("promote PREPARING: selected profession is %s[%d]", g.profession, idx)
		spy := idx
		for spy == idx {
			log.Tracef("promote PREPARING: trying to generate index for spy up to %d (%d == %d)", limit, idx, spy)
			spy = rand.Int31n(limit)
		}
		log.Tracef("promote PREPARING: selected spy is %d", spy)
		g.members[string(g.room.Players()[spy].Player().ID())].isSpy = true
		g.phase = "ASKING"
		for _, m := range g.members {
			m.RoomMember.SetStatus("ASKING")
		}
		g.room.SetState("ACTIVE")
	case "ACTIVE":
	case "FINISHED":
	}

}

func NewProfessionsGame(r *resolve.Room) resolve.GameImpl {
	return &ProfessionsGame{room: r, members: make(map[string]*Member, 0), phase: "NEW"}
}

func InitProfessions() {
	gameObj = resolve.AddGame("profs", "Professions", NewProfessionsGame)
	// fmt.Printf("Professions: init: game object was created: %v\n", gameObj)
	rand.Seed(time.Now().UnixNano())
}
