FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk update && apk add --no-cache make git gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build-release

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/bin/dbank /usr/local/bin

EXPOSE 8080
CMD ["/usr/local/bin/dbank", "serve"]
