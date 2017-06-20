FROM alpine:3.4
RUN apk add --update dnsmasq && rm -rf /var/cache/apk/*
COPY dnsmasq.conf /etc/dnsmasq.conf
EXPOSE 53/tcp 53/udp
