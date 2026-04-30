.PHONY: run build test fmt tidy clean

# Application settings
APP_NAME=api
PORT=8080

# Go commands
run:
	go run cmd/api/main.go

build:
	go build -o bin/$(APP_NAME) cmd/api/main.go

test:
	go test -v ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/
