// main.go
package main

import (
	"log"
	"net/http"

	"internal_chat_system/handlers"
	"internal_chat_system/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	hub := ws.NewHub()
	go hub.Run()

	r.Post("/chat/send", handlers.SendMessage(hub))
	r.Get("/ws", handlers.HandleWebSocket(hub))

	log.Println("Server started on :8080")
	http.ListenAndServe(":8080", r)
}
