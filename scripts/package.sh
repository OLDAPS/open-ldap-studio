#!/usr/bin/env bash
# Turns the output of `wails build` (build/bin) into the files users download.
#
#   scripts/package.sh <linux|darwin|windows> <version> <outdir>
#
# linux    open-ldap-studio_<v>_linux-amd64.deb and .tar.gz
# darwin   open-ldap-studio_<v>_darwin-universal.dmg (unsigned)
# windows  open-ldap-studio_<v>_windows-amd64-setup.exe (NSIS; build with -nsis)
#
# Every artefact is unsigned. Signing and notarisation are tracked separately
# (T329/T330, #457/#458).
set -euo pipefail

os="${1:?usage: package.sh <linux|darwin|windows> <version> <outdir>}"
version="${2:?version required}"
out="${3:?outdir required}"

root=$(cd "$(dirname "$0")/.." && pwd)
bin="$root/build/bin"
mkdir -p "$out"
out=$(cd "$out" && pwd)

case "$os" in
  linux)
    exe="$bin/open-ldap-studio"
    [ -x "$exe" ] || { echo "missing $exe; run wails build first" >&2; exit 1; }

    VERSION="$version" BINARY="$exe" \
      nfpm package --packager deb --config "$root/build/linux/nfpm.yaml" \
      --target "$out/open-ldap-studio_${version}_linux-amd64.deb"

    # A portable copy for distributions without dpkg.
    tar -czf "$out/open-ldap-studio_${version}_linux-amd64.tar.gz" \
      -C "$bin" open-ldap-studio -C "$root" LICENSE
    ;;

  darwin)
    app="$bin/open-ldap-studio.app"
    [ -d "$app" ] || { echo "missing $app; run wails build first" >&2; exit 1; }

    stage=$(mktemp -d)
    cp -R "$app" "$stage/"
    ln -s /Applications "$stage/Applications"
    cp "$root/LICENSE" "$stage/LICENSE"
    hdiutil create -volname "Open LDAP Studio" -srcfolder "$stage" -ov -format UDZO \
      "$out/open-ldap-studio_${version}_darwin-universal.dmg"
    rm -rf "$stage"
    ;;

  windows)
    installer="$bin/open-ldap-studio-amd64-installer.exe"
    [ -f "$installer" ] || { echo "missing $installer; build with wails build -nsis" >&2; exit 1; }
    cp "$installer" "$out/open-ldap-studio_${version}_windows-amd64-setup.exe"
    ;;

  *)
    echo "unknown os: $os" >&2
    exit 2
    ;;
esac

ls -la "$out"
