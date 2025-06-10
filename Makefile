
cover:
	go test -v -race ./...
	go test -cover ./v1/pool/...

cover_html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

all: up

rebuild: down build up

build:
	sudo docker compose build

up:
	sudo docker compose up -d

down:
	sudo docker compose down

clean:
	sudo docker compose down -v

restart:
	sudo docker compose restart

logs:
	sudo docker compose logs -f

ps:
	sudo docker compose ps

.PHONY: all build up down restart rebuild clean logs ps