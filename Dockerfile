# Builder runs natively on the build machine (arm64), cross-compiles for target arch
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder

ARG TARGETARCH
ARG TARGETOS=linux

ENV SERVER_DIR=/openim-server
WORKDIR $SERVER_DIR

COPY . .
RUN go mod download

# Install Mage and build service binaries for the target architecture
RUN go install github.com/magefile/mage@v1.15.0
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH mage build

# Build the launcher binary for the target architecture using go build directly.
# This avoids mage -compile which does not respect GOARCH for cross-compilation.
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /tmp/mage ./scripts/launcher/

# Final image: alpine only, no Go toolchain needed at runtime
FROM alpine:3.19

RUN apk add --no-cache bash

ENV SERVER_DIR=/openim-server
WORKDIR $SERVER_DIR

COPY --from=builder $SERVER_DIR/_output $SERVER_DIR/_output
COPY --from=builder $SERVER_DIR/config $SERVER_DIR/config
COPY --from=builder /tmp/mage /usr/local/bin/mage
COPY --from=builder $SERVER_DIR/start-config.yml $SERVER_DIR/

ENTRYPOINT ["sh", "-c", "mage start && tail -f /dev/null"]
