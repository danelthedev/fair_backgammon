# Fair Backgammon

Online backgammon with a twist: standard rules plus optional modifiers
(negative dice, bigger dice, zero face, fair doubles, power-ups, custom
boards, Romanian marț tehnic). One Go binary serves the API and the
React frontend.

## Features

- **Online PvP** — create/join lobbies, live play over WebSocket.
- **Solo vs bots** — Fruit Fly Bot (neural reservoir brain with live activity panel) and Hard Bot (gnubg-based 2-ply search).
- **Hotseat** — two players, one device.
- **Modifiers** — negative dice, bigger dice (up to d20), zero face, fair doubles, re-roll / skip / protection power-ups, custom board layouts, Romanian marț tehnic (on by default, opt-out).
- **Doubling cube** — with accept/reject/redouble flow.
- **Extras** — gif reactions, heavy sounds and visual user customization, match scores with gammon/backgammon multipliers.

## Setup

Prerequisites: Go ≥ 1.26, Node ≥ 18.

```sh
# backend (from repo root)
go mod download
go build -o bin/fair_backgammon .

# frontend
cd web && npm ci && npm run build && cd ..

# run (serves API + embedded frontend on :8080)
PORT=8080 ./bin/fair_backgammon
```

Dev mode (frontend hot-reload + backend separately):

```sh
cd web && npm run dev      # vite, proxies API/WS to the Go server
go run .                   # backend on :8080
```

## Install / deploy

Single binary + single port. Environment:

| Var         | Purpose                                              |
| ----------- | ---------------------------------------------------- |
| `PORT`      | listen port (default `8080`)                         |
| `KLIPY_KEY` | gif search/trending (gifs degrade gracefully w/o it) |

Docker:

```sh
docker build -t fair-backgammon .
docker run -p 8080:8080 -e PORT=8080 fair-backgammon
# or
docker compose up --build
```

Health check: `GET /api/health` → `{"ok":true}`.
