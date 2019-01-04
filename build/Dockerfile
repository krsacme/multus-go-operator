FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ADD build/_output/bin/multus-go-operator /usr/local/bin/multus-go-operator
