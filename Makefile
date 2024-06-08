run_pre_config: cmd/main.go
	go run cmd/server/main.go 3000 ./public

run_help: cmd/server/main.go
	go run cmd/server/main.go

run: cmd/server/main.go
	go run cmd/server/main.go $(port) $(folder)

build: cmd/server/main.go
	go build -o build/server cmd/server/main.go