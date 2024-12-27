# First stage. Building a binary
# -----------------------------------------------------------------------------
ARG BRANCH=master
FROM golang:1.23-alpine AS base

ARG VERSION

# Download the source code
RUN apk --no-cache add git
RUN git clone https://github.com/marcelGoerentz/Threadfin.git /src

WORKDIR /src

RUN git checkout ${BRANCH} && git pull \
    && sed -i "s/const Version = \".*\"/const Version = \"${VERSION}\"/" threadfin.go \
    && go mod tidy \
    && go mod vendor

FROM base AS master
RUN go build .

FROM base AS beta
RUN go build -tags beta .

FROM ${BRANCH} AS builder
RUN echo "Build ${BRANCH} version"

# Second stage. Creating an image
# -----------------------------------------------------------------------------
FROM alpine:3.18

ARG BUILD_DATE
ARG VCS_REF
ARG THREADFIN_PORT=34400
ARG THREADFIN_VERSION

LABEL org.label-schema.build-date="${BUILD_DATE}" \
      org.label-schema.name="Threadfin" \
      org.label-schema.description="Dockerized Threadfin" \
      org.label-schema.url="https://hub.docker.com/r/marcelGoerentz/threadfin/" \
      org.label-schema.vcs-ref="${VCS_REF}" \
      org.label-schema.vcs-url="https://github.com/marcelGoerentz/Threadfin" \
      org.label-schema.vendor="Threadfin" \
      org.label-schema.version="${THREADFIN_VERSION}" \
      org.label-schema.schema-version="1.0" \
      DISCORD_URL="https://discord.gg/hrqg9tgcMZ"

ENV THREADFIN_BIN=/home/threadfin/bin
ENV THREADFIN_CONF=/home/threadfin/conf
ENV THREADFIN_HOME=/home/threadfin
ENV THREADFIN_TEMP=/tmp/threadfin
ENV THREADFIN_CACHE=/home/threadfin/cache
ENV THREADFIN_UID=1000
ENV THREADFIN_GID=1000
ENV THREADFIN_USER=threadfin
ENV THREADFIN_DEBUG=0
ENV THREADFIN_PORT=${THREADFIN_PORT}
ENV THREADFIN_LOG=/var/log/threadfin.log

# Add binary to PATH
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$THREADFIN_BIN

# Set working directory
WORKDIR ${THREADFIN_HOME}

# Install needed dependencies and configure environment
RUN apk update && apk upgrade && apk add ca-certificates curl ffmpeg vlc doas tzdata \
  && echo "permit persist :wheel" >> /etc/doas.d/doas.conf \
  && addgroup -S threadfin -g "${THREADFIN_GID}" \
  && adduser threadfin -G threadfin -u "${THREADFIN_UID}" -g "${THREADFIN_GID}" -s /bin/sh -D \
  &&adduser threadfin wheel \
  && echo "threadfin:threadfin" | chpasswd \
  && sed -i 's/geteuid/getppid/' /usr/bin/vlc \
  && mkdir -p ${THREADFIN_BIN} ${THREADFIN_CONF} ${THREADFIN_TEMP} ${THREADFIN_CACHE} \
  && chown -R threadfin:threadfin ${THREADFIN_BIN} ${THREADFIN_CONF} ${THREADFIN_TEMP} ${THREADFIN_CACHE} \
  && chmod -R 755 ${THREADFIN_HOME}

# Set user
USER threadfin

# Copy built binary from builder image
COPY --from=builder /src/threadfin ${THREADFIN_BIN}/

# Configure container volume mappings
VOLUME ${THREADFIN_CONF} ${THREADFIN_TEMP}

EXPOSE ${THREADFIN_PORT}

ENTRYPOINT ["/bin/sh", "-c", "threadfin -port=${THREADFIN_PORT} -config=${THREADFIN_CONF} -debug=${THREADFIN_DEBUG}"]
