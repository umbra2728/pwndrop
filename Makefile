TARGET := pwndrop
BUILD_DIR := ./build
COMPOSE := docker compose

.PHONY: all build clean test init up down logs config-check

all: build

build:
	@echo "*** building..."
	@mkdir -p $(BUILD_DIR)
	@env CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(TARGET) ./main.go
	@rm -rf $(BUILD_DIR)/admin
	@echo "*** copying admin panel"
	@cp -r ./www $(BUILD_DIR)/admin
	@chmod 700 $(BUILD_DIR)/$(TARGET)

test:
	@go test ./...

init:
	@./scripts/init.sh

up:
	@$(COMPOSE) up -d --build

down:
	@$(COMPOSE) down

logs:
	@$(COMPOSE) logs -f --tail=100

config-check:
	@$(COMPOSE) config

clean:
	@go clean
	@rm -rf $(BUILD_DIR)
