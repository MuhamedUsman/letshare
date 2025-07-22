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
  
  ### Debian/Ubuntu (amd64)
  ```bash
  # Download the .deb file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/Letshare_1.0.0_linux_amd64.deb

  # Install the package
  sudo dpkg -i Letshare_1.0.0_linux_amd64.deb
  sudo apt-get install -f  # Fix any dependency issues
  ```
  
  ### Debian/Ubuntu (arm64)
  ```bash
  # Download the .deb file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/Letshare_1.0.0_linux_arm64.deb

  # Install the package
  sudo dpkg -i Letshare_1.0.0_linux_arm64.deb
  sudo apt-get install -f  # Fix any dependency issues
  ```
  
  ### Red Hat/Fedora/CentOS (amd64)
  ```bash
  # Download the .rpm file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/Letshare_1.0.0_linux_amd64.rpm

  # Install the package
  sudo rpm -i Letshare_1.0.0_linux_amd64.rpm
  # or
  sudo dnf install Letshare_1.0.0_linux_amd64.rpm  # Fedora
  sudo yum install Letshare_1.0.0_linux_amd64.rpm  # CentOS/RHEL
  ```
  
  ### Red Hat/Fedora/CentOS (arm64)
  ```bash
  # Download the .rpm file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/Letshare_1.0.0_linux_arm64.rpm

  # Install the package
  sudo rpm -i Letshare_1.0.0_linux_arm64.rpm
  # or
  sudo dnf install Letshare_1.0.0_linux_arm64.rpm  # Fedora
  sudo yum install Letshare_1.0.0_linux_arm64.rpm  # CentOS/RHEL
  ```
  
  ### Alpine Linux (amd64)
  ```bash
  # Download the .apk file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/Letshare_1.0.0_linux_amd64.apk

  # Install the package
  sudo apk add --allow-untrusted Letshare_1.0.0_linux_amd64.apk
  ```
  
  ### Alpine Linux (arm64)
  ```bash
  # Download the .apk file from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/Letshare_1.0.0_linux_arm64.apk

  # Install the package
  sudo apk add --allow-untrusted Letshare_1.0.0_linux_arm64.apk
  ```
  
  ### Arch Linux (amd64)
  ```bash
  # Download the package from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/Letshare_1.0.0_linux_amd64.pkg.tar.zst

  # Install the package
  sudo pacman -U Letshare_1.0.0_linux_amd64.pkg.tar.zst
  ```
  
  ### Arch Linux (arm64)
  ```bash
  # Download the package from releases
  wget https://github.com/MuhamedUsman/letshare/releases/latest/download/Letshare_1.0.0_linux_arm64.pkg.tar.zst

  # Install the package
  sudo pacman -U Letshare_1.0.0_linux_arm64.pkg.tar.zst
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

## Extras
### Terminal Size
- Coloumns: `145`
- Rows: `35`
### Terminal Theme(Monokai)
Add this your windows terminal app `settings.json` file, in the `schemes` array
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
### Terminal Font
- Download & Install all the fonts from [Recursive.zip](https://github.com/ryanoasis/nerd-fonts/tree/master/patched-fonts/Recursive#option-1-download-already-patched-font).
- Set the terminal font face to `RecMonoCasual Nerd Font Propo`.
