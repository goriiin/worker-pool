FROM golang:1.23-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /app/main ./cmd

FROM alpine:latest AS prod

WORKDIR /app

COPY --from=build /app/main .

COPY ./v1/config/config.yml .

ENTRYPOINT ["./main"]