ARG PLATFORM

FROM --platform=${PLATFORM} mcr.microsoft.com/oss/go/microsoft/golang:1.25.10-1-azurelinux3.0-amd64 AS builder
COPY internal/go.mod internal/go.sum internal/
COPY rhcos-publisher/go.mod rhcos-publisher/go.sum rhcos-publisher/
RUN cd rhcos-publisher && go mod download
WORKDIR /app
COPY . .
ARG ARO_HCP_REVISION
# https://github.com/microsoft/go/tree/microsoft/main/eng/doc/fips#build-option-to-require-fips-mode
# NOTE: starting with go 1.27, GODEBUG=fips140=only can be used at runtime to also reject
# non-FIPS algorithms. See https://github.com/microsoft/go/tree/microsoft/main/eng/doc/fips
ENV CGO_ENABLED=1 GOFIPS140=latest
RUN --mount=type=cache,target=/go/cache \
    GOCACHE=/go/cache \
    make --directory rhcos-publisher build ARO_HCP_REVISION=${ARO_HCP_REVISION}

FROM --platform=${PLATFORM} mcr.microsoft.com/azurelinux/distroless/base:3.0
USER 65532:65532
WORKDIR /
COPY --from=builder /app/rhcos-publisher/rhcos-publisher .
ARG ARO_HCP_REVISION
ARG ARO_HCP_COMMIT_TIME
LABEL vcs-ref="${ARO_HCP_REVISION}"
LABEL vcs-ref-time="${ARO_HCP_COMMIT_TIME}"
LABEL vcs-url="https://github.com/Azure/ARO-HCP"
ENTRYPOINT ["/rhcos-publisher"]
