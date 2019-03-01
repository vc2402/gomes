package main

import (
	"context"
	"flag"

	"github.com/vc2402/gomes/store"

	"github.com/vc2402/gomes/games"

	"github.com/vc2402/gomes/resolve"

	"github.com/kataras/iris"
	"github.com/kataras/iris/core/host"
	"github.com/vc2402/utils"

	log "github.com/cihub/seelog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var stor *store.Store

func main() {
	flag.String("port", ":8086", "address and port to listen on (default ':8086')")
	flag.String("cfg", "properties", "configuration file name (without extension!)")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
	viper.SetConfigName(viper.GetString("cfg"))
	viper.AddConfigPath("./cfg/")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()
	if err != nil {
		log.Warnf("readConfig: %v", err)
	}
	utils.Init()
	// resolve.Init()
	initGames()
	stor = initStore()
	ctx := context.Background()
	ctx = context.WithValue(ctx, "store", stor)
	app := iris.Default()
	// app.Use(recover.New())
	app.Logger().Install(utils.ExternalLogger)

	gql := resolve.InitGraphQL(stor)

	app.Options("/api/query", CORS)
	// app.Options("/api2/query", CORS)
	app.Options("/api/query/ws", CORS)
	app.Use(CORS)
	// app.Any("/api/query", resolve.Handler(ctx))
	app.Any("/api/query", gql.GQLHandler(ctx))
	app.Any("/api/query/ws", iris.FromStd(gql.WSHandler))

	err = app.Run(iris.Addr(viper.GetString("port"), configureHost), iris.WithoutServerError(iris.ErrServerClosed))
	if err != nil {
		log.Critical("server.Run: ", err)
		log.Flush()
	}
}
func configureHost(su *host.Supervisor) {
	// here we have full access to the host that will be created
	// inside the `Run` function.
	//
	// we register a shutdown "event" callback
	su.RegisterOnShutdown(func() {
		log.Tracef("Server shus down")

	})
	// su.RegisterOnError
	// su.RegisterOnServe
}

//CORS for CORS calls
func CORS(ctx iris.Context) {
	origin := ctx.GetHeader("Origin")
	log.Tracef("CORS: origin is %s ", origin)
	if origin != "" {
		ctx.Header("Access-Control-Allow-Origin", origin)
		acrh := ctx.GetHeader("Access-Control-Request-Headers")
		if acrh != "" {
			ctx.Header("Access-Control-Allow-Headers", acrh)
		}
		ctx.Header("Access-Control-Allow-Credentials", "true")
		ctx.Header("Access-Control-Allow-Methods", "POST, GET, PUT, OPTIONS")
	}
	if ctx.Method() == "OPTIONS" {
		ctx.StatusCode(200)
		// }
	} else {
		ctx.Next()
	}
}

func initStore() *store.Store {
	// db, err := store.InitBolt("gomes.bolt")
	db, err := store.Init()
	if err == nil {
		return db
	}
	return nil
}

func stopStore() {
	if stor != nil {
		stor.Stop()
	}
}

func initGames() {
	games.InitProfessions()
	games.InitRPSGame()
}
