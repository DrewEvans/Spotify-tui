package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Client struct {
    ID                   int64
    ClientID             string
    ClientSecret         string
    SpotifyToken         string
    SpotifyRefreshToken  string
    SpotifyTokenExpiry   string
}

type Store struct {
	conn *sql.DB
}

func (s *Store) Init() error{
	var err error 
	s.conn, err = sql.Open("sqlite3", "./Termify.db")
	if err != nil {
		return err
	}

	createTableStmt := `CREATE TABLE IF NOT EXISTS user (
    user_id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id TEXT NOT NULL,
    client_secret TEXT NOT NULL,
    spotify_token TEXT,
    spotify_refresh_token TEXT,
    spotify_token_expiry TEXT
);`

	if _, err := s.conn.Exec(createTableStmt); err != nil {
		return err
	}

	return nil
}

func (s *Store) GetUser() (*Client, error) {
    row := s.conn.QueryRow("SELECT user_id, client_id, client_secret, spotify_token, spotify_refresh_token, spotify_token_expiry FROM user LIMIT 1")
    user := &Client{}
    err := row.Scan(&user.ID, &user.ClientID, &user.ClientSecret, &user.SpotifyToken, &user.SpotifyRefreshToken, &user.SpotifyTokenExpiry)
    if err == sql.ErrNoRows {
        return nil, nil // No user found
    }
    if err != nil {
        return nil, err
    }
    return user, nil
}
func (s *Store) SaveUser(user *Client) error {
    _, err := s.conn.Exec(
        `INSERT INTO user (client_id, client_secret, spotify_token, spotify_refresh_token, spotify_token_expiry)
         VALUES (?, ?, ?, ?, ?)`,
        user.ClientID, user.ClientSecret, user.SpotifyToken, user.SpotifyRefreshToken, user.SpotifyTokenExpiry,
    )
    return err
}