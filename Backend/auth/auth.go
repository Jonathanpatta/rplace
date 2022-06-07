package auth

import (
	"context"
	"encoding/json"
	"errors"
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
	PK        string `json:"PK,omitempty"`
	Token     string `json:"token,omitempty"`
	ValidTill int64  `json:"valid_till,omitempty"`
	LastUsed  int64  `json:"last_used,omitempty"`
}

func (t *Token) CreatePk() {
	t.PK = "TOKEN#" + t.Token
}

func (t *Token) IsValid() bool {
	now := time.Now().Unix()
	if now > t.ValidTill {
		return false
	}
	if t.Token == "" {
		return false
	}

	_, err := uuid.Parse(t.Token)
	if err != nil {
		return false
	}

	return true
}

type User struct {
	PK             string
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	HashedPassword string `json:"hashed_password,omitempty"`
	Token          Token
}

func (u *User) CreatePk() {
	u.PK = "USER#" + u.Username
}

type Server struct {
	DbCli         *dynamodb.Client
	SessionsStore *sessions.CookieStore
	TableName     *string
}

func NewServer(DbCli *dynamodb.Client, store *sessions.CookieStore) *Server {
	return &Server{
		DbCli:         DbCli,
		TableName:     aws.String("Place-Clone"),
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

func (s *Server) IsValidUser(r *http.Request) (User, error) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		return User{}, err
	}

	user.CreatePk()

	out, err := s.DbCli.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              s.TableName,
		KeyConditionExpression: aws.String("#PK = :name"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":name": &types.AttributeValueMemberS{Value: user.PK},
		},
		ExpressionAttributeNames: map[string]string{
			"#PK": "PK",
		},
	})
	if err != nil {
		return User{}, err
	}

	var users []User
	err = attributevalue.UnmarshalListOfMaps(out.Items, &users)
	if err != nil {
		return User{}, err
	}
	if len(users) != 1 {
		return User{}, errors.New("unique user not returned")
	}
	err = bcrypt.CompareHashAndPassword([]byte(users[0].HashedPassword), []byte(user.Password))
	if err != nil {
		return User{}, err
	}

	return users[0], nil
}

func (s *Server) GenerateToken(w http.ResponseWriter, r *http.Request) {
	user, err := s.IsValidUser(r)
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
		generatedNewToken := GenerateNewToken()
		user.Token = *generatedNewToken

		_, err = s.DbCli.PutItem(context.TODO(), &dynamodb.PutItemInput{
			Item: map[string]types.AttributeValue{
				"PK":         &types.AttributeValueMemberS{Value: "TOKEN#" + user.Token.Token},
				"SK":         &types.AttributeValueMemberS{Value: strconv.Itoa(int(user.Token.ValidTill))},
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

		tokenStore, err := s.SessionsStore.Get(r, "Token")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tokenStore.Values["token"] = generatedNewToken

		tokenJson, err := json.Marshal(user.Token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, string(tokenJson))
	}
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user.HashedPassword = string(bytes)
	user.CreatePk()

	out, err := s.DbCli.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              s.TableName,
		KeyConditionExpression: aws.String("#PK = :name"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":name": &types.AttributeValueMemberS{Value: user.PK},
		},
		ExpressionAttributeNames: map[string]string{
			"#PK": "PK",
		},
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(out.Items) != 0 {
		http.Error(w, "user already exists", http.StatusInternalServerError)
		return
	}

	_, err = s.DbCli.PutItem(context.TODO(), &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"PK":             &types.AttributeValueMemberS{Value: user.PK},
			"SK":             &types.AttributeValueMemberS{Value: strconv.Itoa(int(time.Now().Unix()))},
			"username":       &types.AttributeValueMemberS{Value: user.Username},
			"hashedpassword": &types.AttributeValueMemberS{Value: user.HashedPassword},
		},
		TableName: s.TableName,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello")
}

type Options struct {
	DbCli *dynamodb.Client
	Store *sessions.CookieStore
}

func NewRouter(DbCli *dynamodb.Client, store *sessions.CookieStore) *mux.Router {
	server := NewServer(DbCli, store)
	r := mux.NewRouter()

	r.HandleFunc("/generateToken", server.GenerateToken).Methods("POST")
	r.HandleFunc("/register", server.Register).Methods("POST")
	r.HandleFunc("/ping", server.Ping).Methods("GET")

	return r
}

func AddSubrouter(o *Options, r *mux.Router) {
	server := NewServer(o.DbCli, o.Store)
	router := r.PathPrefix("/auth").Subrouter()

	router.HandleFunc("/generateToken", server.GenerateToken).Methods("POST")
	router.HandleFunc("/register", server.Register).Methods("POST")
	router.HandleFunc("/ping", server.Ping).Methods("GET")
}
