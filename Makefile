APP_NAME=skylight-bridge
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

vet:
	go vet ./...

build:
	go build $(LDFLAGS) -o $(APP_NAME)

lint:
	golangci-lint run ./...

test:
	go test ./... -v

clean:
	rm -f $(APP_NAME)

generate-config: build
	./$(APP_NAME) --generate-config $(ARGS)

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(APP_NAME):dev .

docker-run:
	docker run --rm -p 8080:8080 \
		-v $(PWD)/config.yaml:/config/config.yaml:ro \
		-v skylight-bridge-data:/data \
		$(APP_NAME):dev

.PHONY: vet build lint test clean generate-config docker-build docker-run
