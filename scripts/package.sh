#!/usr/bin/env bash
# Turns the output of `wails build` (build/bin) into the files users download.
#
#   scripts/package.sh <linux|darwin|windows> <version> <outdir>
#
# linux    open-ldap-studio_<v>_linux-amd64.deb (dpkg-deb) and .tar.gz
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

    # Built with dpkg-deb, which every Debian-based machine and runner already
    # has, from a staging tree laid out like the target filesystem. The webview
    # comes from the system, so the dependencies are what the Wails build links
    # against (-tags webkit2_41). Modes are set explicitly rather than inherited
    # from the checkout or the umask, and --root-owner-group records root
    # ownership without needing to be root.
    stage=$(mktemp -d)
    pkg="$stage/pkg"
    for d in DEBIAN usr usr/bin usr/share usr/share/applications usr/share/doc \
             usr/share/doc/open-ldap-studio usr/share/icons usr/share/icons/hicolor \
             usr/share/icons/hicolor/512x512 usr/share/icons/hicolor/512x512/apps; do
      install -d -m 0755 "$pkg/$d"
    done
    install -m 0755 "$exe" "$pkg/usr/bin/open-ldap-studio"
    install -m 0644 "$root/build/linux/open-ldap-studio.desktop" \
      "$pkg/usr/share/applications/open-ldap-studio.desktop"
    install -m 0644 "$root/build/appicon.png" \
      "$pkg/usr/share/icons/hicolor/512x512/apps/open-ldap-studio.png"
    install -m 0644 "$root/LICENSE" "$pkg/usr/share/doc/open-ldap-studio/copyright"

    # Installed-Size is in KiB and counts everything except the control files.
    size=$(du -sk "$pkg" | cut -f1)
    sed -e "s/@VERSION@/${version}/" -e "s/@INSTALLED_SIZE@/${size}/" \
      "$root/build/linux/control" > "$pkg/DEBIAN/control"

    dpkg-deb --root-owner-group --build "$pkg" \
      "$out/open-ldap-studio_${version}_linux-amd64.deb"
    rm -rf "$stage"

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
