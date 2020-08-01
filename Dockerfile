FROM alpine:3.9
COPY deploy usr/local/bin/
ENTRYPOINT ["./usr/local/bin/deploy"]