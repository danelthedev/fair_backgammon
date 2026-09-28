FROM node:18-slim AS webbuild
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-bookworm AS gobuild
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=webbuild /app/web/dist ./web/dist
RUN go build -o /bin/fair_backgammon .

FROM debian:12-slim
RUN apt-get update && apt-get install -y --no-install-recommends wget ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=gobuild /bin/fair_backgammon /usr/local/bin/fair_backgammon
ENV PORT=8080
EXPOSE 8080
CMD ["fair_backgammon"]
