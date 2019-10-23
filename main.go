package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"

	log "github.com/sirupsen/logrus"

	"github.com/zmb3/spotify"
)

var (
	state  string = "jfkdlsajfkldjafljdlajf"
	client *spotify.Client
)

func fatalOnErr(msg string, err error) {
	if err != nil {
		log.Fatalf("%s: %s", err)
	}
}

func handler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func main() {
	// Setup amqp
	// conn, err := amqp.Dial(os.Args[1])
	// fatalOnErr("dial amqp", err)
	// ch, err := conn.Channel()
	// fatalOnErr("amqp channel", err)
	// defer ch.Close()
	// _, err = ch.QueueDeclare("spotify", false, false, false, false, nil) //TODO: make configurable
	// fatalOnErr("declaring queue", err)

	// setup spotify
	auth := spotify.NewAuthenticator(
		"http://localhost:1234/cb",
		spotify.ScopeUserReadCurrentlyPlaying,
		spotify.ScopeUserReadPlaybackState,
		spotify.ScopeUserReadRecentlyPlayed,
	)
	auth.SetAuthInfo(CLIENTID, SECRET)

	http.HandleFunc("/", handler(func(w http.ResponseWriter, r *http.Request) error {
		if client == nil {
			http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
			return nil
		}

		items, err := client.PlayerRecentlyPlayed()
		if err != nil {
			return fmt.Errorf("Getting recently played: %s", err)
		}
		for _, track := range items {
			io.WriteString(w, track.Track.String()+"\n")
		}
		return nil
	}))
	http.HandleFunc("/auth", handler(func(w http.ResponseWriter, r *http.Request) error {
		http.Redirect(w, r, auth.AuthURL(state), http.StatusTemporaryRedirect)
		return nil
	}))

	http.HandleFunc("/cb", handler(func(w http.ResponseWriter, r *http.Request) error {
		tok, err := auth.Token(state, r)
		if err != nil {
			return fmt.Errorf("Getting token: %s", err)
		}
		if st := r.FormValue("state"); st != state {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return nil
		}
		c := auth.NewClient(tok)
		client = &c
		user, err := client.CurrentUser()
		if err != nil {
			return fmt.Errorf("Auth: Error getting user from client: %s", err)
		}
		log.Infof("Auth successful, user: %s", user.ID)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return nil
	}))

	go http.ListenAndServe(":1234", nil)

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh
	log.Info("Exiting on signal...")
}
