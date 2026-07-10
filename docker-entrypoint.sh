#!/bin/sh
set -e

# Started without root privileges (e.g. docker run --user): keep the current user
if [ "$(id -u)" != "0" ]; then
    exec threadfin -port="${THREADFIN_PORT}" -config="${THREADFIN_CONF}" -debug="${THREADFIN_DEBUG}"
fi

# Remap the threadfin user to the requested PUID/PGID (linuxserver.io convention).
# Without PUID/PGID the container behaves exactly as before.
if [ -n "${PUID}" ] || [ -n "${PGID}" ]; then
    CURRENT_UID="$(id -u "${THREADFIN_USER}")"
    CURRENT_GID="$(id -g "${THREADFIN_USER}")"
    PUID="${PUID:-${CURRENT_UID}}"
    PGID="${PGID:-${CURRENT_GID}}"

    if [ "${PGID}" != "${CURRENT_GID}" ]; then
        sed -i "s/^\(${THREADFIN_USER}:x\):${CURRENT_GID}:/\1:${PGID}:/" /etc/group
    fi
    if [ "${PUID}" != "${CURRENT_UID}" ] || [ "${PGID}" != "${CURRENT_GID}" ]; then
        sed -i "s/^\(${THREADFIN_USER}:x\):${CURRENT_UID}:${CURRENT_GID}:/\1:${PUID}:${PGID}:/" /etc/passwd
    fi

    # Make sure the working directories are owned by the requested IDs
    for dir in "${THREADFIN_BIN}" "${THREADFIN_CONF}" "${THREADFIN_TEMP}" "${THREADFIN_CACHE}"; do
        if [ -d "${dir}" ] && [ "$(stat -c '%u:%g' "${dir}")" != "${PUID}:${PGID}" ]; then
            chown -R "${PUID}:${PGID}" "${dir}"
        fi
    done
fi

# Drop root privileges and start Threadfin
exec su-exec "${THREADFIN_USER}" threadfin -port="${THREADFIN_PORT}" -config="${THREADFIN_CONF}" -debug="${THREADFIN_DEBUG}"
