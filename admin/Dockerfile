# Base and builder image will need to be replaced by Fips compliant one
FROM --platform=linux/amd64 mcr.microsoft.com/oss/go/microsoft/golang:1.24-fips-cbl-mariner2.0@sha256:a6a9c72282cc8e689c79b393bf875aa8e02e224e6d8928d7b366dfe62389d072 as builder

WORKDIR /app/admin
COPY . . 
# https://github.com/microsoft/go/tree/microsoft/main/eng/doc/fips#build-option-to-require-fips-mode
ENV CGO_ENABLED=1 GOFLAGS='-tags=requirefips'
RUN go build -o aro-hcp-admin

FROM --platform=linux/amd64 mcr.microsoft.com/cbl-mariner/distroless/base:2.0-nonroot@sha256:ce44fc29db88c9aba8041a50c1abcd19a54f997c2b99a8c513e8ec113261374a
WORKDIR /
COPY --from=builder /app/admin/aro-hcp-admin .
ENTRYPOINT ["/aro-hcp-admin"]
