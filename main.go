package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"fair_backgammon/api"
	"fair_backgammon/lobby"
	"fair_backgammon/fly"

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
		room.AddSub(ch, user)
		defer func() {
			room.RemoveSub(ch)
			hub.HandleDisconnect(code, user)
		}()

		// send initial state
		room.BroadcastState()

		// keep idle connections alive (proxies drop quiet sockets after ~55s)
		conn.SetReadLimit(512)
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})
		ping := time.NewTicker(25 * time.Second)
		defer ping.Stop()

		// writer loop
		go func() {
			for {
				select {
				case msg, ok := <-ch:
					if !ok {
						return
					}
					conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
						return
					}
				case <-ping.C:
					conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						return
					}
				}
			}
		}()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var msg struct {
				T      string  `json:"t"`
				From   *int    `json:"from"`
				To     *int    `json:"to"`
				Die    *int    `json:"die"`
				Action *string `json:"action"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				b, _ := json.Marshal(map[string]string{"t": "error", "msg": "bad json"})
				select {
				case ch <- b:
				default:
				}
				continue
			}
			idx := room.PlayerIndex(user)
			if idx == -1 {
				b, _ := json.Marshal(map[string]string{"t": "error", "msg": "not in room"})
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

//go:embed all:web/dist
var dist embed.FS

func main() {
	hub := lobby.NewHub()

	http.HandleFunc("/api/session", api.HandleSession)
	http.HandleFunc("/api/lobby", api.HandleCreateLobby(hub))
	http.HandleFunc("/api/lobby/vs-fly", api.HandleCreateVsFly(hub))
	http.HandleFunc("/api/lobby/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(strings.ToLower(r.URL.Path), "/leave") {
			api.HandleLeaveLobby(hub)(w, r)
			return
		}
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
	sub, err := fs.Sub(dist, "web/dist")
	if err != nil {
		log.Printf("embedded dist not found, will serve from filesystem: %v", err)
	}
	// SPA fallback: try file, else index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws") {
			http.NotFound(w, r)
			return
		}
		// ponytail: hashed vite assets never change — cache hard; entry html stays no-store below
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		// try exact file from embedded (entry point always goes through no-store fallback below)
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" && p != "index.html" && sub != nil {
			if _, err := fs.Stat(sub, p); err == nil {
				http.FileServer(http.FS(sub)).ServeHTTP(w, r)
				return
			}
		}
		// try filesystem (local dev / Heroku post-build)
		if p != "" && p != "index.html" {
			if _, err := os.Stat("web/dist" + r.URL.Path); err == nil {
				http.FileServer(http.Dir("web/dist")).ServeHTTP(w, r)
				return
			}
		}
		// fallback index
		var b []byte
		if sub != nil {
			b, _ = fs.ReadFile(sub, "index.html")
		}
		if len(b) == 0 {
			b, _ = os.ReadFile("web/dist/index.html")
		}
		if len(b) == 0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
    // Play-vs-Fly: in-process bot when fly.json loads, else python subprocess.
    if heads, err := fly.Load(); err != nil {
        log.Printf("fly data unavailable (%v), subprocess fallback", err)
    } else {
        log.Printf("fly in-process ready (variants=%v)", fly.Variants(heads))
        api.FlyLocal = func(hub *lobby.Hub, room *lobby.Room, botname, variant string) {
            fly.Play(hub, room.Code, botname, variant, heads)
        }
    }
    flyBot, flyWeights := os.Getenv("FLY_BOT"), os.Getenv("FLY_W_TRAINED")
    if flyBot != "" {
        flyURL := os.Getenv("FLY_URL")
        if flyURL == "" {
            flyURL = "http://localhost:" + port
        }
        api.FlySpawn = func(code, botname, variant string) error {
            parts := strings.Fields(flyBot)
            args := append(append([]string{}, parts[1:]...), "--join", code, "--name", botname, "--url", flyURL)
if flyWeights != "" {
args = append(args, "--w", flyWeights)
}
            cmd := exec.Command(parts[0], args...)
            if f, err := os.OpenFile("/tmp/musca-fly.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
                cmd.Stdout, cmd.Stderr = f, f
            }
            log.Printf("fly spawn %s variant=%s code=%s", botname, variant, code)
            return cmd.Start()
        }
    }
	log.Println("listening :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
