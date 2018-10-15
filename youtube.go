package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
	"log"
	"net/http"
	"runtime/debug"
)

type Video struct {
	Id           string `id:"id"`
	Link         string `json:"link"`
	Description  string `json:"description"`
	Title        string `json:"title"`
	IsLive       bool   `json:"live"`
	ThumbnailURL string `json:"thumbnail_url"`
}

type YoutubeHandler struct {
	service youtube.Service
}

func newYouTubeHandler(youTubeApiKey string) YoutubeHandler {
	transport := &transport.APIKey{
		Key: youTubeApiKey,
	}
	client := &http.Client{Transport: transport}

	var err error
	service, err := youtube.New(client)
	if err != nil {
		log.Panicf("ERROR in creating youtube New client ", err)
	}

	return YoutubeHandler{service: *service}
}

func (handler YoutubeHandler) fetchLastVideos() func(w http.ResponseWriter, r *http.Request) {
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
		channelName := vars["channel"]

		channelId := handler.getChannelIdByChannelName(channelName)
		searchVideosResponse, _ := handler.getLastVideosInChannel(channelId, 5)
		items := searchVideosResponse.Items

		var resp []Video
		for _, item := range items {
			video := Video{
				Id:           item.Id.VideoId,
				Link:         "https://www.youtube.com/watch?v=" + item.Id.VideoId,
				Description:  item.Snippet.Description,
				Title:        item.Snippet.Title,
				IsLive:       item.Snippet.LiveBroadcastContent == "live",
				ThumbnailURL: item.Snippet.Thumbnails.High.Url,
			}
			resp = append(resp, video)
		}
		res, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(res)
	}
}

func (handler YoutubeHandler) searchChannels(part string, query string, maxResults int64) (*youtube.SearchListResponse, error) {
	return handler.service.Search.List(part).Type("channel").Q(query).MaxResults(maxResults).Do()
}

func (handler YoutubeHandler) getLastVideosInChannel(channelId string, maxResults int64) (*youtube.SearchListResponse, error) {
	return handler.service.Search.List("snippet").Type("video").ChannelId(channelId).MaxResults(maxResults).Order("date").Do()
}

func (handler YoutubeHandler) getChannelIdByChannelName(channelName string) string {
	searchListResponse, err := handler.searchChannels("snippet", channelName, 1)
	if err != nil {
		panic("Error in channel search")
	}
	if len(searchListResponse.Items) == 0 {
		panic("Channel not found")
	}

	return searchListResponse.Items[0].Snippet.ChannelId
}
