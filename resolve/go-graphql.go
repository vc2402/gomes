package resolve

import (
	"context"
	"net/http"

	"github.com/vc2402/gomes/store"

	log "github.com/cihub/seelog"
	"github.com/functionalfoundry/graphqlws"
	"github.com/kataras/iris"

	"github.com/graphql-go/graphql"
)

type GomesScheme struct {
	schema              *graphql.Schema
	subscriptionManager graphqlws.SubscriptionManager
	WSHandler           http.Handler
	storage             *store.Store
	updateChannel       chan string
}

type Client struct {
	name string
	send chan interface{}
}

var Schema *GomesScheme

var TokenHMACSecret = []byte("hJlasdf;jk60sadf96GgasfghfgHGfyfgOoSDflkjh^asdf87Gkhgasdfl")

func getStorage() *store.Store {
	return Schema.storage
}

func InitGraphQL(stor *store.Store) *GomesScheme {
	// Schema
	var gameType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Game",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.ID,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						game := p.Source.(*Game)
						return game.id, nil
					},
				},
				"name": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						game := p.Source.(*Game)
						return game.name, nil
					},
				},
			},
		},
	)
	var playerType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Player",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.ID,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						pl := p.Source.(*Player)
						return pl.ID, nil
					},
				},
				"name": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						pl := p.Source.(*Player)
						return pl.Name, nil
					},
				},
			},
		},
	)
	var roomMemberType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "RoomMember",
			Fields: graphql.Fields{
				"player": &graphql.Field{
					Type: playerType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						rm := p.Source.(*RoomMember)
						return rm.Player, nil
					},
				},
				"status": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						rm := p.Source.(*RoomMember)
						return rm.Status, nil
					},
				},
				"index": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						rm := p.Source.(*RoomMember)
						return rm.Index, nil
					},
				},
			},
		},
	)

	// var gameStatetType = graphql.NewEnum(
	// 	graphql.EnumConfig{
	// 		Name: "GameState",
	// 		Values: graphql.EnumValueConfigMap{
	// 			"COLLECTING": &graphql.EnumValueConfig{
	// 				Value: "COLLECTING",
	// 			},
	// 			"PREPARING": &graphql.EnumValueConfig{
	// 				Value: "PREPARING",
	// 			},
	// 			"ACTIVE": &graphql.EnumValueConfig{
	// 				Value: "ACTIVE",
	// 			},
	// 			"FINISHED": &graphql.EnumValueConfig{
	// 				Value: "FINISHED",
	// 			},
	// 		},
	// 	},
	// )

	var kvPairType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "KVPair",
			Fields: graphql.Fields{
				"name": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						pair := p.Source.(*KVPair)
						return pair.Name(), nil
					},
				},
				"value": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						pair := p.Source.(*KVPair)
						return pair.Value(), nil
					},
				},
			},
		},
	)

	var actionResultType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "ActionResult",
			Fields: graphql.Fields{
				"status": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						ar := p.Source.(*ActionResult)
						return ar.ActionStatus, nil
					},
				},
			},
		},
	)

	var loginResultType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "LoginResult",
			Fields: graphql.Fields{
				"token": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						ar := p.Source.(string)
						return ar, nil
					},
				},
			},
		},
	)

	var roomInputType = graphql.NewInputObject(
		graphql.InputObjectConfig{
			Name: "RoomInput",
			Fields: graphql.InputObjectConfigFieldMap{
				"name": &graphql.InputObjectFieldConfig{
					Type: graphql.String,
				},
				"gameID": &graphql.InputObjectFieldConfig{
					Type: graphql.ID,
				},
			},
		},
	)

	var loginInputType = graphql.NewInputObject(
		graphql.InputObjectConfig{
			Name: "LoginInput",
			Fields: graphql.InputObjectConfigFieldMap{
				"name": &graphql.InputObjectFieldConfig{
					Type: graphql.String,
				},
				"password": &graphql.InputObjectFieldConfig{
					Type: graphql.String,
				},
			},
		},
	)

	var kvPairInputType = graphql.NewInputObject(
		graphql.InputObjectConfig{
			Name: "KVPairInput",
			Fields: graphql.InputObjectConfigFieldMap{
				"name": &graphql.InputObjectFieldConfig{
					Type: graphql.String,
				},
				"value": &graphql.InputObjectFieldConfig{
					Type: graphql.String,
				},
			},
		},
	)

	var actionInputType = graphql.NewInputObject(
		graphql.InputObjectConfig{
			Name: "Action",
			Fields: graphql.InputObjectConfigFieldMap{
				"name": &graphql.InputObjectFieldConfig{
					Type: graphql.String,
				},
				"bits": &graphql.InputObjectFieldConfig{
					Type: graphql.NewList(kvPairInputType),
				},
			},
		},
	)

	var roomType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Room",
			Fields: graphql.Fields{
				"id": &graphql.Field{
					Type: graphql.ID,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.ID, nil
					},
				},
				"name": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.Name, nil
					},
				},
				"game": &graphql.Field{
					Type: gameType,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.Game, nil
					},
				},
				"you": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.You(p.Context), nil
					},
				},
				"round": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.Round, nil
					},
				},
				"state": &graphql.Field{
					// Type: gameStatetType,
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.State, nil
					},
				},
				"phase": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.Phase(p.Context), nil
					},
				},
				"actions": &graphql.Field{
					Type: graphql.NewList(graphql.String),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.Actions(p.Context), nil
					},
				},
				"players": &graphql.Field{
					Type: graphql.NewList(roomMemberType),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.Players, nil
					},
				},
				"params": &graphql.Field{
					Type: graphql.NewList(kvPairType),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						room := p.Source.(*Room)
						return room.Params(p.Context), nil
					},
				},
			},
		},
	)

	var roomUpdateInfoType = graphql.NewObject(
		graphql.ObjectConfig{
			Name: "RoomUpdateInfo",
			Fields: graphql.Fields{
				"name": &graphql.Field{
					Type: graphql.String,
				},
				"value": &graphql.Field{
					Type: graphql.String,
				},
			},
		},
	)

	rootQuery := graphql.ObjectConfig{Name: "Query",
		Fields: graphql.Fields{
			"me": &graphql.Field{
				Type:        playerType,
				Description: "me request",
			},
			"listRooms": &graphql.Field{
				Type:        graphql.NewList(roomType),
				Description: "List all the rooms",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// log.Tracef("resolve: %+v", p)
					return listRooms()
				},
			},
			"getRoom": &graphql.Field{
				Type:        roomType,
				Description: "get the room",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.ID,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					log.Tracef("resolve: %+v", p)
					id := p.Args["id"].(string)
					return getRoom(p.Context, id)
				},
			},
			"listGames": &graphql.Field{
				Type:        graphql.NewList(gameType),
				Description: "List all the games",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// log.Tracef("resolve: %+v", p)
					return listGames(), nil
				},
			},

			"history": &graphql.Field{
				Type:        graphql.NewList(kvPairType),
				Description: "get the room",
				Args: graphql.FieldConfigArgument{
					"roomID": &graphql.ArgumentConfig{
						Type: graphql.ID,
					},
					"round": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: -1,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// log.Tracef("resolve: %+v", p)
					id := p.Args["roomID"].(string)
					round := int32(p.Args["round"].(int))
					return history(p.Context, id, &round), nil
				},
			},
		},
	}

	mutation := graphql.ObjectConfig{Name: "Mutation",
		Fields: graphql.Fields{
			"login": &graphql.Field{
				Type:        loginResultType,
				Description: "login ",
				Args: graphql.FieldConfigArgument{
					"credentials": &graphql.ArgumentConfig{
						Type: loginInputType,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					credentials := p.Args["credentials"].(map[string]interface{})
					name := credentials["name"].(string)
					pswd := credentials["password"].(string)
					log.Tracef("login: got: %s: %s", name, pswd)
					return Login(name, pswd)
				},
			},
			"createRoom": &graphql.Field{
				Type:        roomType,
				Description: "create room ",
				Args: graphql.FieldConfigArgument{
					"room": &graphql.ArgumentConfig{
						Type: roomInputType,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					room := p.Args["room"].(map[string]interface{})
					id := room["gameID"].(string)
					name := room["name"].(string)
					log.Tracef("createRoom: got: %s: %s", name, id)
					return newRoom(p.Context, id, name)

				},
			},
			"joinRoom": &graphql.Field{
				Type:        roomMemberType,
				Description: "join the room ",
				Args: graphql.FieldConfigArgument{
					"roomID": &graphql.ArgumentConfig{
						Type: graphql.ID,
					},
					"name": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					roomID := p.Args["roomID"].(string)
					name := p.Args["name"].(string)
					log.Tracef("joinRoom: got: %s: %s", name, roomID)
					return joinRoom(p.Context, roomID, name)
				},
			},
			"play": &graphql.Field{
				Type:        actionResultType,
				Description: "do play in the room ",
				Args: graphql.FieldConfigArgument{
					"roomID": &graphql.ArgumentConfig{
						Type: graphql.ID,
					},
					"action": &graphql.ArgumentConfig{
						Type: actionInputType,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					roomID := p.Args["roomID"].(string)
					action := p.Args["action"].(map[string]interface{})
					actionObj := Action{Name: action["name"].(string), Bits: make([]KVPairInput, 0)}
					for _, p := range action["bits"].([]interface{}) {
						actionObj.Bits = append(actionObj.Bits, KVPairInput{
							Name:  p.(map[string]interface{})["name"].(string),
							Value: p.(map[string]interface{})["value"].(string),
						})
					}
					log.Tracef("play: got: %s: %+v", roomID, action)
					return play(p.Context, roomID, &actionObj)

				},
			},
			"deleteRoom": &graphql.Field{
				Type:        roomType,
				Description: "delete room ",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.ID,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id := p.Args["id"].(string)
					log.Tracef("deleteRoom:  %s", id)
					return deleteRoom(p.Context, id)

				},
			}},
	}

	subscription := graphql.ObjectConfig{Name: "Subscription",
		Fields: graphql.Fields{
			"roomUpdates": &graphql.Field{
				Type:        roomUpdateInfoType,
				Description: "room updates subscription",
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.ID,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {

					return map[string]interface{}{"name": "update", "value": "changed"}, nil
				},
			},
		},
	}

	schemaConfig := graphql.SchemaConfig{Query: graphql.NewObject(rootQuery),
		Subscription: graphql.NewObject(subscription),
		Mutation:     graphql.NewObject(mutation)}
	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		log.Criticalf("failed to create new schema, error: %v", err)
	}
	sch := &GomesScheme{schema: &schema, storage: stor}
	sch.initSubscription()
	Schema = sch
	return sch
}

func (gs *GomesScheme) GQLHandler(cont context.Context) iris.Handler {
	return func(ctx iris.Context) {
		gs.Process(ctx, cont)
	}
}

type request struct {
	OperationName string                 `json:"operationName"`
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
}

func (gs *GomesScheme) Process(ctx iris.Context, cont context.Context) {
	log.Debugf("Handler: new request; %v", ctx.RemoteAddr)
	// id := ctx.GetCookie(cSESSION_ID)
	// if id != "" {
	// 	log.Debugf("Handler: new request; session-id: %s", id)
	// } else {
	// 	id = utils.RandString(32)
	// 	ctx.SetCookieKV(cSESSION_ID, id, iris.CookieExpires(5*24*time.Hour))
	// 	log.Infof("Handler: generating new session-id: %s", id)
	// }
	// cont = context.WithValue(cont, cSESSION_ID, id)

	request := request{}
	err := ctx.ReadJSON(&request)
	if err != nil {
		log.Warnf("can't parse body: %v", err)
		return
	}
	log.Tracef("  got new query: %v", request)
	if request.OperationName != "Login" {
		id, _ := Authenticate(ctx, true)
		if id == "" {
			// ctx.JSON(map[string]interface{}{"error": err.Error()})
			log.Tracef("not authenticated. Returning: %d", ctx.GetStatusCode())
			return
		}
		cont = context.WithValue(cont, cSESSION_ID, id)
	}

	// cont = context.WithValue(cont, "Schema", gs)
	cont = context.WithValue(cont, "UpdateChannel", gs.updateChannel)
	result := graphql.Do(graphql.Params{
		Schema:         *gs.schema,
		RequestString:  request.Query,
		OperationName:  request.OperationName,
		VariableValues: request.Variables,
		Context:        cont,
	})
	log.Tracef("   got result: %+v", result)
	// if request.OperationName == "roomUpdates" || strings.Index(request.Query, "roomUpdates") != -1 {
	// 	log.Tracef("going to serve subscription")
	// 	// gs.ServeSubscription(ctx.ResponseWriter(), ctx.Request(), id)
	// } else {
	if len(result.Errors) > 0 {
		log.Warnf("wrong result, unexpected errors: %v", result.Errors)
		ctx.JSON(map[string]interface{}{"error": result.Errors})
	} else {
		ctx.JSON(map[string]interface{}{"data": result.Data})
	}
	// }
}

func (s *GomesScheme) initSubscription() {
	s.updateChannel = make(chan string)
	s.subscriptionManager = graphqlws.NewSubscriptionManager(s.schema)
	s.WSHandler = graphqlws.NewHandler(graphqlws.HandlerConfig{
		// Wire up the GraphqL WebSocket handler with the subscription manager
		SubscriptionManager: s.subscriptionManager,
	})
	go func() {
		for msg := range s.updateChannel {
			s.sendSubscriptions(msg)
		}
	}()
}

func (s *GomesScheme) sendSubscriptions(string) {
	subscriptions := s.subscriptionManager.Subscriptions()

	for conn, _ := range subscriptions {
		// Things you have access to here:
		// conn.ID()   // The connection ID
		// conn.User() // The user returned from the subscription manager's Authenticate

		for _, subscription := range subscriptions[conn] {
			// Things you have access to here:
			// subscription.ID            // The subscription ID (unique per conn)
			// subscription.OperationName // The name of the subcription
			// subscription.Query         // The subscription query/queries string
			// subscription.Variables     // The subscription variables
			// subscription.Document      // The GraphQL AST for the subscription
			// subscription.Fields        // The names of top-level queries
			// subscription.Connection    // The GraphQL WS connection

			// Prepare an execution context for running the query
			ctx := context.Background()

			// Re-execute the subscription query
			params := graphql.Params{
				Schema:         *s.schema, // The GraphQL schema
				RequestString:  subscription.Query,
				VariableValues: subscription.Variables,
				OperationName:  subscription.OperationName,
				Context:        ctx,
			}
			result := graphql.Do(params)

			// Send query results back to the subscriber at any point
			data := graphqlws.DataMessagePayload{
				// Data can be anything (interface{})
				Data: result.Data,
				// Errors is optional ([]error)
				Errors: graphqlws.ErrorsFromGraphQLErrors(result.Errors),
			}
			subscription.SendData(&data)
		}
	}
}
