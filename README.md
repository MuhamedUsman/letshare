<br><div align="center">
  <img width="500" src="https://github.com/user-attachments/assets/9c3e3a63-4066-4099-be76-b2fe3139dc2b" alt="Letshare Logo"/><br><br>
  <a href="https://github.com/MuhamedUsman/letshare/releases"><img src="https://img.shields.io/badge/OS-linux%2C%20windows%2C%20macOS-0078D4" alt="OS"></a>
  <a href="https://github.com/MuhamedUsman/letshare/releases"><img src="https://img.shields.io/github/v/release/MuhamedUsman/letshare" alt="Latest Release"></a>
  <a href="https://github.com/MuhamedUsman/letshare/releases"><img src="https://img.shields.io/github/downloads/MuhamedUsman/letshare/total" alt="downloads"></a>
</div><br>

## About
Letshare is a terminal-based file sharing application that creates a local web server and uses mDNS for automatic network discovery. Easily share files and folders across your local network using a simple, intuitive TUI — no complex setup needed.
<br><br>

![Letshare](https://github.com/user-attachments/assets/153408e9-a0f9-4e9a-ba76-7ffb2948102b)

## Who is it for?
Letshare is built for anyone who wants to share files fast, hassle-free, and locally.
- **Developers** — Instantly share your build artifacts or test files with teammates without uploading to the cloud
- **Teachers & Professors** — Distribute course material to students in a lab or classroom, all connected to the same Wi-Fi or LAN.
- **Teams on the same network** — Share resources in offices, co-working spaces, or home labs without dealing with network drives or external servers.

## Features
- Instant file sharing over local network
- Access via IP or .local hostname
- Automatic peer discovery with mDNS
- Intuitive TUI interface
- Runs locally — no internet required
- Cross-platform support (Linux, Windows, macOS)
- Mobile-friendly with QR codes
- Includes Preferences section for customized behaviour
- Built-in download manager with pause, resume, delete options
- Automatically zip directories (with or without compression) before sharing
- Graceful shutdown — the server continues serving active downloads even after the server is shut down

## Requirements
- **Administrator/Root privileges** (required to bind to port `80`)
- Why port `80`, so users don't have to write `:port` when they write the URL

## Installation
<details>
  <summary>Linux</summary>
  
  ### Debian/Ubuntu (amd64)
  ```bash
  # Download the .deb file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/letshare_1.0.0_linux_amd64.deb
  # Install the package with automatic dependency resolution
  sudo apt install ./Letshare_1.0.0_linux_amd64.deb
  ```
  
  ### Debian/Ubuntu (arm64)
  ```bash
  # Download the .deb file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/letshare_1.0.0_linux_arm64.deb
  # Install the package with automatic dependency resolution
  sudo apt install ./Letshare_1.0.0_linux_arm64.deb
  ```
  
  ### Red Hat/Fedora/CentOS (amd64)
  ```bash
  # Download the .rpm file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/letshare_1.0.0_linux_amd64.rpm
  # Install the package with automatic dependency resolution
  sudo dnf install Letshare_1.0.0_linux_amd64.rpm  # Fedora/RHEL 8+
  # or
  sudo yum install Letshare_1.0.0_linux_amd64.rpm  # CentOS/RHEL 7
  ```
  
  ### Red Hat/Fedora/CentOS (arm64)
  ```bash
  # Download the .rpm file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/letshare_1.0.0_linux_arm64.rpm
  # Install the package with automatic dependency resolution
  sudo dnf install Letshare_1.0.0_linux_arm64.rpm  # Fedora/RHEL 8+
  # or
  sudo yum install Letshare_1.0.0_linux_arm64.rpm  # CentOS/RHEL 7
  ```
  
  ### Alpine Linux (amd64)
  ```bash
  # Download the .apk file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/letshare_1.0.0_linux_amd64.apk
  # Install dependencies first, then the package
  sudo apk add avahi avahi-tools
  sudo apk add --allow-untrusted Letshare_1.0.0_linux_amd64.apk
  ```
  
  ### Alpine Linux (arm64)
  ```bash
  # Download the .apk file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/letshare_1.0.0_linux_arm64.apk
  # Install dependencies first, then the package
  sudo apk add avahi avahi-tools
  sudo apk add --allow-untrusted Letshare_1.0.0_linux_arm64.apk
  ```
  
  ### Arch Linux (amd64)
  ```bash
  # Download the package from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/letshare_1.0.0_linux_amd64.pkg.tar.zst
  # Install the package with automatic dependency resolution
  sudo pacman -U Letshare_1.0.0_linux_amd64.pkg.tar.zst
  ```
  
  ### Arch Linux (arm64)
  ```bash
  # Download the package from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/letshare_1.0.0_linux_arm64.pkg.tar.zst
  # Install the package with automatic dependency resolution
  sudo pacman -U Letshare_1.0.0_linux_arm64.pkg.tar.zst
  ```

  ### Manual Binary Installation (All Distributions)
  If you prefer not to use package managers or encounter dependency issues:
  ```bash
  # Install mDNS dependencies manually based on your distribution:
  
  # Debian/Ubuntu:
  sudo apt update && sudo apt install avahi-daemon avahi-utils
  
  # Red Hat/Fedora/CentOS:
  sudo dnf install avahi avahi-tools  # or use yum on older systems
  
  # Alpine:
  sudo apk add avahi avahi-tools
  
  # Arch:
  sudo pacman -S avahi
  
  # Then download and install the binary:
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/letshare_Linux_x86_64.tar.gz
  tar -xzf Letshare_Linux_x86_64.tar.gz
  sudo mv Letshare /usr/local/bin/
  sudo chmod +x /usr/local/bin/Letshare
  ```
</details>

<details>
  <summary>Windows</summary><br>
  
  ```powershell
  winget install MuhamedUsman.Letshare
  ```
</details>

<details>
  <summary>macOS</summary><br>

  ```bash
  # Add the tap (only needed once)
  brew tap MuhamedUsman/homebrew-letshare

  # Install Letshare
  brew install --cask letshare
  ```
</details>

## Quick Start
- Run `letshare` in your terminal
- Navigate to the directory you want to share
- Select files/folders using the TUI
- Share the displayed URL with others on your network
- Access files via TUI or Browser at `http://[instance-name].local` or the IP address

## Caveats
- Older Android devices (pre-Android 12) have problems resolving multicast DNS (.local domains). 
  While newer Android versions support mDNS, network configuration and device-specific implementations 
  may still cause issues. Using IP addresses ensures compatibility across all devices, which is why 
  the QR code feature exists in the app.
- Some download managers (including IDM) may not properly resolve .local domains due to 
  limited mDNS support in their networking implementation. Using direct IP addresses 
  ensures reliable downloads across different client applications.
- If you're connected to a VPN, Letshare may bind to your VPN-assigned IP (e.g., `172.x.x.x`),
  which is not accessible to devices on your local network,
  make sure you're disconnected from the VPN before starting the server.

## Extras
<details>
  <summary>Terminal Size</summary>
  
  - Coloumns: `145`
  - Rows: `35`
</details>

<details>
  <summary>Terminal Font</summary>
  
- Download and Install all the fonts from [Recursive.zip](https://github.com/ryanoasis/nerd-fonts/tree/master/patched-fonts/Recursive#option-1-download-already-patched-font)
- Set the terminal font face to `RecMonoCasual Nerd Font Propo` and font size to `10`
</details>

<details>
  <summary>Terminal Theme (Windows Specific Guide)</summary>
  
- Enable Acrylic Material and set the opacity to 85%
- Add this your windows terminal app `settings.json` file, in the `schemes` array
```json
{
  "background": "#272822",
  "black": "#3E3D32",
  "blue": "#03395C",
  "brightBlack": "#272822",
  "brightBlue": "#66D9EF",
  "brightCyan": "#66D9EF",
  "brightGreen": "#A6E22E",
  "brightPurple": "#AE81FF",
  "brightRed": "#F92672",
  "brightWhite": "#F8F8F2",
  "brightYellow": "#FD971F",
  "cursorColor": "#FFFFFF",
  "cyan": "#66D9EF",
  "foreground": "#F8F8F2",
  "green": "#A6E22E",
  "name": "Monokai",
  "purple": "#AE81FF",
  "red": "#F92672",
  "selectionBackground": "#FFFFFF",
  "white": "#F8F8F2",
  "yellow": "#FFE792"
}
```
</details>

## Feedback and Contributions
I've done my best to build this project thoughtfully, but there's always room for improvement. Your contributions play a vital role in helping it grow and get better.

Feel free to contribute by [submitting an issue](https://github.com/MuhamedUsman/letshare/issues/new), suggesting ideas, or opening a pull request.
Feedback and contributions are always welcome and appreciated!

## License
This product is distributed under [MIT license](https://github.com/MuhamedUsman/letshare/blob/main/LICENSE).<br>
_Free for Commercial and Non-Commercial Use._
