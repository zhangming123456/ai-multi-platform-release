FROM golang:1.26-bookworm AS builder

WORKDIR /src

COPY backend-go/go.mod backend-go/go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY backend-go/ .

RUN CGO_ENABLED=1 go build -o /out/backend main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /out/backend /app/backend
COPY --from=builder /src/conf /app/conf

RUN mkdir -p /app/data

EXPOSE 8000

CMD ["/app/backend"]
