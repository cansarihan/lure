FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=none
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
      -o /out/lure ./cmd/lure

FROM alpine:3.20
RUN adduser -D -u 10001 lure && mkdir -p /data && chown lure /data
COPY --from=build /out/lure /usr/local/bin/lure
USER lure
VOLUME /data
# The decoys listen on unprivileged ports; map 22, 23, 80 and 6379 to them on
# the host. The panel stays on its own port.
EXPOSE 8080 2222 2323 6379 9000
ENTRYPOINT ["/usr/local/bin/lure"]
CMD ["run", "-sensor", "http=:8080", "-sensor", "ssh=:2222", "-sensor", "telnet=:2323", "-sensor", "raw:redis=:6379", "-panel", ":9000", "-journal", "/data/events.jsonl"]
