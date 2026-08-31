package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"fair_backgammon/api"
	"fair_backgammon/lobby"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func wsHandler(hub *lobby.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.ToUpper(r.URL.Query().Get("code"))
		if code == "" {
			http.Error(w, "missing code", 400)
			return
		}
		room := hub.Get(code)
		if room == nil {
			http.Error(w, "room not found", 404)
			return
		}
		c, _ := r.Cookie("user")
		if c == nil || strings.TrimSpace(c.Value) == "" {
			http.Error(w, "set username first", 401)
			return
		}
		user := strings.TrimSpace(c.Value)
		idx := room.PlayerIndex(user)
		if idx == -1 {
			http.Error(w, "not in room", 403)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		ch := make(chan []byte, 16)
		room.AddSub(ch)
		defer room.RemoveSub(ch)

		// send initial state
		room.BroadcastState()

		// writer loop
		go func() {
			for msg := range ch {
				_ = conn.WriteMessage(websocket.TextMessage, msg)
			}
		}()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var msg struct {
				T    string `json:"t"`
				From *int   `json:"from"`
				To   *int   `json:"to"`
				Die  *int   `json:"die"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				b, _ := json.Marshal(map[string]string{"t": "error", "msg": "bad json"})
				select {
				case ch <- b:
				default:
				}
				continue
			}
			room.GameTurn(conn, room, user, idx, msg, ch)
		}
	}
}

func sendErr(conn *websocket.Conn, s string) {
	b, _ := json.Marshal(map[string]string{"t": "error", "msg": s})
	_ = conn.WriteMessage(websocket.TextMessage, b)
}
//go:embed web/dist
var dist embed.FS

func main() {
	hub := lobby.NewHub()

	http.HandleFunc("/api/session", api.HandleSession)
	http.HandleFunc("/api/lobby", api.HandleCreateLobby(hub))
	http.HandleFunc("/api/lobby/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(strings.ToLower(r.URL.Path), "/join") {
			api.HandleJoinLobby(hub)(w, r)
			return
		}
		api.HandleGetLobby(hub)(w, r)
	})
	http.HandleFunc("/ws", wsHandler(hub))

	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	})

	// serve frontend — ponytail: one binary
	sub, _ := fs.Sub(dist, "web/dist")
	// SPA fallback: try file, else index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}
		// try exact file
		if _, err := fs.Stat(sub, strings.TrimPrefix(r.URL.Path, "/")); err == nil {
			http.FileServer(http.FS(sub)).ServeHTTP(w, r)
			return
		}
		// fallback index
		b, _ := fs.ReadFile(sub, "index.html")
		w.Header().Set("Content-Type", "text/html")
		w.Write(b)
	})

	log.Println("listening :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
