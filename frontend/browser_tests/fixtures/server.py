"""Build and manage the Vite preview server once for the entire pytest run."""

import os
import shutil
import signal
import subprocess
import time
from pathlib import Path
from urllib.error import URLError
from urllib.request import urlopen

FRONTEND = Path(__file__).resolve().parents[2]
BASE_URL = "http://127.0.0.1:4173"


def is_ready() -> bool:
    try:
        with urlopen(BASE_URL, timeout=1) as response:
            return response.status == 200
    except (OSError, URLError):
        return False


class PreviewServer:
    def __init__(self) -> None:
        self.process: subprocess.Popen | None = None
        self.log_path = FRONTEND / "playwright-report" / "preview-server.log"

    def start(self) -> None:
        if is_ready():
            raise RuntimeError(f"Stop the server at {BASE_URL} before running frontend browser checks")

        npm = shutil.which("npm.cmd" if os.name == "nt" else "npm")
        if npm is None:
            raise RuntimeError("Node.js and npm are required to build the frontend")

        deadline = time.monotonic() + 120
        self.log_path.parent.mkdir(parents=True, exist_ok=True)
        with self.log_path.open("w", encoding="utf-8") as log:
            subprocess.run(
                [npm, "run", "build"],
                cwd=FRONTEND,
                stdout=log,
                stderr=subprocess.STDOUT,
                timeout=120,
                check=True,
            )
            self.process = subprocess.Popen(
                [npm, "run", "preview", "--", "--host", "127.0.0.1", "--port", "4173", "--strictPort"],
                cwd=FRONTEND,
                stdout=log,
                stderr=subprocess.STDOUT,
                start_new_session=os.name != "nt",
            )

        while time.monotonic() < deadline:
            if self.process.poll() is not None:
                raise RuntimeError("The preview server exited before becoming ready")
            if is_ready():
                return
            time.sleep(0.1)
        raise TimeoutError(f"The preview server did not become ready at {BASE_URL} within 120 seconds")

    def stop(self) -> None:
        if self.process is None or self.process.poll() is not None:
            return
        if os.name == "nt":
            subprocess.run(
                ["taskkill", "/PID", str(self.process.pid), "/T", "/F"],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=False,
            )
        else:
            os.killpg(self.process.pid, signal.SIGTERM)
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(self.process.pid, signal.SIGKILL)
                self.process.wait()
