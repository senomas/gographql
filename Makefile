BINARY_NAME=gographql

.PHONY: all test clean

build:
	# GOARCH=amd64 GOOS=darwin go build -o ${BINARY_NAME}-darwin main.go
  # GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}-linux main.go
  GOARCH=amd64 GOOS=window go build -o ${BINARY_NAME}-windows main.go

run:
	GIN_MODE=release go run main.go

test:
	docker-compose up -d postgres
	go clean -testcache
	go test ./... -v -failfast

qtest:
	go test ./test/book/... -v -failfast