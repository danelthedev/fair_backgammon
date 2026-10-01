package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// ponytail: Klipy key stays server-side, frontend hits /api/gifs/* only.

// loadKlipyKey pulls KlipyKey/KLIPY_KEY from local .env when the env var isn't set.
// Hands-off `go run .` / docker-compose: the key just lives in .env at repo root.
var loadKeyOnce sync.Once

func loadKlipyKey() {
	loadKeyOnce.Do(func() {
		if os.Getenv("KLIPY_KEY") != "" || os.Getenv("KlipyKey") != "" {
			return
		}
		b, err := os.ReadFile(".env")
		if err != nil {
			return
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "KlipyKey=") && !strings.HasPrefix(line, "KLIPY_KEY=") {
				continue
			}
			p := strings.SplitN(line, "=", 2)
			if len(p) == 2 {
				os.Setenv(p[0], strings.TrimSpace(p[1]))
			}
		}
	})
}

type gifItem struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Preview string `json:"preview"`
}

var gifCache = struct {
	sync.Mutex
	m map[string]cacheEntry
}{m: make(map[string]cacheEntry)}

type cacheEntry struct {
	exp  time.Time
	body []byte
}

func gifCached(key string, ttl time.Duration, fill func() ([]byte, int, error)) ([]byte, int) {
	gifCache.Lock()
	if e, ok := gifCache.m[key]; ok && time.Now().Before(e.exp) {
		b := e.body
		gifCache.Unlock()
		return b, 200
	}
	gifCache.Unlock()
	b, code, err := fill()
	if err != nil || code != 200 {
		return b, code
	}
	gifCache.Lock()
	gifCache.m[key] = cacheEntry{exp: time.Now().Add(ttl), body: b}
	// ponytail: tiny cache, evict expired on write
	for k, e := range gifCache.m {
		if time.Now().After(e.exp) {
			delete(gifCache.m, k)
		}
		if len(gifCache.m) > 200 {
			break
		}
	}
	gifCache.Unlock()
	return b, code
}

// pickStr returns first non-empty string.
func pickStr(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// normalize Klipy/Tenor-shaped payloads into [{id,url,preview}].
// normalize Klipy payloads ({data:{data:[...]}}) into [{id,url,preview}].
func normalizeGifs(raw []byte) []byte {
	var top map[string]any
	if err := json.Unmarshal(raw, &top); err != nil {
		return raw
	}
	var arr []any
	for _, k := range []string{"data", "results", "gifs", "items"} {
		if a, ok := top[k].([]any); ok {
			arr = a
			break
		}
		if m, ok := top[k].(map[string]any); ok {
			for _, k2 := range []string{"data", "results", "gifs", "items"} {
				if a, ok := m[k2].([]any); ok {
					arr = a
					break
				}
			}
		}
		if arr != nil {
			break
		}
	}
	if arr == nil {
		return raw
	}
	out := make([]gifItem, 0, len(arr))
	for _, it := range arr {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		u := pickGifURL(m)
		if u == "" || !strings.HasPrefix(u, "https://") || len(u) > 500 {
			continue
		}
		id := pickStr(str(m["id"]))
		out = append(out, gifItem{ID: id, URL: u, Preview: pickGifPreview(m, u)})
		if len(out) >= 24 {
			break
		}
	}
	b, _ := json.Marshal(map[string]any{"items": out})
	return b
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func nested(m map[string]any, path ...string) map[string]any {
	cur := m
	for _, p := range path {
		n, ok := cur[p].(map[string]any)
		if !ok {
			return nil
		}
		cur = n
	}
	return cur
}

// fileFormat grabs size.format.url from Klipy's file map (sizes hd..xs, formats gif/webp/jpg/mp4).
func fileFormat(m map[string]any, sizes, formats []string) string {
	f, ok := m["file"].(map[string]any)
	if !ok {
		return ""
	}
	for _, sz := range sizes {
		if fm, ok := f[sz].(map[string]any); ok {
			for _, ft := range formats {
				if g, ok := fm[ft].(map[string]any); ok && str(g["url"]) != "" {
					return str(g["url"])
				}
			}
		}
	}
	return ""
}

func pickGifURL(m map[string]any) string {
	// Klipy v1: file.hd.gif; Tenor-compat: media_formats.gif.url / url / media[n].gif
	if u := fileFormat(m, []string{"hd", "md", "sm", "xs"}, []string{"gif", "webp", "jpg"}); u != "" {
		return u
	}
	u := pickStr(str(m["url"]), str(m["gif"]), str(m["gif_url"]))
	if u != "" {
		return u
	}
	if mf := nested(m, "media_formats"); mf != nil {
		for _, k := range []string{"gif", "mediumgif", "tinygif"} {
			if g, ok := mf[k].(map[string]any); ok && str(g["url"]) != "" {
				return str(g["url"])
			}
		}
	}
	return ""
}

func pickGifPreview(m map[string]any, full string) string {
	// grid thumbs: smallest gif is fine; keep the card URL (hd) for the full send
	if p := fileFormat(m, []string{"sm", "xs", "md"}, []string{"gif", "webp", "jpg"}); p != "" {
		return p
	}
	if mf := nested(m, "media_formats"); mf != nil {
		for _, k := range []string{"nanogif", "tinygif", "preview"} {
			if g, ok := mf[k].(map[string]any); ok && str(g["url"]) != "" {
				return str(g["url"])
			}
		}
	}
	if p := pickStr(str(m["preview"]), str(m["preview_url"])); p != "" {
		return p
	}
	return full
}

func klipyGet(path string, q url.Values) ([]byte, int, error) {
	loadKlipyKey()
	key := strings.TrimSpace(os.Getenv("KLIPY_KEY"))
	if key == "" {
		key = strings.TrimSpace(os.Getenv("KlipyKey"))
	}
	if key == "" {
		return []byte(`{"error":"gif not configured"}`), 501, nil
	}
	// ponytail: key rides in the path — that's the Klipy v1 contract
	u := "https://api.klipy.com/api/v1/" + url.PathEscape(key) + path + "?" + q.Encode()
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return []byte(`{"error":"upstream unavailable"}`), 502, nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != 200 {
		return []byte(`{"error":"upstream error"}`), 502, nil
	}
	return normalizeGifs(raw), 200, nil
}

// HandleGifSearch proxies Klipy GIF search: GET /api/gifs/search?q=hi&limit=12
func HandleGifSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 50 {
		http.Error(w, "q too long", 400)
		return
	}
	limit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if limit == "" {
		limit = "12"
	}
	key := "search:" + strings.ToLower(q) + ":" + limit
	body, code := gifCached(key, 60*time.Second, func() ([]byte, int, error) {
		v := url.Values{}
		v.Set("q", q)
		v.Set("limit", limit)
		return klipyGet("/gifs/search", v)
	})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(code)
	w.Write(body)
}

// HandleGifTrending proxies Klipy trending: GET /api/gifs/trending?limit=12
func HandleGifTrending(w http.ResponseWriter, r *http.Request) {
	limit := strings.TrimSpace(r.URL.Query().Get("limit"))
	if limit == "" {
		limit = "12"
	}
	body, code := gifCached("trending:"+limit, 300*time.Second, func() ([]byte, int, error) {
		v := url.Values{}
		v.Set("limit", limit)
		return klipyGet("/gifs/trending", v)
	})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(code)
	w.Write(body)
}
