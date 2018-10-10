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
	"strconv"
)

type InstaPost struct {
	Description string `json:"description"`
	PhotoURL    string `json:"photo_url"`
	Likes       int64  `json:"likes"`
	ID          string `json:"id"`
	PostUrl     string `json:"post_url"`
}

type InstaUser struct {
	UserName string      `json:"user_name"`
	Posts    []InstaPost `json:"posts"`
}

var (
	serveHost = flag.String("serve_host", getEnv("SERVER_HOST", ""),
		"Host to serve requests incoming to Streaming Configurator")
	servePort = flag.String("serve_port", getEnv("SERVER_PORT", "8080"),
		"Port to serve requests incoming to Streaming Configurator")

	dcUserId  = flag.String("dc_user_id", getEnv("DC_USER_ID", ""), "")
	sessionID = flag.String("session_id", getEnv("SESSION_ID", ""), "")
	csrfToken = flag.String("csrf_token", getEnv("CSRF_TOKEN", ""), "")
	g         errgroup.Group
	mgr       instago.IGApiManager
)

func main() {
	mgr = *instago.NewInstagramApiManager(*dcUserId, *sessionID, *csrfToken)
	medias, _ := mgr.GetAllPostMedia("nc_ficus")
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
	r.Handle("/api/insta/{username}/{last}",
		handlers.LoggingHandler(
			os.Stdout,
			handler(handleFunc())),
	).Methods("GET")
	r.Handle("/api/insta/{username}",
		handlers.LoggingHandler(
			os.Stdout,
			handler(handleFunc())),
	).Methods("GET")

	return JsonContentType(handlers.CompressHandler(r))
}

func handleFunc() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("{\"Error\": \"%+v\"}", r)))
			}
		}()
		vars := mux.Vars(r)
		name := vars["username"]
		id := getInt(vars["last"], 0)
		medias, _ := mgr.GetAllPostMedia(name)
		resp := &InstaUser{UserName: name, Posts: []InstaPost{}}
		for _, media := range medias {
			mediaId := getInt(media.Id, 0)
			if mediaId <= id {
				break
			}
			postInfo := InstaPost{
				PhotoURL: media.DisplayUrl,
				PostUrl:  media.GetPostUrl(),
				Likes:    media.EdgeMediaPreviewLike.Count,
				ID:       media.Id,
			}
			if len(media.EdgeMediaToCaption.Edges) > 0 {
				postInfo.Description = media.EdgeMediaToCaption.Edges[0].Node.Text
			}
			resp.Posts = append(resp.Posts, postInfo)
		}
		res, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(res)
	}
}

func getInt(strValue string, defaultValue int64) int64 {
	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		fmt.Printf("Incorrect int value, default value %+v will be used ", defaultValue)
		return defaultValue
	}
	return intValue
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
