FROM docker.io/library/golang:1.26.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server ./cmd/server
RUN go build -o worker ./cmd/worker


FROM docker.io/library/python:3.13-slim

WORKDIR /app

RUN python -m venv /opt/venv

COPY processors/dataset/requirements.txt ./processors/dataset/requirements.txt
COPY processors/image/requirements.txt ./processors/image/requirements.txt

RUN /opt/venv/bin/pip install --no-cache-dir \
    -r processors/dataset/requirements.txt \
    -r processors/image/requirements.txt

COPY --from=builder /app/server ./server
COPY --from=builder /app/worker ./worker
COPY --from=builder /app/processors ./processors
COPY --from=builder /app/web ./web

EXPOSE 8082

CMD ["./server"]