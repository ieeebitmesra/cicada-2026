# Dockerfile for crypto-sphinx-oracle challenge with Caddy delivery
FROM python:3.11-slim

ENV CADDY_NO_CAP_SYS_RESOURCE=1

RUN apt-get update && apt-get install -y --no-install-recommends \
    socat \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 ctf

# Copy Caddy from official image and strip all xattr capabilities
COPY --from=caddy:2-alpine /usr/bin/caddy /tmp/caddy_raw
RUN cat /tmp/caddy_raw > /usr/bin/caddy && \
    rm /tmp/caddy_raw && \
    chmod 755 /usr/bin/caddy

# Setup challenge runtime
WORKDIR /home/ctf
COPY crypto.py ./challenge.py
RUN chown -R ctf:ctf /home/ctf && \
    chmod 755 /home/ctf/challenge.py

# Setup Caddy file delivery
WORKDIR /srv/public
COPY public/index.html ./index.html
COPY crypto.py ./crypto.py
COPY crypto.py ./challenge.py
COPY Caddyfile /etc/caddy/Caddyfile
RUN chmod -R 755 /srv/public

COPY entrypoint.sh /entrypoint.sh
RUN chmod 755 /entrypoint.sh

EXPOSE 5000 8080

ENTRYPOINT ["/entrypoint.sh"]
