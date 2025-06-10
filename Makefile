

default: help

help:
	@echo 'coverage: make cover'
	@echo ''


cover:
	go test -cover ./v1/pool/...

cover_html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

run:
	go run main.go

.PHONY: run