FROM golang:1.15-alpine as build-stage

RUN apk --no-cache add \
    g++ \
    git \
    make \
    bash

ARG VERSION
ENV VERSION=${VERSION}

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./
RUN ./scripts/build/build.sh

# Final image.
FROM alpine:latest
RUN apk --no-cache add \
    ca-certificates
COPY --from=build-stage /src/bin/k8s-image-autoproxy /usr/local/bin/k8s-image-autoproxy
ENTRYPOINT ["/usr/local/bin/k8s-image-autoproxy"]