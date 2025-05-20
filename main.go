package main

import (
	"fmt"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"log"
	"net/http"
)

func main() {
	fmt.Print("\033[H\033[2J") // Clear terminal

	go func() {
		http.HandleFunc("/login", loginHandler)
		http.HandleFunc("/callback", callbackHandler)
		if err := http.ListenAndServe(":8888", nil); err != nil {
			fmt.Println("HTTP server error:", err)
		}
	}()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	store := new(Store)
	if err := store.Init(); err != nil {
		log.Fatalf("unable to init store: %v", err)
	}
	p := bubbletea.NewProgram(model{store: store})
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}