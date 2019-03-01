package games

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"sync"
	"time"

	log "github.com/cihub/seelog"

	"github.com/vc2402/gomes/resolve"
)

type Member struct {
	// *resolve.RoomMember
	idx        int32
	profession string
	isSpy      bool
	next       int32
	vote       int
	active     bool
}

type question struct {
	question string
	response string
	asking   int32
	replying int32
}

type ProfessionsGame struct {
	room              *resolve.Room
	phase             string
	profession        string
	sentAway          int32
	members           []*Member
	votesCount        int
	membersCount      int
	guessedProfession string
	*question
}

var gameObj *resolve.Game
var profMux sync.Mutex

const (
	cPERSONID = "session-id"

	cActionSetProfession = "SetProfession"
	cActionStartGame     = "StartGame"
	cActionAsk           = "Ask"
	cActionAnswer        = "Answer"
	cActionVote          = "Vote"
	cActionGuess         = "Guess"
	cSolvedResult        = "FIGURED-OUT"
	cGuessedResult       = "GUESSED"
	cStateCollecting     = "COLLECTING"
	cStateActive         = "ACTIVE"
	cStateFinished       = "FINISHED"
	cStatePreparing      = "PREPARING"
	cStateComplete       = "COMPLETE"
	cPhaseProfessions    = "PROFESSIONS"
	cPhaseVoting         = "VOTING"
	cStateWaiting        = "WAITING"
	cStateReplying       = "REPLYING"
	cStateAsking         = "ASKING"
)

func (g *ProfessionsGame) Play(ctx context.Context, act *resolve.Action) *resolve.ActionResult {
	profMux.Lock()
	defer profMux.Unlock()
	r := &resolve.ActionResult{ActionStatus: "invalid"}
	m := g.getMember(ctx)
	log.Tracef("Play: action %s for %v", act.Name, *m)
	if m != nil {
		actions := g.getActions(ctx, m)
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
				for idx := range g.members {
					g.room.Players[idx].SetStatus(cStatePreparing)
				}
				g.phase = cPhaseProfessions
				g.room.SetState(ctx, cStatePreparing)
				r.ActionStatus = "ok"
			case cActionSetProfession:
				if len(act.Bits) > 0 {
					prof := act.Bits[0].Value
					m.profession = prof
					g.room.Players[m.idx].SetStatus(cStateComplete)
					r.ActionStatus = "ok"
					g.promote(ctx)
				}
			case cActionAsk:
				if len(act.Bits) > 0 {
					quest := act.Bits[0].Value
					q := &question{asking: m.idx, question: quest, replying: m.next}
					g.question = q
					g.room.Players[m.idx].SetStatus(cStateWaiting)
					g.room.Players[m.next].SetStatus(cStateReplying)
					g.phase = cStateReplying
					r.ActionStatus = "ok"
					g.room.AddHistory(&resolve.KVPair{
						Key: g.room.Players[m.idx].Player.Name + " > " + g.room.Players[m.next].Player.Name,
						Val: quest})
					g.room.NotifyOnChange(ctx)
				} else {
					r.ActionStatus = "invalid"
				}
			case cActionAnswer:
				if len(act.Bits) > 0 {
					// TODO: check round is finished
					answ := act.Bits[0].Value
					q := g.question
					q.response = answ
					g.room.Players[m.idx].SetStatus(cStateAsking)
					g.phase = cStateAsking
					r.ActionStatus = "ok"

					g.room.AddHistory(&resolve.KVPair{
						Key: g.room.Players[q.asking].Player.Name + " < " + g.room.Players[m.idx].Player.Name,
						Val: answ})
					g.promote(ctx)
					g.room.NotifyOnChange(ctx)
				} else {
					r.ActionStatus = "invalid"
				}
			case cActionVote:
				if len(act.Bits) > 0 {
					if idx, err := strconv.Atoi(act.Bits[0].Value); err != nil || idx >= len(g.members) || idx < 0 || !g.members[idx].active {
						r.ActionStatus = "incorrect"
						log.Warnf("Vote: invalid index: %s", act.Bits[0].Value)
					} else {
						m.vote = idx
						g.votesCount++
						g.promote(ctx)
					}
				}
			case cActionGuess:
				if len(act.Bits) > 0 {
					guess := act.Bits[0].Value
					g.guessedProfession = guess
					if guess == g.profession {
						g.room.SetState(ctx, cStateFinished)
						g.phase = cGuessedResult
						g.room.NotifyOnChange(ctx)
					}
				}
			}
		} else {
			r.ActionStatus = "invalid"
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
		switch g.room.State {
		case cStateCollecting:

		case cStatePreparing:

		case cStateActive:
			if !m.isSpy {
				ret = append(ret, &resolve.KVPair{Key: "profession", Val: g.profession})
			} else {
				ret = append(ret, &resolve.KVPair{Key: "isSpy", Val: "true"})
			}
			if g.room.Phase(ctx) != cPhaseVoting && g.sentAway != -1 {
				ret = append(ret, &resolve.KVPair{Key: "exile", Val: g.room.Players[g.sentAway].Player.Name})
			}
			if g.guessedProfession != "" {
				ret = append(ret, &resolve.KVPair{Key: "guess", Val: g.guessedProfession})
			}
		case cStateFinished:
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
	return g.getActions(ctx, m)
}

func (g *ProfessionsGame) NewMember(ctx context.Context, rm *resolve.RoomMember) {
	profMux.Lock()
	defer profMux.Unlock()
	log.Tracef("NewMember: %v", *rm)
	m := &Member{profession: "", isSpy: false, idx: rm.Index}
	g.members = append(g.members, m)
	m.next = 0
	count := len(g.room.Players)
	if count > 0 {
		g.members[count-1].next = int32(count)
	}
	log.Tracef("NewMember: adding member with id %s %v", rm.Player.ID, *m)
	rm.SetStatus("NEW")
}

func (g *ProfessionsGame) getMember(ctx context.Context) *Member {
	rm, ok := ctx.Value(resolve.CROOMMEMBER).(*resolve.RoomMember)
	// log.Tracef("getMember: %v (%v)", rm, ok)
	if !ok {
		return nil
	}
	return g.members[rm.Index]
}

func (g *ProfessionsGame) getActions(ctx context.Context, m *Member) []string {
	ret := make([]string, 0)
	if m != nil {
		log.Debugf("getActions: invoked on status %s for memeber: %v", g.room.State, *m)
		switch g.room.State {
		case cStateCollecting:
			log.Debugf("getActions: %v", g.members)
			if len(g.members) >= 2 {
				ret = append(ret, cActionStartGame)
			}
		case cStatePreparing:
			if m.profession == "" {
				ret = append(ret, cActionSetProfession)
			}
		case cStateActive:
			if m.active {
				if g.room.Players[m.idx].Status == cStateAsking {
					ret = append(ret, cActionAsk)
				} else if g.room.Players[m.idx].Status == cStateReplying {
					ret = append(ret, cActionAnswer)
				} else if g.room.Phase(ctx) == cPhaseVoting && m.vote == -1 {
					ret = append(ret, cActionVote)
				}
				if m.isSpy {
					ret = append(ret, cActionGuess)
				}
			}
		case cStateFinished:
		}
	} else {
		log.Debugf("getActions: invoked for memeber: nil")

	}
	return ret
}

func (g *ProfessionsGame) promote(ctx context.Context) {
	log.Tracef("going to promote from state %s", g.room.State)
	switch g.room.State {
	case cStatePreparing:
		for _, m := range g.members {
			if m.profession == "" {
				return
			}
		}
		limit := int32(len(g.room.Players))
		idx := rand.Int31n(limit)
		g.profession = g.members[idx].profession
		log.Tracef("promote PREPARING: selected profession is %s[%d]", g.profession, idx)
		spy := idx
		for spy == idx {
			log.Tracef("promote PREPARING: trying to generate index for spy up to %d (%d == %d)", limit, idx, spy)
			spy = rand.Int31n(limit)
		}
		log.Tracef("promote PREPARING: selected spy is %d", spy)
		g.members[spy].isSpy = true
		g.phase = cStateAsking
		g.votesCount = 0
		g.membersCount = len(g.members)
		for i, m := range g.members {
			if i == 0 {
				g.room.Players[idx].SetStatus(cStateAsking)
			} else {
				g.room.Players[idx].SetStatus(cStateWaiting)
			}
			m.active = true
		}
		// g,members[0].RoomMember.SetStatus()
		g.room.SetState(ctx, cStateActive)
	case cStateActive:
		log.Debugf("promote: ACTIVE; phase: %s question is: %+v", g.phase, *g.question)
		if g.phase == cStateAsking && g.question.replying < g.question.asking {
			g.phase = cPhaseVoting
			for idx, m := range g.members {
				if m.active {
					g.room.Players[idx].SetStatus(cPhaseVoting)
					m.vote = -1
				}
			}
			g.votesCount = 0
			g.room.NotifyOnChange(ctx)
		} else if g.phase == cPhaseVoting && g.votesCount == g.membersCount {
			votes := make([]int, len(g.members))
			max := 0
			maxIdx := -1
			for i, p := range g.members {
				if p.active {
					g.addVoteToHistory(p, p.vote)
					votes[p.vote]++
					if votes[p.vote] > max {
						max = votes[p.vote]
						maxIdx = i
					}
				}
			}
			g.members[maxIdx].active = false
			g.membersCount--
			g.sentAway = int32(maxIdx)
			if g.members[maxIdx].isSpy {
				g.room.SetState(ctx, cStateFinished)
				g.phase = cSolvedResult
			} else if g.membersCount < 3 {
				g.room.SetState(ctx, cStateFinished)
				g.phase = cGuessedResult
			} else {
				g.room.AddHistory(&resolve.KVPair{Key: "!", Val: g.room.Players[maxIdx].Player.Name})
				g.room.NextRound()
				g.phase = cStateAsking
				g.votesCount = 0
				prev := -1
				first := -1
				for i, p := range g.members {
					if p.active {
						if first == -1 {
							first = i
							g.room.Players[i].SetStatus(cStateAsking)
						} else {
							g.room.Players[i].SetStatus(cStateWaiting)
							p.next = int32(prev)
						}
						prev = i
					}
				}
				if prev != -1 {
					g.members[prev].next = int32(first)
					log.Debugf("promote: members loop was renewed")
				} else {
					log.Errorf("promote: can not loop members!")
				}
				g.members[maxIdx].next = -1
			}
			g.room.NotifyOnChange(ctx)
		}
	case cStateFinished:
	}

}

func (g *ProfessionsGame) addVoteToHistory(m *Member, idx int) {
	g.room.AddHistory(&resolve.KVPair{Key: g.room.Players[m.idx].Player.Name + " #", Val: g.room.Players[idx].Player.Name})
}

func (g *ProfessionsGame) SaveState() interface{} {
	members := make([]map[string]interface{}, len(g.members), len(g.members))
	for i, mb := range g.members {
		mem := map[string]interface{}{
			"idx": mb.idx, "profession": mb.profession, "isSpy": mb.isSpy,
			"next": mb.next, "active": mb.active, "vote": mb.vote,
		}
		members[i] = mem
	}
	ret := map[string]interface{}{
		"phase": g.phase, "members": members, "profession": g.profession,
		"membersCount": g.membersCount, "sentAway": g.sentAway,
		"guessedProfession": g.guessedProfession, "votesCount": g.votesCount,
	}
	if g.question != nil {
		ret["asking"] = g.asking
		ret["replying"] = g.replying
		ret["question"] = g.question
		ret["response"] = g.response
	}

	log.Tracef("SaveState: returning: %+v", ret)
	return ret
}

func (g *ProfessionsGame) LoadState(saved interface{}) error {
	in, ok := saved.(map[string]interface{})
	if ok {
		g.phase = in["phase"].(string)
		g.profession = in["profession"].(string)
		g.membersCount = int(in["membersCount"].(float64))
		g.sentAway = int32(in["sentAway"].(float64))
		g.guessedProfession = in["guessedProfession"].(string)
		g.votesCount = int(in["votesCount"].(float64))
		if in["question"] != nil {
			g.question = &question{
				asking:   int32(in["asking"].(float64)),
				replying: int32(in["replying"].(float64)),
				question: in["question"].(string),
				response: in["response"].(string),
			}
		}

		mems := in["members"].([]interface{})
		g.members = make([]*Member, len(mems), len(mems))
		log.Tracef("LoadState: loading members; len: %d", len(g.members))
		for _, memb := range mems {
			mb := memb.(map[string]interface{})
			idx := int(mb["idx"].(float64))
			log.Tracef("LoadState: next member: %d: %+v", idx, mb)
			g.members[idx] = &Member{
				idx:        int32(mb["idx"].(float64)),
				profession: mb["profession"].(string),
				active:     mb["active"].(bool),
				next:       int32(mb["next"].(float64)),
				isSpy:      mb["isSpy"].(bool),
				vote:       int(mb["vote"].(float64)),
			}
		}
		log.Tracef("LoadState: exiting")
		return nil
	}
	return errors.New("Invalid game format")
}

func NewProfessionsGame(r *resolve.Room) resolve.GameImpl {
	return &ProfessionsGame{room: r, members: make([]*Member, 0), phase: "NEW", sentAway: -1}
}

func InitProfessions() {
	gameObj = resolve.AddGame("profs", "Professions", NewProfessionsGame)
	// fmt.Printf("Professions: init: game object was created: %v\n", gameObj)
	rand.Seed(time.Now().UnixNano())
}
