package main

import (
    "bufio"
    "bytes"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "math/rand"
    "net/http"
    "net/url"
    "os"
    "strings"
    "time"

    bubbletea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
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

// Msg types for Bubble Tea
type userMsg struct {
    data  User
    err   error
    token string
}

type curlMsg struct {
    data string
    err  error
}

// Bubbletea model
type model struct {
    user        User
    err         error
    state       int
    AccessToken string

    // Add these fields:
    NowPlayingTrack  string
    NowPlayingArtist string
    NowPlayingAlbum  string
}

var (
    appNameStyle        = lipgloss.NewStyle().Padding(0, 1).Bold(true).Foreground(lipgloss.Color("#1DB954"))
    faintStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Faint(true)
    listEnumeratorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).MarginRight(1)
)

// Init kicks off the API call
func (m model) Init() bubbletea.Cmd {
    return func() bubbletea.Msg {
        apiKey := os.Getenv("SPOTIFY_TOKEN")
        if apiKey == "" {
            return userMsg{err: fmt.Errorf("No access token. Please authenticate via /login.")}
        }
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
            m.AccessToken = msg.token
        }
        return m, nil
    case bubbletea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, bubbletea.Quit
        case "p":
            m.state = playerView
            return m, fetchPlaying()
        case "esc":
            m.state = defaultView
            return m, nil
        case "`": // backtick key
            if m.state == defaultView {
                m.err = fmt.Errorf("To refresh your token, visit http://127.0.0.1:8888/login in your browser.")
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
    s := appNameStyle.Render(`
 ______    ___  ____   ___ ___  ____  _____  __ __ 
|      |  /  _]|    \ |   |   ||    ||     ||  |  |
|      | /  [_ |  D  )| _   _ | |  | |   __||  |  |
|_|  |_||    _]|    / |  \_/  | |  | |  |_  |  ~  |
  |  |  |   [_ |    \ |   |   | |  | |   _] |___, |
  |  |  |     ||  .  \|   |   | |  | |  |   |     |
  |__|  |_____||__|\_||___|___||____||__|   |____/ 
                                                   
`) + "\n"

    if m.AccessToken == "" {
        s += "No access token available.\n"
        s += faintStyle.Render("Visit http://127.0.0.1:8888/login in your browser to authenticate.\n")
    }     

    if m.err != nil {
        s += faintStyle.Render(fmt.Sprintf("Error: %v\n", m.err))
    }

    // Default view
    if m.state == defaultView {
        s += faintStyle.Render("Press 'p' to enter player.") + "\n\n"
    }

    // Spotify Player view
    if m.state == playerView {
        if m.NowPlayingTrack != "" {
            s += fmt.Sprintf(
                "\nNow Playing: %s\nArtist: %s\nAlbum: %s\n",
                m.NowPlayingTrack,
                m.NowPlayingArtist,
                m.NowPlayingAlbum,
            )
        } else {
            s += "\nNo track currently playing.\n"
        }
        s += "\n" + listEnumeratorStyle.Render("'\u2190' previous track\n'enter' to play & pause\n '\u2192'skip track")
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
            return userMsg{err: fmt.Errorf("No access token. Please authenticate via /login.")}
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

        if resp.StatusCode == 401 {
            return userMsg{err: fmt.Errorf("token_expired")}
        }

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
        if apiKey == "" {
            return curlMsg{err: fmt.Errorf("No access token. Please authenticate via /login.")}
        }
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

        if resp.StatusCode == 401 {
            return curlMsg{err: fmt.Errorf("token_expired")}
        }

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            return curlMsg{err: err}
        }
        return curlMsg{data: string(body)}
    }
}

func fetchPlaying() bubbletea.Cmd {
    return func() bubbletea.Msg {
        apiKey := os.Getenv("SPOTIFY_TOKEN")
        if apiKey == "" {
            return curlMsg{err: fmt.Errorf("No access token. Please authenticate via /login.")}
        }
        url := "https://api.spotify.com/v1/me/player/currently-playing"

        req, err := http.NewRequest("GET", url, nil)
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

        if resp.StatusCode == 401 {
            return curlMsg{err: fmt.Errorf("token_expired")}
        }

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            return curlMsg{err: err}
        }
        return curlMsg{data: string(body)}
    }
}

// --- OAuth Handlers ---

func loginHandler(w http.ResponseWriter, r *http.Request) {
    clientID := os.Getenv("SPOTIFY_CLIENT_ID")
    redirectURI := "http://127.0.0.1:8888/callback"
    state := generateRandomString(16)
    scope := "user-read-private user-read-email user-read-playback-state user-modify-playback-state"

    params := url.Values{}
    params.Add("response_type", "code")
    params.Add("client_id", clientID)
    params.Add("scope", scope)
    params.Add("redirect_uri", redirectURI)
    params.Add("state", state)

    authURL := "https://accounts.spotify.com/authorize?" + params.Encode()
    http.Redirect(w, r, authURL, http.StatusFound)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    if code == "" {
        http.Error(w, "No code in callback", http.StatusBadRequest)
        return
    }

    clientID := os.Getenv("SPOTIFY_CLIENT_ID")
    clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
    redirectURI := "http://127.0.0.1:8888/callback"

    data := url.Values{}
    data.Set("grant_type", "authorization_code")
    data.Set("code", code)
    data.Set("redirect_uri", redirectURI)

    req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", bytes.NewBufferString(data.Encode()))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    auth := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
    req.Header.Set("Authorization", "Basic "+auth)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    var tokenResp struct {
        AccessToken  string `json:"access_token"`
        RefreshToken string `json:"refresh_token"`
        ExpiresIn    int    `json:"expires_in"`
        TokenType    string `json:"token_type"`
        Scope        string `json:"scope"`
    }
    if err := json.Unmarshal(body, &tokenResp); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Write tokens to .env file
    // Read existing .env lines
    envLines := make(map[string]string)
    file, err := os.Open(".env")
    if err == nil {
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
            line := scanner.Text()
            if strings.Contains(line, "=") {
                parts := strings.SplitN(line, "=", 2)
                envLines[parts[0]] = parts[1]
            }
        }
        file.Close()
    }

    // Update or add token values
    envLines["SPOTIFY_TOKEN"] = tokenResp.AccessToken
    envLines["SPOTIFY_REFRESH_TOKEN"] = tokenResp.RefreshToken

    // Write back all env variables
    var newEnv []string
    for k, v := range envLines {
        newEnv = append(newEnv, fmt.Sprintf("%s=%s", k, v))
    }
    envContent := strings.Join(newEnv, "\n") + "\n"
    if err := ioutil.WriteFile(".env", []byte(envContent), 0600); err != nil {
        http.Error(w, "Failed to write .env: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Also set them in the current process environment
    os.Setenv("SPOTIFY_TOKEN", tokenResp.AccessToken)
    os.Setenv("SPOTIFY_REFRESH_TOKEN", tokenResp.RefreshToken)

    w.Header().Set("Content-Type", "application/json")
    w.Write(body)
}

func generateRandomString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    rand.Seed(time.Now().UnixNano())
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}
