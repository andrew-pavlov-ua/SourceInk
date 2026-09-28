FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/sourceink-api ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/sourceink-migrate ./cmd/migrate

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S sourceink \
    && adduser -S -G sourceink sourceink

COPY --from=build /out/sourceink-api /usr/local/bin/sourceink-api
COPY --from=build /out/sourceink-migrate /usr/local/bin/sourceink-migrate
USER sourceink
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
CMD ["/usr/local/bin/sourceink-api"]
