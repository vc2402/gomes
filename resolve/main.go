package resolve

// "context"

// "github.com/vc2402/utils"

// log "github.com/cihub/seelog"
// "github.com/kataras/iris"
// "github.com/vc2402/gomes/schema"

// graphql "github.com/graph-gophers/graphql-go"
// "github.com/graph-gophers/graphql-go/relay"

// var (
// 	handler *relay.Handler
// )

const (
	cSESSION_ID = "session-id"
)

type Resolver struct {
	req string
}

// type RoomQuery struct {
// 	ID graphql.ID
// }

// type RoomInput struct {
// 	Name   string
// 	GameID graphql.ID
// }

type Action struct {
	Name string
	Bits []KVPairInput
}

type KVPairInput struct {
	Name  string
	Value string
}

type KVPair struct {
	Key string `json:"key,omitempty" bson:"key"`
	Val string `json:"val,omitempty" bson:"val"`
}

type ActionResult struct {
	ActionStatus string
}

// func (r *Resolver) ListGames(ctx context.Context) ([]*Game, error) {
// 	log.Tracef("ListGames: session-id: %s", ctx.Value(cSESSION_ID))
// 	return listGames(), nil
// }

// func (r *Resolver) GetRoom(ctx context.Context, args *RoomQuery) (*Room, error) {
// 	log.Tracef("GetRoom: new query")
// 	return getRoom(string(args.ID))
// }

// func (r *Resolver) DeleteRoom(ctx context.Context, args *RoomQuery) (*Room, error) {
// 	return deleteRoom(string(args.ID))
// }

// func (r *Resolver) ListRooms(ctx context.Context) ([]*Room, error) {
// 	return listRooms()
// }

// func (r *Resolver) Me(ctx context.Context) (*Player, error) {
// 	id := ctx.Value(cSESSION_ID).(string)
// 	return getPlayer(id), nil
// }

// func (r *Resolver) CreateRoom(ctx context.Context, args *struct{ Room RoomInput }) (*Room, error) {
// 	return newRoom(string(args.Room.GameID), args.Room.Name)
// }

// func (r *Resolver) JoinRoom(ctx context.Context,
// 	args *struct {
// 		RoomID graphql.ID
// 		Name   string
// 	}) (*RoomMember, error) {

// 	return joinRoom(ctx, string(args.RoomID), args.Name)
// }

// func (r *Resolver) Play(ctx context.Context,
// 	args *struct {
// 		RoomID graphql.ID
// 		Action Action
// 	}) (*ActionResult, error) {
// 	return play(ctx, string(args.RoomID), &args.Action)
// }

// func (r *Resolver) History(ctx context.Context,
// 	args *struct {
// 		RoomID graphql.ID
// 		Round  *int32
// 	}) []*KVPair {
// 	round := -1
// 	if args.Round != nil {
// 		round = int(*args.Round)
// 	}
// 	log.Tracef("History: room: %s, round: %d session-id: %s", args.RoomID, round, ctx.Value(cSESSION_ID))
// 	return history(ctx, string(args.RoomID), args.Round)
// }

// func (r *Resolver) RoomUpdates(ctx context.Context, args *struct {
// 	RoomID graphql.ID
// }) (chan *RoomUpdate, error) {
// 	return roomUpdates(ctx, string(args.RoomID))
// }

func (p *KVPair) Name() string          { return p.Key }
func (p *KVPair) Value() string         { return p.Val }
func (r *ActionResult) Status() *string { return &r.ActionStatus }

// func Handler(cont context.Context) iris.Handler {
// 	return func(ctx iris.Context) {
// 		log.Debugf("Handler: new request; %v", ctx.RemoteAddr)
// 		id := ctx.GetCookie(cSESSION_ID)
// 		if id != "" {
// 			log.Debugf("Handler: new request; session-id: %s", id)
// 		} else {
// 			id = utils.RandString(32)
// 			ctx.SetCookieKV(cSESSION_ID, id)
// 			log.Infof("Handler: generating new session-id: %s", id)
// 		}
// 		cont = context.WithValue(cont, cSESSION_ID, id)
// 		req := ctx.Request().WithContext(cont)
// 		handler.ServeHTTP(ctx.ResponseWriter(), req)
// 	}
// }

// func Init() {
// 	initGame()
// 	s, err := schema.GetSchema()
// 	if err != nil {
// 		panic(err)
// 	}
// 	schema := graphql.MustParseSchema(s, &Resolver{}, graphql.UseStringDescriptions())
// 	handler = &relay.Handler{Schema: schema}
// }
