#!/bin/bash

deb_systemctl=$(command -v deb-systemd-invoke || echo systemctl)
${deb_systemctl} stop mymy.service >/dev/null || true

systemctl disable mymy.service >/dev/null || true
systemctl --system daemon-reload >/dev/null || true