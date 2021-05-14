FROM golang:1.16.0-alpine as builder

RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

WORKDIR /go/src/github.com/advancedhosting/csi-advancedhosting

COPY . .

ARG TAG
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -ldflags "-w -s -X github.com/advancedhosting/csi-advancedhosting/advancedhosting.version=${TAG}" -o csi-driver ./cmd/driver

FROM alpine:3.13
RUN apk add --no-cache ca-certificates e2fsprogs xfsprogs blkid xfsprogs-extra e2fsprogs-extra

WORKDIR /

COPY --from=builder /go/src/github.com/advancedhosting/csi-advancedhosting/csi-driver .
ENTRYPOINT ["/csi-driver"]