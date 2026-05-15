FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /memleak-demo .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /memleak-demo /usr/local/bin/memleak-demo
ENV PORT=8080 \
    LEAK_CHUNK_MB=8 \
    LEAK_INTERVAL_SEC=3
EXPOSE 8080
USER nobody
ENTRYPOINT ["/usr/local/bin/memleak-demo"]
