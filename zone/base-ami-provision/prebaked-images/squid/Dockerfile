FROM alpine:3.4
RUN apk add --update squid && rm -rf /var/cache/apk/*
ADD squid.conf /etc/squid/squid.conf
ADD whitelist.txt /etc/squid/whitelist.txt
EXPOSE 3128
CMD squid -N -C -n proxy -d1 2>&1
