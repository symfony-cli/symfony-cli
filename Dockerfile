FROM scratch as build

COPY --from=composer:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY symfony /usr/local/bin/

FROM scratch

ENV SYMFONY_ALLOW_ALL_IP=true

ENTRYPOINT ["/usr/local/bin/symfony"]

COPY --from=build . .
