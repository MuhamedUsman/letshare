<div align="center">
  <img width="400" alt="Asset 1@4x" src="https://github.com/user-attachments/assets/0d7dd28f-4f78-4b8e-beaa-a37c06917548" style="pointer-events: none;"/><br><br>
  <a href="https://github.com/MuhamedUsman/letshare/releases"><img src="https://img.shields.io/badge/OS-linux%2C%20windows%2C%20macOS-0078D4" alt="OS"></a>
  <a href="https://github.com/MuhamedUsman/letshare/releases"><img src="https://img.shields.io/github/v/release/MuhamedUsman/letshare" alt="Latest Release"></a>
  <a href="https://github.com/MuhamedUsman/letshare/releases"><img src="https://img.shields.io/github/downloads/MuhamedUsman/letshare/total" alt="downloads"></a>
</div><br>
    
![Letshare](https://github.com/user-attachments/assets/153408e9-a0f9-4e9a-ba76-7ffb2948102b)

## How it works
- Traverse through directories, choose a directory, select its contents.
- It will start a server serving those files.
- Users on the local network can download the hosted files, either from TUI or browser.

## Installation
<details>
  <summary>Windows</summary><br>
  
  ```powershell
  winget install MuhamedUsman.Letshare
  ```
</details>
<details>
  <summary>Linux</summary>
  
  ### Debian/Ubuntu
  ```bash
  # Download the .deb file from releases
  wget https://github.com/[username]/Letshare/releases/latest/download/letshare_[version]_Linux_x86_64.deb

  # Install the package
  sudo dpkg -i letshare_*.deb
  sudo apt-get install -f  # Fix any dependency issues
  ```
  ### Red Hat/Fedora/CentOS
  ```bash
  # Download the .rpm file and install
  sudo rpm -i letshare_*.rpm
  # or
  sudo dnf install letshare_*.rpm  # Fedora
  sudo yum install letshare_*.rpm  # CentOS/RHEL
  ```
  ### Alpine Linux
  ```bash
  # Download the .apk file and install
  sudo apk add --allow-untrusted letshare_*.apk
  ```
  ### Arch Linux
  ```bash
  # Download and install the package
  sudo pacman -U letshare_*.pkg.tar.xz
  ```
</details>
