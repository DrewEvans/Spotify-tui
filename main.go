package main

import (
	"log"
	"fmt"
	"github.com/joho/godotenv"
	bubbletea "github.com/charmbracelet/bubbletea"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	store := new(Store)
	if err := store.Init(); err != nil {
		log.Fatalf("unable to init store: %v", err)
	}

	p := bubbletea.NewProgram(model{})
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
	}
}