package placeclone

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Jonathanpatta/rplace/cache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	DbCli        *dynamodb.Client
	TableName    *string
	SessionStore *sessions.CookieStore
	Image        *Image
}

func NewServer(DbCli *dynamodb.Client, store *sessions.CookieStore) Server {
	client, err := cache.NewClient("/cachedb")
	if err != nil {
		fmt.Println("cache client could not be created")
	}
	err = client.Put("string1", Pixel{
		Pk:           "laskjdf",
		Sk:           "asdf",
		Row:          12,
		Col:          23,
		Color:        "asfd",
		Author:       "asdf",
		LastModified: time.Now().Unix(),
	})
	if err != nil {
		fmt.Println("Could not put into cache")
	}
	var pixel Pixel
	err = client.Get("string1", &pixel)
	if err != nil {
		fmt.Println("Could not get from cache")
	}

	fmt.Println(pixel)

	err = client.Delete("string1")
	if err != nil {
		fmt.Println("Could not delete from cache")
	}

	err = client.Get("string1", &pixel)
	if err != nil {
		fmt.Println("Could not get from cache")
	}

	return Server{
		DbCli:        DbCli,
		TableName:    aws.String("Place-Clone"),
		Image:        NewImage("main image", 25, 25),
		SessionStore: store,
	}
}

func (s *Server) Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello")
}

func (s *Server) Home(w http.ResponseWriter, r *http.Request) {

	//start := time.Now()
	//output, err := s.DbCli.Scan(context.TODO(), &dynamodb.ScanInput{
	//	TableName: s.TableName,
	//})
	//end := time.Now()
	//
	//if err != nil {
	//	fmt.Println("ERROR OVER HERE")
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//fmt.Println("PRINT OVER HERE")
	//var items []Item
	//err = attributevalue.UnmarshalListOfMaps(output.Items, &items)
	//
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//
	//for _, item := range items {
	//	fmt.Fprintln(w, item)
	//	fmt.Fprintln(w, "******************")
	//}
	//fmt.Fprintln(w, "completed in:", end.Sub(start).Nanoseconds()/(1000000))
}

func (s *Server) UpdatePixel(w http.ResponseWriter, r *http.Request) {
	var p Pixel
	err := json.NewDecoder(r.Body).Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedPixel, err := s.Image.UpdatePixelFromObject(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out, err := s.DbCli.PutItem(context.TODO(), &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"PK":            &types.AttributeValueMemberS{Value: updatedPixel.Pk},
			"SK":            &types.AttributeValueMemberS{Value: updatedPixel.Sk},
			"row":           &types.AttributeValueMemberN{Value: strconv.Itoa(updatedPixel.Row)},
			"col":           &types.AttributeValueMemberN{Value: strconv.Itoa(updatedPixel.Col)},
			"color":         &types.AttributeValueMemberS{Value: updatedPixel.Color},
			"last_modified": &types.AttributeValueMemberN{Value: strconv.Itoa(int(updatedPixel.LastModified))},
		},
		TableName: s.TableName,
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, out.Attributes)
}

func (s *Server) GetPixels(w http.ResponseWriter, r *http.Request) {
	_, err := json.Marshal(s.Image)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out, err := s.DbCli.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              s.TableName,
		KeyConditionExpression: aws.String("#PK = :name"),
		FilterExpression:       aws.String("(#row between :zero and :rows) and (#col between :zero and :cols)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":name": &types.AttributeValueMemberS{Value: s.Image.Name},
			//":sortKey": &types.AttributeValueMemberS{Value: ""},
			":rows": &types.AttributeValueMemberN{Value: strconv.Itoa(s.Image.Rows)},
			":cols": &types.AttributeValueMemberN{Value: strconv.Itoa(s.Image.Cols)},
			":zero": &types.AttributeValueMemberN{Value: strconv.Itoa(0)},
		},
		ExpressionAttributeNames: map[string]string{
			"#PK":  "PK",
			"#row": "row",
			"#col": "col",
		},
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var pixels []Pixel
	err = attributevalue.UnmarshalListOfMaps(out.Items, &pixels)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pixelArray, err := json.Marshal(pixels)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, string(pixelArray))

}

func NewRouter(DbCli *dynamodb.Client, store *sessions.CookieStore) *mux.Router {
	server := NewServer(DbCli, store)

	r := mux.NewRouter()

	r.HandleFunc("/ping", server.Ping)
	r.HandleFunc("/", server.Home)
	r.HandleFunc("/pixels", server.GetPixels).Methods("GET")
	r.HandleFunc("/updatePixel", server.UpdatePixel).Methods("POST")

	return r
}
