BINARY_NAME=gographql
TEST_PACKAGE=./graph

.PHONY: all test clean

gen:
	go generate ./...

build:
	# GOARCH=amd64 GOOS=darwin go build -o ${BINARY_NAME}-darwin main.go
  # GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}-linux main.go
  GOARCH=amd64 GOOS=window go build -o ${BINARY_NAME}-windows main.go

run:
	docker-compose up -d postgres
	GIN_MODE=release go run .

clean:
	go clean
	go clean -testcache

test:
	docker-compose up -d postgres
	go clean -testcache
	go test ${TEST_PACKAGE} -p 1 -v -failfast
	go clean -testcache
	TEST_DB_POSTGRES="host=localhost user=demo password=password dbname=demo port=5432 sslmode=disable TimeZone=Asia/Jakarta" go test ${TEST_PACKAGE} -v -failfast

db:
	docker-compose up -d postgres
	go clean -testcache
	LOGGER=1 TEST_DB_POSTGRES="host=localhost user=demo password=password dbname=demo port=5432 sslmode=disable TimeZone=Asia/Jakarta" go test ./graph -v -failfast

mock:
	go clean -testcache
	LOGGER=1 go test ${TEST_PACKAGE} -v -failfast
