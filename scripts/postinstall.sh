#!/bin/bash

systemctl --system daemon-reload >/dev/null || true
systemctl enable mysql-tarantool-replicator.service >/dev/null || true

deb_systemctl=$(command -v deb-systemd-invoke || echo systemctl)
${deb_systemctl} restart mysql-tarantool-replicator.service >/dev/null || true