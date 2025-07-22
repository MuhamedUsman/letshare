#!/bin/bash

# Remove capabilities if they were set
if command -v setcap >/dev/null 2>&1; then
    setcap -r /usr/bin/Letshare 2>/dev/null || true
fi

echo "Uninstalling Letshare..."