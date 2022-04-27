BINARY_NAME=agenex

build:
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY_NAME}-amd64-darwin main.go
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY_NAME}-amd64-linux main.go
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY_NAME}-amd64-windows main.go
