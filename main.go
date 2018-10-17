package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
)

var (
	serveHost = flag.String("serve_host", getEnv("SERVER_HOST", ""),
		"Host to serve requests incoming to Instagram Provider")
	servePort = flag.String("serve_port", getEnv("SERVER_PORT", "8080"),
		"Port to serve requests incoming to Instagram Provider")

	dcUserId         = flag.String("dc_user_id", getEnv("DC_USER_ID", ""), "")
	sessionID        = flag.String("session_id", getEnv("SESSION_ID", ""), "")
	csrfToken        = flag.String("csrf_token", getEnv("CSRF_TOKEN", ""), "")
	youTubeApiKey    = flag.String("youtube_api_key", getEnv("YOUTUBE_API_KEY", ""), "")
	g                errgroup.Group
	instagramHandler InstagramHandler
	youtubeHandler   YoutubeHandler
	monitoringHandler MonitoringHandler
)

func main() {
	instagramHandler = newInstagramHandler(*dcUserId, *sessionID, *csrfToken)
	youtubeHandler = newYouTubeHandler(*youTubeApiKey)
	monitoringHandler = NewMonitoringHandler(youtubeHandler, instagramHandler)

	medias, _ := instagramHandler.PrivateAPIManager.GetAllPostMedia("nc_ficus")
	log.Print(medias[0].DisplayUrl)

	mainEndpoints := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", *serveHost, *servePort),
		Handler: handler(),
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

func handler() http.Handler {
	r := mux.NewRouter()
	handler := Handler()
	r.Handle("/api/instagram/posts/{username}/{last}",
		handlers.LoggingHandler(
			os.Stdout,
			handler(instagramHandler.handlePostsRequest())),
	).Methods("GET")
	r.Handle("/api/instagram/posts/{username}",
		handlers.LoggingHandler(
			os.Stdout,
			handler(instagramHandler.handlePostsRequest())),
	).Methods("GET")
	r.Handle("/api/instagram/stories/{username}",
		handlers.LoggingHandler(
			os.Stdout,
			handler(instagramHandler.handleStoriesRequest())),
	).Methods("GET")
	r.Handle("/api/instagram/stories/{username}/{last}",
		handlers.LoggingHandler(
			os.Stdout,
			handler(instagramHandler.handleStoriesRequest())),
	).Methods("GET")
	r.Handle("/api/youtube/{channel}",
		handlers.LoggingHandler(
			os.Stdout,
			handler(youtubeHandler.fetchLastVideos())),
	).Methods("GET")
	r.Handle("/health",
		handlers.LoggingHandler(
			os.Stdout,
			handler(monitoringHandler.handleHealthRequest())),
	).Methods("GET")
	return JsonContentType(handlers.CompressHandler(r))
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
