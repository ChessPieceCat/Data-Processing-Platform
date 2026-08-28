FROM docker.io/library/golang:1.26.5

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        python3 \
        python3-pip \
        python3-venv \
    && rm -rf /var/lib/apt/lists/*

RUN python3 -m venv /opt/venv

RUN /opt/venv/bin/pip install --no-cache-dir \
    -r processors/dataset/requirements.txt \
    -r processors/image/requirements.txt

RUN go build -o server ./cmd/server
RUN go build -o worker ./cmd/worker

EXPOSE 8082

CMD ["./server"]