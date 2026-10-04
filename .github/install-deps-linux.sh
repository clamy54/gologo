#!/usr/bin/env bash
# Dependances systeme pour compiler GoLogo sous Debian/Ubuntu : un compilateur C,
# les bibliotheques de developpement de Gio (X11/Wayland, EGL/GLES, xkbcommon,
# Vulkan) et ALSA pour le son. Partage par les workflows CI et release.
set -euo pipefail
sudo apt-get update
sudo apt-get install -y --no-install-recommends \
	gcc pkg-config libasound2-dev \
	libxkbcommon-dev libxkbcommon-x11-dev \
	libwayland-dev wayland-protocols \
	libx11-dev libx11-xcb-dev libxcb1-dev libxcursor-dev libxfixes-dev \
	libegl1-mesa-dev libgles2-mesa-dev libvulkan-dev libffi-dev
