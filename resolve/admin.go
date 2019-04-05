package resolve

import (
	"context"

	log "github.com/cihub/seelog"
	"github.com/kataras/iris"
	"github.com/vc2402/gomes/store"
	"github.com/vc2402/utils"
)

type adminBody struct {
	Method string                 `json:"method,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
}

func AdminHandler(cont context.Context) iris.Handler {
	return func(ctx iris.Context) {
		ProcessAdmin(ctx, cont)
	}
}

func ProcessAdmin(ctx iris.Context, cont context.Context) {
	id, _ := Authenticate(ctx, true)
	if id == "" {
		createErrorResponse(ctx, -1, "not logged in", 400)
		return
	}
	body := adminBody{}
	err := ctx.ReadJSON(&body)
	if err != nil {
		createErrorResponse(ctx, -101, "can't unmarshal request", 400)
		return
	}
	params := body.Params
	log.Tracef("admin request: %+v", body)
	switch body.Method {
	case "listUsers":
		filter := store.Filter{Limit: 100}
		if params["startsWith"] != nil {
			filter.Field = "Login"
			filter.Mask = params["startsWith"].(string)
			filter.Flags = store.FFSeek
		}
		players, err := listPlayers(filter)
		if err != nil {
			createErrorResponse(ctx, -201, "problem while listing users", 500)
			return
		}
		ctx.JSON(map[string]interface{}{"status": "ok", "code": 0, "description": "users", "users": players})
	case "createUser":
		login, lok := params["login"].(string)
		name, nok := params["name"].(string)
		if !lok || !nok {
			createErrorResponse(ctx, -202, "login and name should be set", 400)
			return
		}
		id := utils.RandString(32)
		pl, err := newPlayer(name, id, login)
		if err != nil {
			createErrorResponse(ctx, -500, "internal server error", 500)
			return
		}
		pswd, pok := params["password"].(string)
		if pok {
			pl.Password = pswd
			pl.save()
		}
		ctx.JSON(map[string]interface{}{"status": "ok", "code": 0, "description": "created", "id": id})
	case "setRoles":
		id, iok := params["userId"].(string)
		if !iok {
			createErrorResponse(ctx, -203, "userId should be set", 400)
			return
		}
		pl := GetPlayer(id)
		if pl == nil {
			createErrorResponse(ctx, -204, "user not found", 404)
			return
		}
		r, ok := params["roles"].([]interface{})
		if !ok {
			log.Debugf("setRoles: can't read roles param. Returning error")
			createErrorResponse(ctx, -206, "roles not found in request", 400)
			return
		}
		roles := make([]string, len(r), len(r))
		for i, role := range r {
			roles[i] = role.(string)
		}
		oldRoles := pl.Roles
		pl.Roles = roles
		log.Tracef("setting roles for %s(%s): %+v", pl.Name, pl.ID, roles)
		pl.save()
		ctx.JSON(map[string]interface{}{"status": "ok", "code": 0, "description": "updated", "oldRoles": oldRoles})
	case "deleteUser":
		id, iok := params["userId"].(string)
		if !iok {
			createErrorResponse(ctx, -203, "userId should be set", 400)
			return
		}
		pl, err := deletePlayer(id)
		if err != nil {
			createErrorResponse(ctx, -205, err.Error(), 400)
			return
		}
		ctx.JSON(map[string]interface{}{"status": "ok", "code": 0, "description": "deleted", "user": *pl})
	}

	// ctx.StatusCode(200)
}
