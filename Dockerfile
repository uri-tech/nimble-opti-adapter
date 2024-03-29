# Build the manager binary
FROM golang:1.20 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/controller/ internal/controller/
COPY metrics/ metrics/
COPY utils/ utils/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /

# Set runtime labels
LABEL org.label-schema.build-date=$BUILD_DATE \
    org.label-schema.name="nimble-opti-adapter" \
    org.label-schema.description="nimble-opti-adapter is a Kubernetes operator that automates certificate renewal management when using ingress with the annotation $(cert-manager.io/cluster-issuer) for services that require TLS communication." \
    org.label-schema.url="https://github.com/uri-tech/nimble-opti-adapter" \
    org.label-schema.vcs-ref=$VCS_REF \
    org.label-schema.vcs-url="https://github.com/uri-tech/nimble-opti-adapter" \
    org.label-schema.vendor="uri-tech" \
    org.label-schema.version=$VERSION \
    org.label-schema.schema-version="1.0"

COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
