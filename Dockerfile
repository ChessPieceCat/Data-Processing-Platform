FROM docker.io/library/golang:1.26.5 

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server ./cmd/server

EXPOSE 8082

CMD ["./server"]