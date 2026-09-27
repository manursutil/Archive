#!/bin/sh
# Build everything into dist/: 

set -eu
cd "$(dirname "$0")"

go generate
go build -o dist/archive
(cd desktop && deno task compile)

echo "built dist/archive and dist/archive-desktop"

# pdftotext is optional 
if ! command -v pdftotext >/dev/null 2>&1; then
  if command -v brew >/dev/null 2>&1; then
    install="brew install poppler"
  elif command -v apt-get >/dev/null 2>&1; then
    install="sudo apt-get install -y poppler-utils"
  else
    install=""
  fi

  if [ -n "$install" ] && [ -t 0 ]; then
    printf "pdftotext not found, needed to add PDFs. Run '%s'? [y/N] " "$install"
    read -r answer || answer=""
    case "$answer" in
      [yY]*) $install || echo "install failed, PDFs stay disabled" ;;
    esac
  else
    echo "note: install poppler (pdftotext) to add PDFs"
  fi
fi
