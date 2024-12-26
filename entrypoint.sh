#!/bin/sh

# Dynamically set UID and GID
if [ "$(id -u threadfin)" != "$PUID" ] || [ "$(id -g threadfin)" != "$PGID" ]; then
  echo "Updating threadfin user to UID: $PUID, GID: $PGID..."
  # Update GID
  if ! getent group $PGID >/dev/null; then
    groupmod -g $PGID threadfin
  else
    echo "Group with GID $PGID already exists, skipping groupmod."
  fi
  # Update UID
  usermod -u $PUID -g $PGID threadfin
  # Fix permissions
  chown -R threadfin:threadfin /home/threadfin /tmp/threadfin
fi

# Execute the provided command as threadfin
exec gosu threadfin "$@"
