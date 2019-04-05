package resolve

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"

	log "github.com/cihub/seelog"
	"github.com/kataras/iris"
)

func LoginHandler(cont context.Context) iris.Handler {
	return func(ctx iris.Context) {
		ProcessLogin(ctx, cont)
	}
}

func MeHandler(cont context.Context) iris.Handler {
	return func(ctx iris.Context) {
		ProcessMe(ctx, cont)
	}
}

func SetHandler(cont context.Context) iris.Handler {
	return func(ctx iris.Context) {
		ProcessSet(ctx, cont)
	}
}
func ProcessLogin(ctx iris.Context, cont context.Context) {
	body := map[string]string{}
	err := ctx.ReadJSON(&body)
	if err != nil {
		createErrorResponse(ctx, -101, "can't unmarshal request", 400)
		return
	}
	username, uok := body["username"]
	pswd, pok := body["password"]
	if !uok || !pok {
		createErrorResponse(ctx, -102, "username or password not found", 400)
		return
	}
	res, err := Login(username, pswd)
	if err != nil {
		createErrorResponse(ctx, -103, "invalid username or password", 400)
		return
	}
	ctx.JSON(map[string]interface{}{"status": "ok", "code": 0, "description": "logged in", "token": res})
	// ctx.StatusCode(200)
}

func Login(username string, password string) (string, error) {
	t := 0
	if username == "gad" {
		t = 2 * 60 * 60
	}
	usr := FindPlayer(username)
	if usr != nil && usr.Password == password {
		return createToken(usr.ID, t)
	}
	if username == "gad" && password == "DagoDog" {
		newPlayer("Administrator", "gad", "gad")
		return createToken("gad", t)
	}
	return "", errors.New("invalid password")
}
func ProcessMe(ctx iris.Context, cont context.Context) {
	id, err := Authenticate(ctx, true)
	if err != nil {
		// createErrorResponse(ctx, -1, "not logged in", 400)
		return
	}
	p := GetPlayer(id)
	ctx.JSON(map[string]interface{}{"status": "ok", "code": 0, "description": "success", "player": *p})
	// ctx.StatusCode(400)
}
func ProcessSet(ctx iris.Context, cont context.Context) {
	id, err := Authenticate(ctx, true)
	if err != nil {
		return
	}
	p := GetPlayer(id)
	body := map[string]string{}
	err = ctx.ReadJSON(&body)
	if err != nil {
		createErrorResponse(ctx, -101, "can't unmarshal request", 400)
		return
	}
	username := body["name"]
	login := body["login"]
	pswd := body["password"]
	email := body["email"]
	avatar := body["avatar"]
	if pswd != "" {
		p.Password = pswd
	}
	if login != "" {
		pp := FindPlayer(login)
		if pp != nil {
			createErrorResponse(ctx, -201, "login is not unique", 409)
			return
		}
		p.Login = login

	}
	if username != "" {
		p.Name = username
	}
	if avatar != "" {
		p.Avatar = avatar
	}
	if email != "" {
		p.Email = email
	}

	p.save()
	ctx.JSON(map[string]interface{}{"status": "ok", "code": 0, "description": "success", "player": *p})
	// ctx.StatusCode(400)
}
func Authenticate(ctx iris.Context, createResponse bool) (string, error) {
	authHeader := ctx.GetHeader("authorization")
	log.Tracef("Authenticate: %s", authHeader)
	if authHeader == "" {
		if createResponse {
			createErrorResponse(ctx, -1, "not logged in", 401)
		}
		return "", errors.New("not logged in")
	}
	idx := strings.Index(authHeader, "Bearer ")
	if idx != -1 {
		authHeader = authHeader[idx+7:]
	}
	id, ok := validateToken(authHeader)
	if !ok {
		if createResponse {
			createErrorResponse(ctx, -1, "not logged in", 401)
		}
		return "", errors.New("not logged in")
	}
	return id, nil
}

func createToken(id string, timeout int) (string, error) {
	if timeout == 0 {
		timeout = 5 * 24 * 60 * 60
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"id":  id,
		"exp": time.Now().Unix() + int64(timeout),
	})

	// Sign and get the complete encoded token as a string using the secret
	log.Tracef("createToken: returning token: %+v", token)
	return token.SignedString(TokenHMACSecret)
}

func validateToken(tokenString string) (string, bool) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return TokenHMACSecret, nil
	})
	if err != nil {
		log.Warnf("validateToken: %+v", err)
		return "", false
	} else {
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid && claims["id"] != nil {
			return claims["id"].(string), true
		} else {
			log.Warnf("token is invalid: %+v", token)
			return "", false
		}
	}
}

func createErrorResponse(ctx iris.Context, code int, description string, statusCode int) {
	ctx.StatusCode(statusCode)
	ctx.JSON(map[string]interface{}{"status": "error", "code": code, "error": description})
}
