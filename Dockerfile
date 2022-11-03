FROM scratch as build

COPY --from=composer:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY symfony /usr/local/bin/

FROM scratch

ENTRYPOINT ["/usr/local/bin/symfony"]

COPY --from=build . .
