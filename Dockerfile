FROM alpine:3.4

ENV VERSION 0.1.0
ADD ./bundles/${VERSION}/binary/watchdog /usr/bin

ENTRYPOINT ["/usr/bin/watchdog"]
