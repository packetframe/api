FROM alpine
COPY doqd /usr/bin/api
ENTRYPOINT ["/usr/bin/api"]