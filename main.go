package main

import (
	"context"
	"fmt"
	"github.com/Jonathanpatta/rplace/auth"
	"github.com/Jonathanpatta/rplace/cache"
	"github.com/Jonathanpatta/rplace/middleware"
	"github.com/Jonathanpatta/rplace/placeclone"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-south-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	client, err := cache.NewClient("/cachedb")
	if err != nil {
		fmt.Println("cache client could not be created")
	}

	client.ClearAll()

	if err != nil {
		fmt.Println(err.Error())
	}

	DbCli := dynamodb.NewFromConfig(cfg)
	sessionStore := sessions.NewCookieStore([]byte("aksjdfjjlasdfjlkjlasdf"))
	middlewareServer := middleware.NewAuthMiddlewareServer(sessionStore, client, DbCli)

	mainRouter := mux.NewRouter()

	mainRouter.Use(middleware.CorsMiddleware)

	placeclone.AddSubrouter(DbCli, sessionStore, client, middlewareServer, mainRouter)
	auth.AddSubrouter(DbCli, sessionStore, mainRouter)

	http.Handle("/", mainRouter)

	http.ListenAndServe(":8000", nil)

}
