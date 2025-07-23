#!/bin/sh
# pre-remove-linux.sh - Multi-distro compliant prerm script
set -e

# Handle different package manager arguments
case "$1" in
    remove|purge|"")
        # Debian/Ubuntu: remove or purge
        # Alpine: empty argument
        ;;
    upgrade|deconfigure)
        # Debian: don't remove capabilities during upgrade
        exit 0
        ;;
    0)
        # Fedora RPM: 0=uninstall, 1=upgrade
        # Only remove capabilities on actual uninstall
        ;;
    1)
        # Fedora RPM: upgrade - don't remove capabilities
        exit 0
        ;;
    failed-upgrade)
        # Debian error case
        exit 0
        ;;
    *)
        echo "prerm called with unknown argument \`$1'" >&2
        exit 1
        ;;
esac

# Remove capabilities if they were set (only on actual removal, not upgrade)
if command -v setcap >/dev/null 2>&1; then
    if [ -f /usr/bin/letshare ]; then
        echo "Removing capabilities from letshare binary..."
        setcap -r /usr/bin/letshare 2>/dev/null || true
    fi
fi

echo "Preparing to uninstall letshare..."

exit 0