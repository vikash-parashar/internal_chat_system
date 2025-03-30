// main.go
package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"internal_chat_system/handlers"
	"internal_chat_system/redis"
	"internal_chat_system/repository"
	"internal_chat_system/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", "postgres://postgres@localhost:5432/chat_db?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Cannot connect to PostgreSQL:", err)
	}

	redis.Init("localhost:6379", "", 0)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	// r.Use(auth.JWTMiddleware)

	hub := ws.NewHub()
	go hub.Run()

	// Subscribe to active location(s)
	go redis.Subscribe("default-location-id", hub)

	// repo := repository.NewMessageRepo(db)
	// handlers.Init(repo)

	repo := repository.NewMessageRepo(db)
	sessionRepo := repository.NewChatSessionRepo(db)
	handlers.Init(repo, sessionRepo)

	r.Post("/chat/send", wrapJSON(handlers.SendMessage(hub)))
	r.Get("/chat/history", wrapJSON(handlers.GetMessageHistory))
	r.Get("/ws", handlers.HandleWebSocket(hub))
	r.Put("/chat/read", wrapJSON(handlers.MarkMessageAsRead))
	r.Get("/chat/sessions", wrapJSON(handlers.ListChatSessions(repo)))
	r.Get("/chat/search", handlers.SearchMessages(repo))
	r.Get("/admin/chat/sessions", handlers.AdminListSessions(repo))
	r.Put("/admin/chat/messages/delete", handlers.AdminDeleteMessages(repo))

	log.Println("✅ Server started on :8080")
	http.ListenAndServe(":8080", r)
}

// wrapJSON ensures content-type JSON and proper error message format
func wrapJSON(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
	}
}

// writeJSON writes a clean JSON response
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// writeError wraps an error message in a JSON response
func writeError(w http.ResponseWriter, status int, errMsg string) {
	log.Printf("❌ %s", errMsg)
	writeJSON(w, status, map[string]string{"error": errMsg})
}
