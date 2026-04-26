.PHONY: build run dev clean

BINARY=bin/server
GO=go
MAIN=./cmd/server/

build:
	$(GO) build -o $(BINARY) $(MAIN)

run: build
	./$(BINARY)

dev: build
	PORT=3001 AI_PROVIDER=openai AI_MODEL=gpt-4o ./$(BINARY)

clean:
	rm -rf bin/

test:
	$(GO) test ./... -v

deps:
	$(GO) mod tidy

# Cross-compile targets
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY)-linux-amd64 $(MAIN)

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BINARY)-darwin-amd64 $(MAIN)

build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BINARY)-windows-amd64.exe $(MAIN)
