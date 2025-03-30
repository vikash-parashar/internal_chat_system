// main.go
package main

import (
	"database/sql"
	"log"
	"net/http"

	"internal_chat_system/handlers"
	"internal_chat_system/middleware/auth"
	"internal_chat_system/redis"
	"internal_chat_system/repository"
	"internal_chat_system/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/chat_db?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	redis.Init("localhost:6379", "", 0)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(auth.JWTMiddleware) // 🔐 Auth middleware

	hub := ws.NewHub()
	go hub.Run()

	// Subscribe to active location(s)
	go redis.Subscribe("default-location-id", hub)

	repo := repository.NewMessageRepo(db)
	handlers.Init(repo)

	r.Post("/chat/send", handlers.SendMessage(hub))
	r.Get("/chat/history", handlers.GetMessageHistory)
	r.Get("/ws", handlers.HandleWebSocket(hub))
	r.Put("/chat/read", handlers.MarkMessageAsRead)

	log.Println("Server started on :8080")
	http.ListenAndServe(":8080", r)
}
