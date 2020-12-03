#!/bin/bash

deb_systemctl=$(command -v deb-systemd-invoke || echo systemctl)
${deb_systemctl} stop mysql-tarantool-replicator.service >/dev/null || true

systemctl disable mysql-tarantool-replicator.service >/dev/null || true
systemctl --system daemon-reload >/dev/null || true