FROM docker.io/library/golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS dependencies

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
