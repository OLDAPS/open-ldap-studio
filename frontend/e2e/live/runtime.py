"""Own the real application and a disposable instance of server/docker-compose.yml.

No replacement bridge, HTTP API, canned responses or injected runtime is used.
Chromium connects to Wails' own development server and websocket dispatcher.
"""

import json
import os
import shutil
import signal
import socket
import subprocess
import sys
import time
from pathlib import Path
from urllib.error import URLError
from urllib.request import urlopen
from uuid import uuid4

FRONTEND = Path(__file__).resolve().parents[2]
ROOT = FRONTEND.parent
REPORT = FRONTEND / "playwright-report"
BASE_DN = "dc=example,dc=org"
ADMIN_DN = f"cn=admin,{BASE_DN}"
ADMIN_PASSWORD = "adminpassword"


def stop_process(process: subprocess.Popen | None) -> None:
    if process is None or process.poll() is not None:
        return
    if os.name == "nt":
        process.terminate()
    else:
        try:
            os.killpg(process.pid, signal.SIGTERM)
        except ProcessLookupError:
            return
    try:
        process.wait(timeout=5)
    except subprocess.TimeoutExpired:
        if os.name == "nt":
            process.kill()
        else:
            os.killpg(process.pid, signal.SIGKILL)
        process.wait(timeout=5)


class LDAPService:
    def __init__(self, project: str | None = None) -> None:
        self.project = project or f"ols-e2e-{uuid4().hex[:12]}"
        self.ports: dict[str, int] = {}
        self.command = [
            "docker", "compose", "--project-name", self.project,
            "--file", str(ROOT / "server/docker-compose.yml"),
            "--file", str(ROOT / "server/docker-compose.e2e.yml"),
        ]
        self.started = False

    def run(self, *args: str, input: str | None = None, timeout: int = 30) -> str:
        result = subprocess.run(
            [*self.command, *args], cwd=ROOT, input=input,
            text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout,
        )
        if result.returncode:
            raise RuntimeError(f"Docker Compose {' '.join(args[:2])} failed:\n{result.stderr}")
        return result.stdout

    def start(self) -> None:
        if shutil.which("docker") is None:
            raise RuntimeError("Docker with Compose 2.24.4+ is required for live E2E tests")
        self.started = True  # Also clean up after a partially failed `up`.
        self.run("up", "--detach", "--wait", "--wait-timeout", "180", "openldap", timeout=240)
        for kind, port in (("ldap", "1389"), ("ldaps", "1636")):
            address = self.run("port", "openldap", port).strip()
            self.ports[kind] = int(address.rsplit(":", 1)[1])

        # Bitnami imports LDIF into a temporary slapd before starting the final
        # listener. A TCP check, or checking just the base DN, can pass too soon.
        expected = sum(
            line.startswith("dn:")
            for path in (ROOT / "server/seed").glob("*.ldif")
            for line in path.read_text(encoding="utf-8").splitlines()
        )
        deadline = time.monotonic() + 60
        stable = 0
        while time.monotonic() < deadline:
            try:
                count = self.search(BASE_DN, scope="sub").count("\ndn:")
                stable = stable + 1 if count == expected else 0
                if stable == 3:
                    return
            except RuntimeError:
                stable = 0
            time.sleep(1)
        raise RuntimeError(f"OpenLDAP did not finish loading all {expected} seed entries")

    def search(self, dn: str, scope: str = "base", attributes: tuple[str, ...] = ("dn",)) -> str:
        return "\n" + self.run(
            "exec", "--no-TTY", "openldap", "ldapsearch", "-LLL", "-o", "ldif-wrap=no",
            "-x", "-H", "ldap://127.0.0.1:1389", "-D", ADMIN_DN, "-w", ADMIN_PASSWORD,
            "-b", dn, "-s", scope, "(objectClass=*)", *attributes,
        )

    def modify(self, ldif: str) -> None:
        self.run(
            "exec", "--no-TTY", "openldap", "ldapmodify", "-x", "-H", "ldap://127.0.0.1:1389",
            "-D", ADMIN_DN, "-w", ADMIN_PASSWORD, input=ldif,
        )

    def delete_tree(self, dn: str) -> None:
        self.run(
            "exec", "--no-TTY", "openldap", "ldapdelete", "-r", "-x",
            "-H", "ldap://127.0.0.1:1389", "-D", ADMIN_DN, "-w", ADMIN_PASSWORD, dn,
        )

    def stop(self) -> None:
        if not self.started:
            return
        try:
            REPORT.mkdir(parents=True, exist_ok=True)
            (REPORT / "ldap.log").write_text(
                self.run("logs", "--no-color"), encoding="utf-8",
            )
        finally:
            self.run("down", "--volumes", "--remove-orphans", timeout=60)
            self.started = False


class LiveSuite:
    def __init__(self) -> None:
        self.ldap = LDAPService()
        self.build_dir = FRONTEND / "test-results" / "application"
        self.binary = self.build_dir / ("open-ldap-studio.exe" if os.name == "nt" else "open-ldap-studio")
        self.log_path = REPORT / "application-build.log"

    def start(self) -> None:
        if sys.platform != "linux":
            raise RuntimeError("Live E2E currently runs on Linux (locally or in the Linux CI job)")
        if not (os.environ.get("DISPLAY") or os.environ.get("WAYLAND_DISPLAY")):
            raise RuntimeError("Wails needs a display. Run headlessly with xvfb-run -a make test-e2e-headless")
        self.ldap.start()
        self.build_dir.mkdir(parents=True, exist_ok=True)
        self.log_path.parent.mkdir(parents=True, exist_ok=True)
        with self.log_path.open("w", encoding="utf-8") as log:
            subprocess.run(
                ["npm", "run", "build"], cwd=FRONTEND, stdout=log,
                stderr=subprocess.STDOUT, check=True, timeout=120,
            )
            subprocess.run(
                ["go", "build", "-tags", "dev,webkit2_41", "-o", str(self.binary), "."],
                cwd=ROOT, stdout=log, stderr=subprocess.STDOUT, check=True, timeout=240,
            )

    def stop(self) -> None:
        self.ldap.stop()

    def worker_data(self) -> dict:
        return {"project": self.ldap.project, "ports": self.ldap.ports, "binary": str(self.binary)}


class NativeApp:
    def __init__(self, binary: Path, data_dir: Path, output_dir: Path) -> None:
        self.binary = binary
        self.data_dir = data_dir
        self.output_dir = output_dir
        self.process: subprocess.Popen | None = None
        self.url = ""
        self.log_path = output_dir / "native-app.log"

    def start(self) -> None:
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
        self.url = f"http://127.0.0.1:{port}"
        self.output_dir.mkdir(parents=True, exist_ok=True)
        environment = {
            **os.environ,
            "XDG_CONFIG_HOME": str(self.data_dir / "config"),
            "XDG_STATE_HOME": str(self.data_dir / "state"),
            # Exercise the app's real session credential fallback without
            # writing test passwords into the developer's desktop keyring.
            "DBUS_SESSION_BUS_ADDRESS": f"unix:path={self.data_dir}/no-session-bus",
        }
        if os.environ.get("DISPLAY"):
            environment["GDK_BACKEND"] = "x11"
        with self.log_path.open("w", encoding="utf-8") as log:
            self.process = subprocess.Popen(
                [str(self.binary), "-assetdir", str(FRONTEND / "dist"),
                 "-devserver", f"127.0.0.1:{port}", "-loglevel", "Error"],
                cwd=ROOT, env=environment, stdout=log, stderr=subprocess.STDOUT,
                start_new_session=True,
            )
        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            if self.process.poll() is not None:
                raise RuntimeError(f"Wails exited during startup:\n{self.log_path.read_text(encoding='utf-8')}")
            try:
                with urlopen(self.url, timeout=1) as response:
                    if response.status == 200:
                        return
            except (OSError, URLError):
                pass
            time.sleep(0.1)
        raise RuntimeError(f"Wails did not start at {self.url}; see {self.log_path}")

    def stop(self) -> None:
        stop_process(self.process)

    def profiles(self) -> list[dict]:
        path = self.data_dir / "config/open-ldap-studio/connections.json"
        if not path.exists():
            return []
        return json.loads(path.read_text(encoding="utf-8"))["data"].get("profiles") or []
