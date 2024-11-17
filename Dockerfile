# First stage. Building a binary
# -----------------------------------------------------------------------------
FROM golang:1.23-alpine AS builder

ARG BRANCH=main

# Download the source code
RUN apk update && apk upgrade && apk add git
RUN git clone https://github.com/marcelGoerentz/Threadfin.git /src

WORKDIR /src

RUN git checkout $BRANCH && git pull
RUN go mod tidy && go mod vendor
RUN go build threadfin.go

# Second stage. Creating an image
# -----------------------------------------------------------------------------
FROM alpine:latest

ARG BRANCH=main
ARG BUILD_DATE
ARG VCS_REF
ARG THREADFIN_PORT=34400
ARG THREADFIN_VERSION

LABEL org.label-schema.build-date="{$BUILD_DATE}" \
      org.label-schema.name="Threadfin" \
      org.label-schema.description="Dockerized Threadfin" \
      org.label-schema.url="https://hub.docker.com/r/marcelGoerentz/threadfin/" \
      org.label-schema.vcs-ref="{$VCS_REF}" \
      org.label-schema.vcs-url="https://github.com/marcelGoerentz/Threadfin" \
      org.label-schema.vendor="Threadfin" \
      org.label-schema.version="{$THREADFIN_VERSION}" \
      org.label-schema.schema-version="1.0" \
      DISCORD_URL="https://discord.gg/bEPPNP2VG8"

ENV THREADFIN_BIN=/home/threadfin/bin
ENV THREADFIN_CONF=/home/threadfin/conf
ENV THREADFIN_HOME=/home/threadfin
ENV THREADFIN_TEMP=/tmp/threadfin
ENV THREADFIN_CACHE=/home/threadfin/cache
ENV THREADFIN_UID=31337
ENV THREADFIN_GID=31337
ENV THREADFIN_USER=threadfin
ENV THREADFIN_BRANCH=$BRANCH
ENV THREADFIN_DEBUG=0
ENV THREADFIN_PORT=34400
ENV THREADFIN_LOG=/var/log/threadfin.log

# Add binary to PATH
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$THREADFIN_BIN

# Set working directory
WORKDIR $THREADFIN_HOME

#Set variables
RUN DEBIAN_FRONTEND=noninteractive && TZ="America/New_York"

# Install needed packages
RUN apk update && apk upgrade && apk add ca-certificates curl ffmpeg vlc doas tzdata

# Add group wheel to doas
RUN echo "permit persist :wheel" >> /etc/doas.d/doas.conf
RUN whoami

# Add threadfin group and user
RUN addgroup -S threadfin -g "${THREADFIN_GID}" \
&& adduser threadfin -G threadfin -u "${THREADFIN_UID}" -g "${THREADFIN_GID}" -s /bin/sh -D \
&&adduser threadfin wheel \
&& echo "threadfin:threadfin" | chpasswd

# For VLC
RUN sed -i 's/geteuid/getppid/' /usr/bin/vlc

# Set user
USER threadfin

RUN mkdir -p $THREADFIN_BIN

# Copy built binary from builder image
COPY --chown=${THREADFIN_UID} --from=builder [ "/src/threadfin", "${THREADFIN_BIN}/" ]

# Set binary permissions
RUN chmod +rx $THREADFIN_BIN/threadfin && mkdir $THREADFIN_HOME/cache

# Create working directories for Threadfin
RUN mkdir $THREADFIN_CONF && chmod a+rwX $THREADFIN_CONF && mkdir $THREADFIN_TEMP && chmod a+rwX $THREADFIN_TEMP

# Configure container volume mappings
VOLUME $THREADFIN_CONF
VOLUME $THREADFIN_TEMP

EXPOSE $THREADFIN_PORT

ENTRYPOINT ["/bin/sh", "-c", "threadfin -port=${THREADFIN_PORT} -config=${THREADFIN_CONF} -debug=${THREADFIN_DEBUG} -branch=${THREADFIN_BRANCH}"]
