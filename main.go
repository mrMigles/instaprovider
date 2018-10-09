package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/siongui/instago"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
)

type UserPhoto struct {
	Description string
	URL         string
	Likes       int
}

type InstaUser struct {
	UserName string
	Photos   []UserPhoto
}

var (
	serveHost = flag.String("serve_host", getEnv("SERVER_HOST", ""),
		"Host to serve requests incoming to Streaming Configurator")
	servePort = flag.String("serve_port", getEnv("SERVER_PORT", "8080"),
		"Port to serve requests incoming to Streaming Configurator")
	g   errgroup.Group
	mgr instago.IGApiManager
)

func main() {
	mgr = *instago.NewInstagramApiManager("IG_DS_USER_ID", "IG_SESSIONID", "IG_CSRFTOKEN")
	medias, _ := mgr.GetAllPostMedia("sergeyivanov93")
	log.Print(medias[0].DisplayUrl)

	mainEndpoints := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", *serveHost, *servePort),
		Handler: RemindHandler(),
	}

	g.Go(func() error {
		return mainEndpoints.ListenAndServe()
	})

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}

}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func RemindHandler() http.Handler {
	r := mux.NewRouter()
	handler := Handler()
	r.Handle("/api/insta/{username}",
		handlers.LoggingHandler(
			os.Stdout,
			handler(HandleRemind())),
	).Methods("GET")

	return JsonContentType(handlers.CompressHandler(r))
}

func HandleRemind() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["username"]
		medias, _ := mgr.GetAllPostMedia(name)
		resp := &InstaUser{UserName: name, Photos: []UserPhoto{}}
		for _, media := range medias {
			resp.Photos = append(resp.Photos, UserPhoto{Description: media.Typename, URL: media.DisplayUrl})
		}
		res, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(res)
	}
}

func Handler() func(func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return func(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
		h := http.HandlerFunc(f)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	}
}

func JsonContentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
	})
}
