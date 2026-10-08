#!/usr/bin/env bash
# Installs a packaged artefact the way a user would and checks that what landed
# on disk starts and reports the version it was built as. The window never
# opens: --version returns first.
#
#   scripts/smoke-installer.sh <linux|darwin|windows> <version> <artefact-dir>
set -euo pipefail

os="${1:?usage: smoke-installer.sh <linux|darwin|windows> <version> <artefact-dir>}"
version="${2:?version required}"
dir="${3:?artefact dir required}"

check() { # check <what> <output>
  echo "$1 says: $2"
  case "$2" in
    "$version "*) echo "$1: version $version confirmed" ;;
    *) echo "$1: expected a line starting with '$version '" >&2; exit 1 ;;
  esac
}

case "$os" in
  linux)
    # apt treats a path without a leading ./ or / as a package name, so give
    # it an absolute one. (Windows below needs the opposite: a relative path
    # that PowerShell can resolve, not the MSYS path realpath would produce.)
    deb=$(realpath "$dir/open-ldap-studio_${version}_linux-amd64.deb")
    sudo apt-get install -y "$deb"
    dpkg -s open-ldap-studio | sed -n '1,3p'
    test -f /usr/share/applications/open-ldap-studio.desktop
    test -f /usr/share/doc/open-ldap-studio/copyright
    check "installed .deb" "$(open-ldap-studio --version)"
    sudo apt-get remove -y open-ldap-studio
    ! command -v open-ldap-studio >/dev/null || { echo "binary still on PATH after removal" >&2; exit 1; }

    tmp=$(mktemp -d)
    tar -xzf "$dir/open-ldap-studio_${version}_linux-amd64.tar.gz" -C "$tmp"
    check "portable tarball" "$("$tmp/open-ldap-studio" --version)"
    ;;

  darwin)
    dmg="$dir/open-ldap-studio_${version}_darwin-universal.dmg"
    mnt=$(mktemp -d)
    hdiutil attach "$dmg" -mountpoint "$mnt" -nobrowse -quiet
    trap 'hdiutil detach "$mnt" -quiet || true' EXIT
    test -L "$mnt/Applications"
    check "app inside the .dmg" "$("$mnt/open-ldap-studio.app/Contents/MacOS/open-ldap-studio" --version)"
    plutil -lint "$mnt/open-ldap-studio.app/Contents/Info.plist"
    ;;

  windows)
    setup="$dir/open-ldap-studio_${version}_windows-amd64-setup.exe"
    # NSIS /S is silent; the installer needs elevation, which a runner has.
    powershell -NoProfile -Command "Start-Process -FilePath '$setup' -ArgumentList '/S' -Wait"
    exe=$(powershell -NoProfile -Command "(Get-ChildItem -Path \$env:ProgramFiles -Recurse -Filter open-ldap-studio.exe -ErrorAction SilentlyContinue | Select-Object -First 1).FullName")
    [ -n "$exe" ] || { echo "installer ran but open-ldap-studio.exe is not under Program Files" >&2; exit 1; }
    echo "installed at: $exe"
    # A GUI-subsystem exe has no console, so read its output from a file.
    powershell -NoProfile -Command "Start-Process -FilePath '$exe' -ArgumentList '--version' -RedirectStandardOutput smoke.txt -Wait -NoNewWindow"
    check "installed .exe" "$(cat smoke.txt)"

    uninstall="$(dirname "$exe")/uninstall.exe"
    powershell -NoProfile -Command "Start-Process -FilePath '$uninstall' -ArgumentList '/S' -Wait"
    ;;

  *)
    echo "unknown os: $os" >&2
    exit 2
    ;;
esac
