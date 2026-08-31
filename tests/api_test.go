package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fair_backgammon/api"
	"fair_backgammon/lobby"
)

func TestSession(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/session", strings.NewReader(`{"username":"alice"}`))
	api.HandleSession(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d", w.Code)
	}
	if c := w.Result().Cookies(); len(c) == 0 || c[0].Value != "alice" {
		t.Fatalf("cookie not set %v", c)
	}
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/api/session", strings.NewReader(`{"username":""}`))
	api.HandleSession(w2, r2)
	if w2.Code != 400 {
		t.Fatalf("want 400 for empty")
	}
}

func TestLobbyCreateJoin(t *testing.T) {
	hub := lobby.NewHub()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/lobby", nil)
	api.HandleCreateLobby(hub)(w, r)
	if w.Code != 401 {
		t.Fatalf("want 401 without cookie got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/api/lobby", nil)
	r.AddCookie(&http.Cookie{Name: "user", Value: "alice"})
	api.HandleCreateLobby(hub)(w, r)
	if w.Code != 200 {
		t.Fatalf("create want 200 got %d %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	code := resp["code"]
	if len(code) != 4 {
		t.Fatalf("bad code %s", code)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/lobby/"+code, nil)
	api.HandleGetLobby(hub)(w, r)
	if w.Code != 200 {
		t.Fatalf("get want 200 got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/api/lobby/"+code+"/join", nil)
	r.AddCookie(&http.Cookie{Name: "user", Value: "bob"})
	api.HandleJoinLobby(hub)(w, r)
	if w.Code != 200 {
		t.Fatalf("join want 200 got %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/api/lobby/"+code+"/join", nil)
	r.AddCookie(&http.Cookie{Name: "user", Value: "eve"})
	api.HandleJoinLobby(hub)(w, r)
	if w.Code != 404 {
		t.Fatalf("want 404 full got %d", w.Code)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/lobby/XXXX", nil)
	api.HandleGetLobby(hub)(w, r)
	if w.Code != 404 {
		t.Fatalf("want 404 bad code")
	}
}
