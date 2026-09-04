# Multi-stage Dockerfile for pwn-maze-oob challenge
FROM ubuntu:22.04 AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    g++ \
    make \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY binaryexpo.cpp .

RUN g++ -O2 -fno-stack-protector -no-pie binaryexpo.cpp -o challenge && \
    cp challenge maze

# Runner stage
FROM ubuntu:22.04

ENV CADDY_NO_CAP_SYS_RESOURCE=1

RUN apt-get update && apt-get install -y --no-install-recommends \
    socat \
    libstdc++6 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 ctf

# Copy Caddy from official image and strip all xattr capabilities via standard file stream
COPY --from=caddy:2-alpine /usr/bin/caddy /tmp/caddy_raw
RUN cat /tmp/caddy_raw > /usr/bin/caddy && \
    rm /tmp/caddy_raw && \
    chmod 755 /usr/bin/caddy

# Setup challenge runtime
WORKDIR /home/ctf
COPY --from=builder /build/challenge .
RUN chown -R root:ctf /home/ctf && \
    chmod 750 /home/ctf/challenge

# Setup Caddy file delivery
WORKDIR /srv/public
COPY public/index.html ./index.html
COPY --from=builder /build/maze ./maze
COPY --from=builder /build/challenge ./challenge
COPY Caddyfile /etc/caddy/Caddyfile
RUN chmod -R 755 /srv/public

COPY entrypoint.sh /entrypoint.sh
RUN chmod 755 /entrypoint.sh

EXPOSE 5000 8080

ENTRYPOINT ["/entrypoint.sh"]
