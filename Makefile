check:
	go vet ./...
	go test ./... -count=1 -v
	go test ./... -race -count=1
	go build ./...
	cd web && npm run test
	@echo "backend+frontend ok"

fuzz:
	go test ./tests -fuzz=FuzzLegalMoves -fuzztime=1s
	go test ./tests -fuzz=FuzzFullGame -fuzztime=1s
