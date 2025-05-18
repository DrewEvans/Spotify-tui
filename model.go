package main

import (
	// "log"

	// "github.com/charmbracelet/bubbles/textarea"
	// "github.com/charmbracelet/bubbles/textinput"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"


	"encoding/json"
    "fmt"
    "net/http"
    "io/ioutil"
	"os"
)

const (
    defaultView = iota
    playerView
)

// User struct matches the expected JSON from Spotify API
type User struct {
	Country     string `json:"country"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	ExplicitContent struct {
		FilterEnabled bool `json:"filter_enabled"`
		FilterLocked  bool `json:"filter_locked"`
	} `json:"explicit_content"`
	ExternalUrls struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
	Followers struct {
		Href  string `json:"href"`
		Total int    `json:"total"`
	} `json:"followers"`
	Href   string `json:"href"`
	ID     string `json:"id"`
	Images []struct {
		URL    string `json:"url"`
		Height int    `json:"height"`
		Width  int    `json:"width"`
	} `json:"images"`
	Product string `json:"product"`
	Type    string `json:"type"`
	URI     string `json:"uri"`
}

// Msg type for passing user data into the model
type userMsg struct {
	data User
	err  error
}

// Bubbletea model
type model struct {
	user User
	err  error
	state int
}

var (
	appNameStyle = lipgloss.NewStyle().Background(lipgloss.Color("99")).Padding(0,1)

	faintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Faint(true)

	listEnumeratorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).MarginRight(1)
)

// Init kicks off the API call
func (m model) Init() bubbletea.Cmd {
	return fetchAPI()
}

// Update handles the incoming message
func (m model) Update(msg bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	switch msg := msg.(type) {

    case userMsg:
        if msg.err != nil {
            m.err = msg.err
        } else {
            m.user = msg.data
        }
        return m, nil

    case bubbletea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, bubbletea.Quit
        case "p":
            m.state = playerView
            return m, nil
        case "esc":
            m.state = defaultView
            return m, nil
        }
    }

    return m, nil
}

// View renders the UI
func (m model) View() string {
	s := appNameStyle.Render(`Termify`) + "\n"

	if m.state == defaultView {
		s += faintStyle.Render("Press 'p' to enter player.") + "\n\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}
	
	if m.user.DisplayName == "" {
		return "Loading user data... \n\npress 'q' | 'ctrl+c' to quit."
	} else {
		s += fmt.Sprintf(
			"User: %s\nFollowers: %d\nSpotify Profile: %s\n",
			m.user.DisplayName,
			m.user.Followers.Total,
			m.user.ExternalUrls.Spotify,
		)
	}

	if m.state == playerView {
		s += "\n\n" + listEnumeratorStyle.Render("play")
		s += faintStyle.Render("Press 'esc' to exit player.") + "\n\n"
	}

	s += faintStyle.Render("'ctrl+c' | 'q' to quit.") + "\n\n"

		
	return s

}

// Command to fetch user data from Spotify
func fetchAPI() bubbletea.Cmd {
	url := "https://api.spotify.com/v1/me"
	apiKey := os.Getenv("SPOTIFY_TOKEN")
	
	return func() bubbletea.Msg {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return userMsg{err: err}
		}

		// Replace this with your actual token
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return userMsg{err: err}
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return userMsg{err: err}
		}

		var u User
		if err := json.Unmarshal(body, &u); err != nil {
			return userMsg{err: err}
		}
		return userMsg{data: u}
	}
}
