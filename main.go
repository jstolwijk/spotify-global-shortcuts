package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/MakeNowJust/hotkey"
	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"github.com/zmb3/spotify"
)

const redirectURI = "http://localhost:8080/sessions"

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadCurrentlyPlaying, spotify.ScopeUserReadPlaybackState, spotify.ScopeUserModifyPlaybackState)
	ch    = make(chan *spotify.Client)
	state = "test_session_123"
)

func main() {

	http.HandleFunc("/sessions", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	// if you didn't store your ID and secret key in the specified environment variables,
	// you can set them manually here
	auth.SetAuthInfo(clientID, secretKey)

	// get the user to this URL - how you do that is up to you
	// you should specify a unique state string to identify the session
	url := auth.AuthURL(state)

	openbrowser(url)

	hkey := hotkey.New()

	for client := range ch {
		fmt.Println("Pausing....")

		hkey.Register(hotkey.Ctrl, '2', func() {
			playerState, err := client.PlayerState()

			if err != nil {
				log.Fatal(err)
				return
			}

			if playerState.CurrentlyPlaying.Playing {
				client.Pause()
			} else {
				client.Play()
			}
		})
		hkey.Register(hotkey.Ctrl, '3', func() {
			client.Next()
		})
		hkey.Register(hotkey.Ctrl, '1', func() {
			client.Previous()
		})

		hkey.Register(hotkey.Ctrl, '4', func() {
			playerState, err := client.PlayerState()

			if err != nil {
				log.Fatal(err)
				return
			}

			client.Seek(playerState.CurrentlyPlaying.Progress + 15000)
		})
	}

	systray.Run(onReady, onExit)
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	fmt.Fprintf(w, "Login Completed!")
	ch <- &client
}

func onReady() {
	systray.SetIcon(icon.Data)
	systray.SetTitle("Spotify global shortcuts")
	systray.SetTooltip("Enables global spotify keyboard shortcuts")
	_ = systray.AddMenuItem("Settings", "Configure settings")

	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	mQuit.SetIcon(icon.Data)
	clicked := mQuit.ClickedCh

	for click := range clicked {
		_ = click
		systray.Quit()
	}
}

func onExit() {
	fmt.Println("Shutting down")
}
