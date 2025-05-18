package main

import (


	// "github.com/charmbracelet/bubbles/textarea"
	// "github.com/charmbracelet/bubbles/textinput"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"


	"bytes"
    "encoding/base64"
    "net/http"
    "net/url"
    "io/ioutil"
    "os"
    "encoding/json"
    "fmt"
	
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
	data  User
	err   error
	token string
}

type curlMsg struct {
    data string
    err  error
}

type spotifyToken struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    ExpiresIn   int    `json:"expires_in"`
}

// Bubbletea model
type model struct {
	user User
	err  error
	state int
	AccessToken string
}

var (
	appNameStyle = lipgloss.NewStyle().Background(lipgloss.Color("99")).Padding(0,1)

	faintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Faint(true)

	listEnumeratorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).MarginRight(1)
)

// Init kicks off the API call
func (m model) Init() bubbletea.Cmd {
	return func() bubbletea.Msg {
        apiKey := os.Getenv("SPOTIFY_TOKEN")
        if apiKey == "" {
            var err error
            apiKey, err = fetchSpotifyToken()
            if err != nil {
                return userMsg{err: err}
            }
            os.Setenv("SPOTIFY_TOKEN", apiKey)
        }
        // Pass the token along with userMsg, or set it in the model after fetchAPI
        return userMsg{data: User{}, err: nil, token: apiKey}
    }
}

// Update handles the incoming message
func (m model) Update(msg bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	switch msg := msg.(type) {

    case userMsg:
        if msg.err != nil {
            m.err = msg.err
        } else {
            m.user = msg.data
            m.AccessToken = msg.token // store the token
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
		case "`": // backtick key
			if m.state == defaultView {
				token, err := fetchSpotifyToken()
				if err != nil {
					m.err = err
				} else {
					m.AccessToken = token
					os.Setenv("SPOTIFY_TOKEN", token)
				}
			}
		case "right":
			if m.state == playerView {
				return m, postSkipTrack("next")
			}
		case "left":
			if m.state == playerView {
				return m, postSkipTrack("previous")
			}
        }
    }

    return m, nil
}

// View renders the UI
func (m model) View() string {
	s := appNameStyle.Render(`Termify`) + "\n"

	if m.AccessToken == "" {
        s += "No access token available.\n"
    } else {
		fmt.Println("Access token: ", m.AccessToken)
        s += "Access token present.\n"
    }
	
	//Default view
	if m.state == defaultView {
		s += faintStyle.Render("Press 'p' to enter player.") + "\n\n"
	}

	// if m.user.DisplayName == "" {
	// 	return "Loading user data... \n\npress 'q' | 'ctrl+c' to quit."
	// } else {
	// 	s += fmt.Sprintf(
	// 		"User: %s\nFollowers: %d\nSpotify Profile: %s\n",
	// 		m.user.DisplayName,
	// 		m.user.Followers.Total,
	// 		m.user.ExternalUrls.Spotify,
	// 	)
	// }

	//Spotify Player view 
	if m.state == playerView {
		s += "\n\n" + listEnumeratorStyle.Render("'\u2190' previous track\n'enter' to play & pause\n '\u2192'skip track")
		s += faintStyle.Render("Press 'esc' to exit player.") + "\n\n"
	}

	s += faintStyle.Render("'ctrl+c' | 'q' to quit.") + "\n\n"

		
	return s

}

// Command to fetch user data from Spotify
func fetchAPI() bubbletea.Cmd {
    return func() bubbletea.Msg {
        apiKey := os.Getenv("SPOTIFY_TOKEN")
        if apiKey == "" {
            // Try to fetch a new token using client credentials flow
            var err error
            apiKey, err = fetchSpotifyToken()
            if err != nil {
                return userMsg{err: err}
            }
            // Optionally, you can set this token in the environment for later use
            os.Setenv("SPOTIFY_TOKEN", apiKey)
        }

        url := "https://api.spotify.com/v1/me"
        req, err := http.NewRequest("GET", url, nil)
        if err != nil {
            return userMsg{err: err}
        }
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
        return userMsg{data: u, token: apiKey}
    }
}

func postSkipTrack(skipDirection string) bubbletea.Cmd {
    return func() bubbletea.Msg {
        apiKey := os.Getenv("SPOTIFY_TOKEN")
        url := "https://api.spotify.com/v1/me/player/"

        req, err := http.NewRequest("POST", url+skipDirection, nil)
        if err != nil {
            return curlMsg{err: err}
        }
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Accept", "application/json")

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            return curlMsg{err: err}
        }
        defer resp.Body.Close()

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            return curlMsg{err: err}
        }
		if skipDirection == "next"{
			fmt.Println("Next track skipped")
		}

		if skipDirection == "previous" {
			fmt.Println("track skipped to previous")
		}
		
        return curlMsg{data: string(body)}
    }
}

func fetchSpotifyToken() (string, error) {
    clientID := os.Getenv("SPOTIFY_CLIENT_ID")
    clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
    if clientID == "" || clientSecret == "" {
        return "", fmt.Errorf("missing client ID or secret")
    }

    data := url.Values{}
    data.Set("grant_type", "client_credentials")

    req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data.Encode()))
    if err != nil {
        return "", err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
    req.Header.Set("Authorization", "Basic "+auth)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    var token spotifyToken
    if err := json.Unmarshal(body, &token); err != nil {
        return "", err
    }
	fmt.Println(token.AccessToken)
    return token.AccessToken, nil
}
