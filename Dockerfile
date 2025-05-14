FROM golang:1.23.4-bookworm

WORKDIR /usr/src/app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o main ./cmd/server

EXPOSE 8080

CMD ["./main"]