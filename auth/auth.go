package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
	"time"
)

type Token struct {
	Token     string `json:"token,omitempty"`
	ValidTill int64  `json:"valid_till,omitempty"`
	LastUsed  int64  `json:"last_used,omitempty"`
}

func (t *Token) IsValid() bool {
	if time.Now().Unix() > t.ValidTill {
		return false
	}
	if t.Token == "" {
		return false
	}

	return true
}

type User struct {
	PK       string
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    Token
}

type Server struct {
	DbCli         *dynamodb.Client
	SessionsStore *sessions.CookieStore
	TableName     *string
}

func NewServer(DbCli *dynamodb.Client, store *sessions.CookieStore) *Server {
	return &Server{
		DbCli:         DbCli,
		TableName:     aws.String("auth"),
		SessionsStore: store,
	}
}

func GenerateNewToken() *Token {
	token := uuid.New()
	return &Token{
		Token:     token.String(),
		ValidTill: time.Now().Add(time.Hour * 24 * 14).Unix(),
	}
}

func (s *Server) GenerateToken(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), 14)

	out, err := s.DbCli.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: s.TableName,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "USER#" + user.Username + "#" + string(bytes)},
		},
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = attributevalue.UnmarshalMap(out.Item, &user)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if user.Token.IsValid() {
		tokenJson, err := json.Marshal(user.Token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, tokenJson)
	} else {
		token := GenerateNewToken()
		user.Token = *token

		_, err := s.DbCli.PutItem(context.TODO(), &dynamodb.PutItemInput{
			Item: map[string]types.AttributeValue{
				"PK":       &types.AttributeValueMemberS{Value: user.PK},
				"SK":       &types.AttributeValueMemberN{Value: strconv.Itoa(int(time.Now().Unix()))},
				"username": &types.AttributeValueMemberS{Value: user.Username},
				"password": &types.AttributeValueMemberS{Value: user.Password},
				"token": &types.AttributeValueMemberM{
					Value: map[string]types.AttributeValue{
						"token":      &types.AttributeValueMemberS{Value: user.Token.Token},
						"valid_till": &types.AttributeValueMemberN{Value: strconv.Itoa(int(user.Token.ValidTill))},
						"last_used":  &types.AttributeValueMemberN{Value: strconv.Itoa(int(user.Token.LastUsed))},
					},
				},
			},
			TableName: s.TableName,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = s.DbCli.PutItem(context.TODO(), &dynamodb.PutItemInput{
			Item: map[string]types.AttributeValue{
				"PK":         &types.AttributeValueMemberS{Value: "TOKEN#" + user.Token.Token},
				"SK":         &types.AttributeValueMemberN{Value: strconv.Itoa(int(user.Token.ValidTill))},
				"token":      &types.AttributeValueMemberS{Value: user.Token.Token},
				"valid_till": &types.AttributeValueMemberN{Value: strconv.Itoa(int(user.Token.ValidTill))},
				"last_used":  &types.AttributeValueMemberN{Value: strconv.Itoa(int(user.Token.LastUsed))},
			},
			TableName: s.TableName,
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tokenJson, err := json.Marshal(user.Token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, tokenJson)
	}
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {

}

func NewRouter(DbCli *dynamodb.Client, store *sessions.CookieStore) *mux.Router {
	server := NewServer(DbCli, store)
	r := mux.NewRouter()

	r.HandleFunc("/getToken", server.GenerateToken).Methods("POST")
	r.HandleFunc("/register", server.Register).Methods("POST")

	return r
}
