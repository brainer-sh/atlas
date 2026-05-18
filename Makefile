VERSION := 0.1.0
BINARY  := atlas
CMD     := ./cmd/atlas

LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test install clean

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

test:
	go test ./...

install:
	go install $(LDFLAGS) $(CMD)

clean:
	rm -f $(BINARY)
