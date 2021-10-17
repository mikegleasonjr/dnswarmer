FROM golang:1.17.2-alpine3.14 as builder

ENV VERSION=0.0.1
RUN go install github.com/mikegleasonjr/dnswarmer@$VERSION

FROM alpine:3.14

RUN apk add --no-cache tzdata

COPY --from=builder /go/bin/dnswarmer /usr/local/bin

ENTRYPOINT [ "dnswarmer" ]
CMD [ "--help" ]
