package main

import (
	"context"
	"github.com/Jonathanpatta/rplace/middleware"
	"github.com/Jonathanpatta/rplace/placeclone"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gorilla/sessions"
	"log"
	"net/http"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-south-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	DbCli := dynamodb.NewFromConfig(cfg)
	sessionStore := sessions.NewCookieStore([]byte("aksjdfjjlasdfjlkjlasdf"))

	placecloneRouter := placeclone.NewRouter(DbCli, sessionStore)
	placecloneRouter.Use(middleware.CorsMiddleware)
	http.Handle("/", placecloneRouter)

	http.ListenAndServe(":8000", nil)

}
