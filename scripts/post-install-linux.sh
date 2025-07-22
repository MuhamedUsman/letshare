#!/bin/bash

# Set capabilities to allow binding to port 80 without root
if command -v setcap >/dev/null 2>&1; then
    echo "Setting capabilities for port 80 binding..."
    setcap 'cap_net_bind_service=+ep' /usr/bin/Letshare || {
        echo "Warning: Could not set capabilities. You may need to run as root to bind to port 80."
    }
fi

# Ensure avahi-daemon is running
if command -v systemctl >/dev/null 2>&1; then
    systemctl is-enabled avahi-daemon >/dev/null 2>&1 || {
        echo "Enabling avahi-daemon for mDNS functionality..."
        systemctl enable avahi-daemon
    }

    systemctl is-active avahi-daemon >/dev/null 2>&1 || {
        echo "Starting avahi-daemon..."
        systemctl start avahi-daemon
    }
elif command -v service >/dev/null 2>&1; then
    # For systems using traditional init
    service avahi-daemon start 2>/dev/null || true
fi

echo "Installation complete!"
echo "Note: This application uses port 80 and mDNS services."
echo "If you encounter permission issues, you may need to run with elevated privileges."