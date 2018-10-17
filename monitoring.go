package main

import (
	"net/http"
	"runtime/debug"
	"encoding/json"
	"log"
	"fmt"
)

type MonitoringHandler struct {
	InstagramHandler InstagramHandler
	YoutubeHandler   YoutubeHandler
}

type MonitoringStatus struct {
	Health string `json:"health"`
	Error  string `json:"error,omitempty"`
}

func NewMonitoringHandler(youtubeHandler YoutubeHandler, instagramHandler InstagramHandler) MonitoringHandler {
	return MonitoringHandler{InstagramHandler: instagramHandler, YoutubeHandler: youtubeHandler}
}

func (handler MonitoringHandler) handleHealthRequest() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				resp := MonitoringStatus{Health: "WARNING", Error: fmt.Sprintf("Error when working service: %+v ", r)}
				res, _ := json.Marshal(resp)
				w.WriteHeader(500)
				w.Write(res)
				log.Println("recovered from ", r)
				debug.PrintStack()
				return
			}
		}()

		if len(handler.InstagramHandler.getPosts("nc_ficus", 0).Posts) == 0 {
			resp := MonitoringStatus{Health: "WARNING", Error: "Cannot obtain posts of public instagram channel"}
			res, _ := json.Marshal(resp)
			w.WriteHeader(500)
			w.Write(res)
			return
		}
		if len(handler.InstagramHandler.getPosts("st4s_r", 0).Posts) == 0 {
			resp := MonitoringStatus{Health: "WARNING", Error: "Cannot obtain posts of private instagram channel"}
			res, _ := json.Marshal(resp)
			w.WriteHeader(500)
			w.Write(res)
			return
		}
		if len(handler.YoutubeHandler.getLastVideos("mrMigles")) == 0 {
			resp := MonitoringStatus{Health: "WARNING", Error: "Cannot obtain videos youtube channel"}
			res, _ := json.Marshal(resp)
			w.WriteHeader(500)
			w.Write(res)
			return
		}
		resp := MonitoringStatus{Health: "UP"}
		res, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(res)
	}
}
