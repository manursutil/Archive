#!/bin/sh
# Build everything into dist/: 

set -eu
cd "$(dirname "$0")"

missing=0
echo "Build dependencies:"
if command -v go >/dev/null 2>&1; then
  echo "  Go 1.27+: builds the CLI and server"
else
  echo "  Go 1.27+: required to build the CLI and server; install Go and rerun"
  missing=1
fi
if command -v deno >/dev/null 2>&1; then
  echo "  Deno 2.x: builds the embedded UI and desktop app"
else
  echo "  Deno 2.x: required to build the UI and desktop app; install Deno and rerun"
  missing=1
fi
if command -v pdftotext >/dev/null 2>&1; then
  echo "  Poppler: PDF importing is available"
else
  echo "  Poppler (pdftotext): optional; without it, PDF importing is unavailable"
fi
if [ "$(uname -s)" = Linux ]; then
  if command -v zenity >/dev/null 2>&1; then
    echo "  zenity: Linux desktop file picker is available"
  else
    echo "  zenity: optional; without it, the Linux desktop file picker is unavailable"
  fi
fi

if [ "$missing" -ne 0 ]; then
  exit 1
fi

offer_optional() {
  binary=$1
  feature=$2
  brew_package=$3
  apt_package=$4
  command -v "$binary" >/dev/null 2>&1 && return

  if command -v brew >/dev/null 2>&1; then
    installer=brew
    package=$brew_package
    install="brew install $package"
  elif command -v apt-get >/dev/null 2>&1; then
    installer=apt-get
    package=$apt_package
    install="sudo apt-get install -y $package"
  else
    echo "Install $brew_package with Homebrew or $apt_package with apt to enable $feature."
    return
  fi

  if [ -t 0 ]; then
    printf "%s is optional and enables %s. Run '%s'? [y/N] " "$package" "$feature" "$install"
    read -r answer || answer=""
    case "$answer" in
      [yY]*)
        if [ "$installer" = brew ]; then
          brew install "$package" || echo "$package install failed; $feature stays unavailable"
        else
          sudo apt-get install -y "$package" || echo "$package install failed; $feature stays unavailable"
        fi
        ;;
    esac
  else
    echo "To enable $feature, run: $install"
  fi
}

offer_optional pdftotext "PDF importing" poppler poppler-utils
if [ "$(uname -s)" = Linux ]; then
  offer_optional zenity "the Linux desktop file picker" zenity zenity
fi

go generate
go build -o dist/archive
(cd desktop && deno task compile)

echo "built dist/archive and dist/archive-desktop"
