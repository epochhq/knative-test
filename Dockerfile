FROM docker.io/library/golang:1.26.10-alpine@sha256:27f2d3b83cfc5803c8078f8e730af0f7c431bf067afaec8f60a36fb763187cba AS dependencies

ENV GOTOOLCHAIN=local GOWORK=off
WORKDIR /app

COPY go.mod ./
RUN test "$(go env GOVERSION)" = "go$(go list -m -f '{{.GoVersion}}')"
RUN go mod download

FROM dependencies AS builder
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o knative-test .

FROM docker.io/library/alpine:3.24.3@sha256:b3413572228e34bd1c9245f5565e6cd127005655e8a7899aef33d8e83255c9e6

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY --from=builder /app/knative-test .

USER nobody
EXPOSE 8080
ENTRYPOINT ["./knative-test"]
