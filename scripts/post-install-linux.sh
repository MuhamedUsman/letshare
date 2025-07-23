#!/bin/sh
# post-install-linux.sh - Multi-distro compliant postinst script
set -e

# Handle different package manager arguments
case "$1" in
    configure|"")
        # Debian/Ubuntu: $1 is "configure" for postinst
        # Alpine: $1 is empty for post-install
        # Fedora: $1 is "1" for initial install, "2" for upgrade
        ;;
    abort-upgrade|abort-remove|abort-deconfigure)
        # Debian error recovery cases - exit cleanly
        exit 0
        ;;
    1|2)
        # Fedora RPM: 1=install, 2=upgrade
        ;;
    *)
        echo "postinst called with unknown argument \`$1'" >&2
        exit 1
        ;;
esac

# Set capabilities to allow binding to port 80 without root
if command -v setcap >/dev/null 2>&1; then
    if [ -f /usr/bin/letshare ]; then
        echo "Setting capabilities for port 80 binding..."
        if ! setcap 'cap_net_bind_service=+ep' /usr/bin/letshare 2>/dev/null; then
            echo "Warning: Could not set capabilities. You may need to run as root to bind to port 80." >&2
            # Don't fail the installation if setcap fails
        fi
    fi
fi

# Ensure avahi-daemon is available and running (only if package exists)
if command -v systemctl >/dev/null 2>&1; then
    # systemd-based systems (most modern distributions)
    if systemctl list-unit-files avahi-daemon.service >/dev/null 2>&1; then
        if ! systemctl is-enabled avahi-daemon >/dev/null 2>&1; then
            echo "Enabling avahi-daemon for mDNS functionality..."
            systemctl enable avahi-daemon 2>/dev/null || true
        fi

        if ! systemctl is-active avahi-daemon >/dev/null 2>&1; then
            echo "Starting avahi-daemon..."
            systemctl start avahi-daemon 2>/dev/null || true
        fi
    fi
elif command -v rc-service >/dev/null 2>&1; then
    # Alpine Linux OpenRC
    if rc-service avahi-daemon status >/dev/null 2>&1 || [ -f /etc/init.d/avahi-daemon ]; then
        rc-update add avahi-daemon default 2>/dev/null || true
        rc-service avahi-daemon start 2>/dev/null || true
    fi
elif command -v service >/dev/null 2>&1; then
    # Traditional init systems
    if [ -f /etc/init.d/avahi-daemon ]; then
        service avahi-daemon start 2>/dev/null || true
    fi
fi

echo "Installation complete!"
echo "Note: This application uses port 80 and mDNS services."
echo "If you encounter permission issues, you may need to run with elevated privileges."

exit 0