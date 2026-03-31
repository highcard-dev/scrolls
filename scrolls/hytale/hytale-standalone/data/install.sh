#!/usr/bin/env bash
set -euo pipefail

# Detect OS
case "$(uname -s)" in
	Darwin) OS="darwin" ;;
	Linux) OS="linux" ;;
	*) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

# Detect architecture
case "$(uname -m)" in
	x86_64|amd64) ARCH="amd64" ;;
	arm64|aarch64) ARCH="arm64" ;;
	*) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

# Build download URL
VERSION="${VERSION:-latest}"
if [[ "$VERSION" == "latest" ]]; then
	URL="https://github.com/highcard-dev/hsm/releases/latest/download/hsm-${OS}-${ARCH}"
else
	[[ "$VERSION" != v* ]] && VERSION="v$VERSION"
	URL="https://github.com/highcard-dev/hsm/releases/download/${VERSION}/hsm-${OS}-${ARCH}"
fi

echo "Downloading hsm ${VERSION} for ${OS}/${ARCH}..."
curl -fL "$URL" -o hsm
chmod +x hsm
echo "Done"
