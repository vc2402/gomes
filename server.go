package main

import (
	"context"
	"flag"

	"github.com/vc2402/gomes/resolve"

	"github.com/kataras/iris"
	"github.com/vc2402/utils"

	log "github.com/cihub/seelog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

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
	resolve.Init()

	ctx := context.Background()

	app := iris.Default()
	// app.Use(recover.New())
	app.Logger().Install(utils.ExternalLogger)

	app.Options("/api/*", CORS)
	app.Use(CORS)
	app.Any("/api/query", resolve.Handler(ctx))

	err = app.Run(iris.Addr(viper.GetString("port")))
	if err != nil {
		log.Critical("server.Run: ", err)
		log.Flush()
	}
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
