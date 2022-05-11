BINARY_NAME=gographql

.PHONY: all test clean

build:
	# GOARCH=amd64 GOOS=darwin go build -o ${BINARY_NAME}-darwin main.go
  # GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}-linux main.go
  GOARCH=amd64 GOOS=window go build -o ${BINARY_NAME}-windows main.go

run:
	GIN_MODE=release go run main.go

clean:
	go clean
	go clean -testcache

test:
	go clean -testcache
	go test ./... -p 1 -v -failfast

db:
	docker-compose up -d postgres
	go clean -testcache
	LOGGER=1 TEST_DB_POSTGRES="host=localhost user=demo password=password dbname=demo port=5432 sslmode=disable TimeZone=Asia/Jakarta" go test ./... -v -failfast

dummy:
	go test ./db -v -failfast
