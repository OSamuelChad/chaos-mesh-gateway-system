# ---- Build stage ----
FROM golang:1.22-alpine AS build
ENV CGO_ENABLED=0 GOOS=linux GOTOOLCHAIN=local
WORKDIR /src

# Cache das dependências
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/app .

# ---- Runtime stage ----
FROM alpine:latest
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S -G app app
COPY --from=build /out/app /usr/local/bin/app
USER app
ENV PORT=8082
EXPOSE 8082
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8082/health >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/app"]
