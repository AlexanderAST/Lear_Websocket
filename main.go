package main

import (
	"context"
	"log"
	"net/http"
)

func setupAPI() {
	ctx := context.Background()
	manager := NewManager(ctx)
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", manager.serveWS)
	http.HandleFunc("/login", manager.loginHandler)
}

func main() {
	setupAPI()
	log.Println("sever started")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
