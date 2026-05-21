FROM golang:1.24-bookworm AS builder

WORKDIR /app
COPY . .
RUN go build -o sparkplug-sql .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y postgresql-client && rm -rf /var/lib/apt/lists/*

RUN mkdir /usr/share/sparkplug && mkdir -p /var/nf/sparkplug-sql
COPY --from=builder /app/sparkplug-sql /usr/bin/sparkplug-sql
COPY tables.sql /usr/share/sparkplug/
COPY views.sql /usr/share/sparkplug/
COPY docker-entrypoint.sh /

WORKDIR /var/nf/sparkplug-sql
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["sparkplug-sql"]
