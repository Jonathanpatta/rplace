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
	userpoolId := "ap-south-1_DTkRR7wmN"
	middlewareServer := middleware.NewAuthMiddlewareServer(sessionStore, client, DbCli, userpoolId)

	mainRouter := mux.NewRouter()

	mainRouter.Use(middleware.CorsMiddleware)

	placecloneServerOptions := &placeclone.Options{
		DbCli:          DbCli,
		Store:          sessionStore,
		CacheCli:       client,
		AuthMiddleware: middlewareServer,
	}

	authServerOptions := &auth.Options{
		DbCli: DbCli,
		Store: sessionStore,
	}

	placeclone.AddSubrouter(placecloneServerOptions, mainRouter)
	auth.AddSubrouter(authServerOptions, mainRouter)

	http.Handle("/", mainRouter)

	http.ListenAndServe(":8000", nil)

}
