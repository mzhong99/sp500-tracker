# docker/python/Dockerfile
FROM python:3.12-bookworm
RUN sed -i 's|http://deb.debian.org|https://deb.debian.org|g' /etc/apt/sources.list.d/debian.sources

RUN apt-get update && apt-get install -y \
    git \
    curl \
    vim \
    less \
    ca-certificates \
    postgresql-client \
 && rm -rf /var/lib/apt/lists/*

COPY python/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt

WORKDIR /workspace
