#!/bin/bash

echo "Uninstalling Letshare..."

# Remove capabilities if they were set
if command -v setcap >/dev/null 2>&1; then
    echo "Removing capabilities from Letshare binary..."
    setcap -r /usr/bin/Letshare 2>/dev/null || true

    # Also remove capabilities from letshare if it's a separate file
    if [ -f /usr/bin/letshare ] && [ ! -L /usr/bin/letshare ]; then
        setcap -r /usr/bin/letshare 2>/dev/null || true
    fi
fi

# Remove lowercase symlink on package removal
if [ -L /usr/bin/letshare ]; then
    echo "Removing letshare symlink..."
    rm -f /usr/bin/letshare
elif [ -f /usr/bin/letshare ] && [ ! -L /usr/bin/letshare ]; then
    # If it's a separate file (not a symlink), remove it too
    echo "Removing letshare binary..."
    rm -f /usr/bin/letshare
fi

# Note: We don't stop avahi-daemon as other applications might depend on it
echo "Package removal complete."
echo "Note: avahi-daemon service was left running as other applications may depend on it."