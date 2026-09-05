# Multi-stage Dockerfile: PyInstaller build + hardened Caddy & Socat delivery
FROM python:3.11-slim AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    binutils \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir pyinstaller

WORKDIR /build
COPY hermes_beacon.py .

RUN pyinstaller --onefile --name challenge hermes_beacon.py && \
    cp dist/challenge /build/challenge && \
    cp dist/challenge /build/beacon && \
    cp dist/challenge /build/challenge.exe

# Runtime Stage
FROM python:3.11-slim

ENV CADDY_NO_CAP_SYS_RESOURCE=1

RUN apt-get update && apt-get install -y --no-install-recommends \
    socat \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 ctf

# Copy Caddy from official image
COPY --from=caddy:2-alpine /usr/bin/caddy /tmp/caddy_raw
RUN cat /tmp/caddy_raw > /usr/bin/caddy && \
    rm /tmp/caddy_raw && \
    chmod 755 /usr/bin/caddy

# Setup challenge runtime
WORKDIR /home/ctf
COPY --from=builder /build/challenge ./challenge
RUN chown -R root:ctf /home/ctf && \
    chmod 750 /home/ctf/challenge

# Setup Caddy file delivery
WORKDIR /srv/public
COPY public/index.html ./index.html
COPY --from=builder /build/challenge ./challenge
COPY --from=builder /build/beacon ./beacon
COPY --from=builder /build/challenge.exe ./challenge.exe
COPY hermes_beacon.py ./hermes_beacon.py
COPY Caddyfile /etc/caddy/Caddyfile
RUN chmod -R 755 /srv/public

COPY entrypoint.sh /entrypoint.sh
RUN chmod 755 /entrypoint.sh

EXPOSE 5000 8080

ENTRYPOINT ["/entrypoint.sh"]
