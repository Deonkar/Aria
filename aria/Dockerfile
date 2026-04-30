FROM golang:1.22-alpine AS build

WORKDIR /app

COPY aria/go.mod ./
RUN go mod download

COPY aria/ ./
RUN go build -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache wget
RUN adduser -D -H -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]
