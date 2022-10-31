FROM scratch

COPY symfony /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/symfony"]
