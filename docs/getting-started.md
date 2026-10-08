# Getting started

How to download, verify, install, run and remove Open LDAP Studio. This page
covers the installation only. Using the application (connecting to a directory,
browsing, searching) will be added here as those features ship; see
[`MVP.md`](MVP.md) for what is planned and in what order.

- [Which file to download](#which-file-to-download)
- [Verify the download](#verify-the-download)
- [Install](#install): [Linux](#linux) · [macOS](#macos) · [Windows](#windows)
- [Check that it works](#check-that-it-works)
- [Where it keeps its files](#where-it-keeps-its-files)
- [Uninstall](#uninstall)
- [Troubleshooting](#troubleshooting)

Releases are published on the repository's **Releases** page. Each release
carries four files and a checksum list. `<version>` below is the release number,
for example `0.1.0`.

> **The builds are unsigned.** Neither Apple nor Microsoft has vouched for the
> application, so macOS and Windows will warn you the first time you open it.
> The steps below get past that warning. Only do this for files you downloaded
> from the project's own Releases page and verified against `SHA256SUMS`.

## Which file to download

| Your system | File | Notes |
|---|---|---|
| Debian, Ubuntu and derivatives (64-bit Intel/AMD) | `open-ldap-studio_<version>_linux-amd64.deb` | installs the libraries it needs |
| Any other 64-bit Linux | `open-ldap-studio_<version>_linux-amd64.tar.gz` | portable; you provide the libraries |
| macOS (Apple silicon and Intel) | `open-ldap-studio_<version>_darwin-universal.dmg` | one build for both |
| Windows, 64-bit Intel/AMD | `open-ldap-studio_<version>_windows-amd64-setup.exe` | installer |
| (all) | `SHA256SUMS` | checksums for the four files above |

There is no build for 32-bit systems, for ARM Linux or for ARM Windows.

## Verify the download

`SHA256SUMS` lists the expected SHA-256 hash of each file. Download it next to
the installer and compare. If a hash does not match, delete the file and
download it again; do not install it.

**Linux**

```bash
sha256sum -c --ignore-missing SHA256SUMS
```

**macOS**

```bash
shasum -a 256 open-ldap-studio_<version>_darwin-universal.dmg
grep darwin-universal SHA256SUMS
```

The two hashes must be identical.

**Windows (PowerShell)**

```powershell
Get-FileHash .\open-ldap-studio_<version>_windows-amd64-setup.exe -Algorithm SHA256
Select-String windows-amd64 .\SHA256SUMS
```

The hash printed by `Get-FileHash` must equal the one in the second line,
ignoring letter case.

A matching hash tells you the file arrived intact. It does not prove who built
it, because the list comes from the same place as the file.

## Install

### Linux

**Debian package** (tested on Ubuntu 22.04 and 24.04; other Debian-based
distributions that provide `libgtk-3-0` and `libwebkit2gtk-4.1-0` should work
but are untested):

```bash
sudo apt install ./open-ldap-studio_<version>_linux-amd64.deb
```

Keep the `./`: without it `apt` looks for a package of that name instead of the
file. `apt` installs the two libraries the application needs. The application
then appears in your application menu under *Development*, and the command
`open-ldap-studio` starts it.

**Portable archive**: install `libgtk-3-0` and `libwebkit2gtk-4.1-0` from your
distribution first, then:

```bash
tar -xzf open-ldap-studio_<version>_linux-amd64.tar.gz
./open-ldap-studio
```

The archive also contains the licence. Distributions that only ship
WebKitGTK 4.0 (for example Ubuntu 20.04) are not supported.

### macOS

1. Open the `.dmg` and drag `open-ldap-studio` onto the *Applications* shortcut
   in the same window.
2. Eject the disk image.
3. Start the application from *Applications*. macOS will refuse the first time
   because the application is unsigned.

To allow it, either:

- **macOS 15 (Sequoia) or later:** open *System Settings → Privacy & Security*,
  scroll to the message about `open-ldap-studio`, choose **Open Anyway** and
  confirm; or
- **macOS 14 or earlier:** Control-click (right-click) the application in
  *Applications*, choose **Open**, then **Open** again in the dialog; or
- **from a terminal, on any version:** remove the "downloaded from the internet"
  flag, which is what makes Gatekeeper check the application:

  ```bash
  xattr -dr com.apple.quarantine /Applications/open-ldap-studio.app
  ```

You only have to do this once. These are macOS's own steps for unsigned
applications; they have not been tried by the project on a physical Mac.

### Windows

1. Run `open-ldap-studio_<version>_windows-amd64-setup.exe`.
2. Windows SmartScreen may show "Windows protected your PC". Choose **More
   info**, then **Run anyway**.
3. The installer asks for administrator permission because it installs for all
   users, into `C:\Program Files\OLDAPS\Open LDAP Studio`. Follow the pages and
   finish.

The application uses Microsoft's WebView2 runtime, which is part of Windows 11
and current Windows 10. If it is missing, the installer sets it up, which needs
an internet connection. The installer adds a shortcut to the Start menu and to
the desktop.

## Check that it works

The application can report which build it is without opening a window:

**Linux** (installed from the `.deb`):

```bash
open-ldap-studio --version
```

**macOS**:

```bash
/Applications/open-ldap-studio.app/Contents/MacOS/open-ldap-studio --version
```

**Windows (PowerShell).** The application has no console, so write its output
to a file:

```powershell
$exe = "C:\Program Files\OLDAPS\Open LDAP Studio\open-ldap-studio.exe"
Start-Process -FilePath $exe -ArgumentList "--version" -RedirectStandardOutput version.txt -Wait -NoNewWindow
Get-Content version.txt
```

The output looks like this:

```
0.1.0 (c4844f25b396, built 2026-10-08T14:30:22Z)
```

That is the version, the first 12 characters of the source commit, and the UTC
time of the build. A build made from source shows `unknown` for the last two.

## Where it keeps its files

All of it is per user. Nothing is written outside your home directory except by
the installer.

| What | Where |
|---|---|
| Settings and saved connections (profiles, trusted certificates, history) | `open-ldap-studio` inside your user config directory: `~/.config` on Linux (or `$XDG_CONFIG_HOME`), `~/Library/Application Support` on macOS, `%AppData%` on Windows |
| Logs | a `logs` folder inside the directory above; on Linux, if `XDG_STATE_HOME` is set, `$XDG_STATE_HOME/open-ldap-studio/logs` instead |

**Passwords are never written to those files.** They are stored by your
operating system's credential service: the Secret Service (GNOME Keyring,
KWallet, KeePassXC) on Linux, Keychain on macOS, Credential Manager on Windows.
If none is available, the application keeps them in memory for the session only
and says so, which means you are asked again the next time it starts. Log files
are written with passwords and other secrets removed.

## Uninstall

**Linux**

```bash
sudo apt remove open-ldap-studio        # the .deb
rm -r ./open-ldap-studio                # the portable copy: delete what you extracted
```

**macOS**: drag `open-ldap-studio` from *Applications* to the Trash.

**Windows**: *Settings → Apps → Installed apps* (*Apps & features* on
Windows 10), choose **Open LDAP Studio**, then **Uninstall**. This also removes
the two shortcuts.

Uninstalling does not delete your settings, saved connections or logs. To remove
them too, delete the `open-ldap-studio` folder listed under
[Where it keeps its files](#where-it-keeps-its-files). Passwords saved in the
operating system's credential service are removed there (for example in
Keychain Access or Credential Manager).

## Troubleshooting

**`apt install` says the file is not a package, or looks for it online.** Add
`./` in front of the file name, or give the full path.

**`apt` reports that `libwebkit2gtk-4.1-0` cannot be installed.** Your
distribution is too old to provide WebKitGTK 4.1. Use a newer release.

**macOS says the application "is damaged and can't be opened".** That is the
Gatekeeper reaction to an unsigned download. Use the `xattr` command above.

**The hash does not match.** The download was interrupted or altered. Delete it
and download it again. If it still does not match, do not install it.

**You want to report a problem.** Include the output of `--version` and the
most recent log from the `logs` folder. Logs have passwords and other secrets
removed, but read them before sharing.
