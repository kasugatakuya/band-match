.PHONY: run test sqlc

run:
	go run ./cmd/server

test:
	go vet ./...
	go test ./...

# internal/db/queries/*.sql や migrations を変えたら実行する(Docker が必要)
sqlc:
	docker run --rm -v "$(CURDIR)":/src -w /src sqlc/sqlc:1.29.0 generate
