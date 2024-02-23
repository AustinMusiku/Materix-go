FROM golang:1.20-alpine

WORKDIR /app

COPY go.mod .

RUN go mod download

COPY . .

RUN go build -o app ./cmd/api

EXPOSE 4000

RUN chmod +x app

CMD ["./app"]
