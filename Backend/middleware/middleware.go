package middleware

import (
	"context"
	"github.com/Jonathanpatta/rplace/auth"
	"github.com/Jonathanpatta/rplace/cache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"net/http"
	"strings"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("content-type", "application/json;charset=UTF-8")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type AuthMiddlewareServer struct {
	SessionStore *sessions.CookieStore
	CacheCli     *cache.Client
	DbCli        *dynamodb.Client
}

func NewAuthMiddlewareServer(store *sessions.CookieStore, cache *cache.Client, DbCli *dynamodb.Client) *AuthMiddlewareServer {
	return &AuthMiddlewareServer{
		SessionStore: store,
		CacheCli:     cache,
		DbCli:        DbCli,
	}
}

func (s *AuthMiddlewareServer) Authorization(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		tokenString := strings.TrimLeft(authHeader, "Bearer ")

		_, err := uuid.Parse(tokenString)

		if err != nil {
			http.Error(w, "invalid token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var token auth.Token

		err = s.CacheCli.Get("TOKEN#"+tokenString, &token)

		if err != nil {

			out, err := s.DbCli.Query(context.TODO(), &dynamodb.QueryInput{
				TableName:              aws.String("Place-Clone"),
				KeyConditionExpression: aws.String("#PK = :name"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":name": &types.AttributeValueMemberS{Value: "TOKEN#" + tokenString},
				},
				ExpressionAttributeNames: map[string]string{
					"#PK": "PK",
				},
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var tokens []auth.Token

			err = attributevalue.UnmarshalListOfMaps(out.Items, &tokens)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(tokens) > 1 {
				if err != nil {
					http.Error(w, "unique token not found", http.StatusInternalServerError)
					return
				}
			}

			if len(tokens) == 1 {
				token = tokens[0]
				if !token.IsValid() {
					if err != nil {
						http.Error(w, "token expired", http.StatusInternalServerError)
						return
					}
				}
				err := s.CacheCli.Put("TOKEN#"+token.Token, token)
				if err != nil {
					http.Error(w, "failed to put in cache", http.StatusInternalServerError)
					return
				}
			}
		}

		if !token.IsValid() {
			http.Error(w, "invalid token", http.StatusInternalServerError)
			return
		}
		next.ServeHTTP(w, r)
	})
}
