package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"runtime/debug"
)

var query = "{\n  graphQLHub\n  twitter {\n    user(identifier: name, identity: \"%s\") {\n      id\n      screen_name\n      name\n      tweets(limit: 5) {\n        text\n        id\n      }\n    }\n  }\n}\n"

type Request struct {
	Query string `json:"query"`
}

type Response struct {
	Data struct {
		Twitter struct {
			User TwitterUser `json:"user"`
		} `json:"twitter"`
	} `json:"data"`
}

type Tweet struct {
	Id   string `json:"id"`
	Text string `json:"text"`
}

type TwitterUser struct {
	Name       string  `json:"name"`
	Tweets     []Tweet `json:"tweets"`
	ScreenName string  `json:"screen_name"`
}

type TwitterHandler struct {
	client http.Client
}

func NewTwitterHandler() TwitterHandler {
	return TwitterHandler{client: http.Client{}}
}

func (handler TwitterHandler) fetchNewTweets() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("{\"Error\": \"%+v\"}", r)))
				fmt.Println("recovered from ", r)
				debug.PrintStack()
			}
		}()

		vars := mux.Vars(r)
		userName := vars["user"]

		resp := handler.getLastTweets(userName)
		res, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(res)
	}
}

func (handler TwitterHandler) getLastTweets(userName string) TwitterUser {
	buidQuery := fmt.Sprintf(query, userName)
	req := Request{Query: buidQuery}
	jsonReq, _ := json.Marshal(req)
	resp, _ := handler.client.Post("https://www.graphqlhub.com/graphql", "application/json", bytes.NewBuffer(jsonReq))
	body, _ := ioutil.ReadAll(resp.Body)
	var result Response
	err := json.Unmarshal(body, &result)
	if err != nil {
		log.Panic(err)
	}
	return result.Data.Twitter.User
}
