BINARY_NAME=agenex

all: gen build
gen:
	go run ./mmap ./mmap/mime.types > mmap.go
	go fmt .
build:
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY_NAME}-amd64-darwin main.go mmap.go
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME}-amd64-linux main.go mmap.go
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY_NAME}-amd64-windows main.go mmap.go
