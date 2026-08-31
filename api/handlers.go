package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"fair_backgammon/lobby"
)

func usernameFromCookie(r *http.Request) string {
	c, _ := r.Cookie("user")
	if c == nil {
		return ""
	}
	return strings.TrimSpace(c.Value)
}

func HandleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	name := strings.TrimSpace(body.Username)
	if name == "" || len(name) > 20 {
		http.Error(w, "username 1-20 chars", 400)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		Value:    name,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"username": name})
}

func HandleCreateLobby(hub *lobby.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		user := usernameFromCookie(r)
		if user == "" {
			http.Error(w, "set username first", 401)
			return
		}
		room := hub.Create(user)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"code": room.Code})
	}
}

func HandleJoinLobby(hub *lobby.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}
		user := usernameFromCookie(r)
		if user == "" {
			http.Error(w, "set username first", 401)
			return
		}
		code := strings.ToUpper(strings.TrimPrefix(r.URL.Path, "/api/lobby/"))
		// strip /join suffix
		code = strings.TrimSuffix(code, "/JOIN")
		code = strings.TrimSuffix(code, "/join")
		_, _, ok := hub.Join(code, user)
		if !ok {
			http.Error(w, "room not found or full", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"code": code})
	}
}

func HandleGetLobby(hub *lobby.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.ToUpper(strings.TrimPrefix(r.URL.Path, "/api/lobby/"))
		if code == "" || strings.Contains(code, "/") {
			http.Error(w, "bad code", 400)
			return
		}
		room := hub.Get(code)
		if room == nil {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code":    room.Code,
			"players": room.Players,
			"board":   room.Game.Board,
			"bar":     room.Game.Bar,
			"off":     room.Game.Off,
			"turn":    room.Game.Turn,
			"dice":    room.Game.Dice,
		})
	}
}
