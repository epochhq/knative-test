# knative-test

A Fibonacci calculator for the EpochCloud Knative scale-to-zero, cold-start and readiness demonstration.

| Method | Path     | Behavior                                                                           |
| ------ | -------- | ---------------------------------------------------------------------------------- |
| GET    | /        | HTML calculator form                                                               |
| POST   | /api/fib | JSON input n from 0 through 10000; integer result string, digit count and duration |
| GET    | /healthz | Status ok and build version                                                        |
| GET    | /readyz  | 503 before readiness, 200 while serving                                            |

For example, POST `{"n":42}` returns n 42, result 267914296, digits 9 and a duration string. Non-POST API calls receive 405; malformed JSON and out-of-range values receive 400. Unknown paths receive 404. The service marks itself unready before its bounded graceful shutdown.

## Local toolchain

Install Nix and direnv, then run `direnv allow`. The shell provides the compiler pinned by go.mod. The root go.mod is the compiler contract; the Nix check and Docker dependency stage compare it with Go's actual version using Go's native module metadata. Shell entry does not install modules.

```bash
go mod download
go -C devtools mod download
```

The isolated devtools module pins Staticcheck and gocyclo. It keeps checker dependencies out of the application's module graph. The selected supported compiler includes the [crypto/x509 fix](https://pkg.go.dev/vuln/GO-2026-6809). This compiler pairs with the eligible analyzer; the next pairing trigger is recorded in tasks/follow-ups.md.

## Checks

After module setup, run the maintained native checks offline:

```bash
export GOPROXY=off GOSUMDB=off
nix flake check --offline --no-update-lock-file
go run -mod=readonly ./tools/check-format
go mod verify
go -C devtools mod verify
go test -mod=readonly -timeout=5s ./...
go vet -mod=readonly ./...
go tool -modfile=devtools/go.mod staticcheck ./...
go tool -modfile=devtools/go.mod gocyclo -over 10 .
```

The formatter bridge makes unformatted Go source a failing check. Tests use in-process handlers and local files/processes. Native analysis tools run through Go's pinned tool directives.

## Packaging

The official Go builder and runtime are pinned by index digest. Production module checksums are copied before dependency download, and a failed download stops the build. Module builds are readonly. Build metadata retains VERSION, COMMIT and BUILD_TIME arguments.

```bash
docker build --target dependencies -t knative-test-dependencies .
docker build --network=none -t knative-test-local \
  --build-arg VERSION="$(cat VERSION)" \
  --build-arg COMMIT="$(git rev-parse HEAD)" .
docker run --rm --read-only --cap-drop=ALL --security-opt=no-new-privileges \
  -p 127.0.0.1:8080:8080 knative-test-local
```

The dependency target prepares the module cache before the offline application build. Each container keeps its unprivileged user and serves PORT (default 8080). Startup failure exits nonzero.

## Dependency updates

Renovate requires 14-day published timestamps for normal updates and vulnerability alerts. The exact Go 1.26.10 crypto/x509 release is the sole compiler age exception. Go module versions, compiler source and builder image update as a group. Supported-family constraints retain the compatible Go 1.26 line until the paired compiler/analyzer change is qualified. Native Nix lock regeneration owns archive hashes. Package-set revisions use the maintained GitHub digest datasource's commit timestamps. The Renovate operator must support native postUpgradeTasks.installTools and allow exactly `nix --extra-experimental-features 'nix-command flakes' flake lock`.

The pinned runtime includes the [Alpine security fixes](https://alpinelinux.org/posts/Alpine-3.21.9-3.22.7-3.23.7-3.24.3-released.html), which remediates the CA compromise and core-package vulnerabilities. Its trust bundle is copied from the pinned official builder, so packaging does not resolve mutable apk packages.

## Deployment purpose

The existing EpochCloud Argo workflow builds the image for its Knative Service. The calculator exercises scale-to-zero, cold starts and readiness behind the platform sidecar. Its existing URL is `https://run-fibonacci-default.epoch.engineering`. Publication and deployment belong to that platform flow.

The existing Knative resource shape remains:

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: fibonacci
  namespace: default
spec:
  template:
    spec:
      containers:
        - image: harbor.epoch.engineering/epochcloud/knative-test:0.1.0
          ports:
            - containerPort: 8080
```
