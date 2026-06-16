# Build stage
FROM docker.io/library/golang:1.24-alpine@sha256:8bee1901f1e530bfb4a7850aa7a479d17ae3a18beb6e09064ed54cfd245b7191 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o knative-test .

# Runtime stage
FROM docker.io/library/alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/knative-test .

# UID 65534 is 'nobody' in Alpine, already exists
USER nobody

EXPOSE 8080

ENTRYPOINT ["./knative-test"]
