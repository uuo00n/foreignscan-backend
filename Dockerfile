FROM docker.m.daocloud.io/library/golang:1.24-bookworm AS dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download


FROM docker.m.daocloud.io/library/golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/server ./cmd/server


FROM docker.m.daocloud.io/library/alpine:3.20 AS runtime

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /out/server /app/server

RUN mkdir -p /app/public /app/uploads

ENV TZ=Asia/Shanghai
ENV FS_UPLOAD_DIR=/app/uploads

EXPOSE 3000

CMD ["/app/server"]
