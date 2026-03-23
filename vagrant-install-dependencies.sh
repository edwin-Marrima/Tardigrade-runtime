
set -e

GO_VERSION="1.25.6"
OS="linux"
ARCH="arm64" # Change to arm64 if you are on an ARM-based system (like a Raspberry Pi or AWS Graviton)

DOWNLOAD_URL="https://go.dev/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
TAR_FILE="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"

echo "Downloading Go ${GO_VERSION}..."
wget -q --show-progress "$DOWNLOAD_URL"

echo "Removing any previous Go installation..."
sudo rm -rf /usr/local/go

echo "Extracting the archive to /usr/local..."
sudo tar -C /usr/local -xzf "$TAR_FILE"

echo "Cleaning up the downloaded archive..."
rm "$TAR_FILE"

GO_PATH_LINE='export PATH=$PATH:/usr/local/go/bin'

for profile in /etc/profile.d/go.sh ~/.profile ~/.bashrc; do
  if ! grep -qF "$GO_PATH_LINE" "$profile" 2>/dev/null; then
    echo "$GO_PATH_LINE" | sudo tee -a "$profile" > /dev/null
  fi
done

export PATH=$PATH:/usr/local/go/bin

echo "========================================="
echo "Go ${GO_VERSION} has been successfully installed!"
echo "PATH has been updated in /etc/profile.d/go.sh, ~/.profile, and ~/.bashrc"
echo "========================================="