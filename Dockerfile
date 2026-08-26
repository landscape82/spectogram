FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 go build -o /out/spectogram ./cmd/spectogram

FROM python:3.11-alpine

WORKDIR /app

COPY --from=builder /out/spectogram /usr/local/bin/spectogram
COPY web/ ./web/
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

EXPOSE 8000

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["serve"]
