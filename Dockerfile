# Global ARG declarations
ARG BRANCH=master

# First stage: Build the binary
FROM golang:1.23-alpine AS base

ARG BRANCH

# Download the source code
RUN apk --no-cache add git \
    && git clone https://github.com/marcelGoerentz/Threadfin.git /src

WORKDIR /src

RUN git checkout ${BRANCH} && git pull \
    && go mod tidy && go mod vendor

FROM base AS master
RUN go build .

FROM base AS beta
RUN go build -tags beta .

FROM ${BRANCH} AS builder
RUN echo "Build ${BRANCH} version"

# Second stage: Create the final image
FROM alpine:3.18

ARG THREADFIN_PORT=34400

LABEL org.label-schema.name="Threadfin" \
      org.label-schema.description="Dockerized Threadfin" \
      org.label-schema.url="https://hub.docker.com/r/marcelGoerentz/threadfin/" \
      org.label-schema.vcs-url="https://github.com/marcelGoerentz/Threadfin" \
      org.label-schema.vendor="Threadfin"

# Environment Variables
ENV THREADFIN_BIN=/home/threadfin/bin
ENV THREADFIN_CONF=/home/threadfin/conf
ENV THREADFIN_HOME=/home/threadfin
ENV THREADFIN_TEMP=/tmp/threadfin
ENV THREADFIN_CACHE=/home/threadfin/cache
ENV THREADFIN_USER=threadfin
ENV THREADFIN_DEBUG=0
ENV THREADFIN_PORT=${THREADFIN_PORT}
ENV THREADFIN_LOG=/var/log/threadfin.log

# Default UID/GID (can be overridden by environment variables)
ENV PUID=31337
ENV PGID=31337

# Add binary to PATH
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$THREADFIN_BIN

# Set working directory
WORKDIR ${THREADFIN_HOME}

# Install dependencies and configure default user/group
RUN apk update && apk upgrade && apk add --no-cache ca-certificates curl ffmpeg vlc doas tzdata shadow \
  \
  # Install gosu for privilege dropping
  && GOSU_VERSION=1.16 \
  && curl -o /usr/local/bin/gosu -fsSL "https://github.com/tianon/gosu/releases/download/${GOSU_VERSION}/gosu-amd64" \
  && curl -o /usr/local/bin/gosu.asc -fsSL "https://github.com/tianon/gosu/releases/download/${GOSU_VERSION}/gosu-amd64.asc" \
  && chmod +x /usr/local/bin/gosu \
  && gosu --version \
  \
  # Configure default user/group
  && echo "permit persist :wheel" >> /etc/doas.d/doas.conf \
  && addgroup -g 31337 threadfin \
  && adduser -u 31337 -G threadfin -s /bin/sh -D threadfin \
  \
  # Prepare home directories
  && mkdir -p ${THREADFIN_BIN} ${THREADFIN_CONF} ${THREADFIN_TEMP} ${THREADFIN_CACHE} \
  && chown -R threadfin:threadfin ${THREADFIN_BIN} ${THREADFIN_CONF} ${THREADFIN_TEMP} ${THREADFIN_CACHE} \
  && chmod -R 755 ${THREADFIN_HOME}

# Copy built binary from builder image
COPY --from=builder /src/threadfin ${THREADFIN_BIN}/

# Script to dynamically set UID/GID
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Configure container volume mappings
VOLUME ${THREADFIN_CONF} ${THREADFIN_TEMP}

EXPOSE ${THREADFIN_PORT}

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sh", "-c", "threadfin -port=$THREADFIN_PORT -config=$THREADFIN_CONF -debug=$THREADFIN_DEBUG"]
