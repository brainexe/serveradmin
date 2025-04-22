
run:
	  go run main.go

build:
	  go build -o main .

test:
	  go test ./...

test-coverage:
	  go test -v ./... -coverprofile=coverage.out
	  go tool cover -html=coverage.out -o coverage.html

linter:
	  golangci-lint run --enable-all
