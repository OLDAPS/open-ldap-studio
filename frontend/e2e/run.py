"""Run the same scenarios as a visible walkthrough or in headless parallel workers."""

import os
import sys
from pathlib import Path


def main() -> int:
    os.chdir(Path(__file__).resolve().parents[1])
    try:
        import pytest
    except ModuleNotFoundError:
        raise SystemExit(
            "Install the Python browser test dependencies first:\n"
            "python3 -m pip install -r requirements-e2e.txt"
        ) from None

    args = sys.argv[1:]
    headed = "--headed" in args
    defaults = [
        "--numprocesses", "0" if headed else ("2" if os.environ.get("CI") else "4"),
        "--reruns", "2" if os.environ.get("CI") else "0",
    ]
    if headed:
        defaults += [
            "--slowmo", "350",
            "--timeout", "120",
            "--tracing", "on",
            "--video", "on",
        ]
    if "--frontend-only" in args and not any(arg.startswith("e2e/") for arg in args):
        defaults += ["e2e", "--ignore=e2e/live"]
    return pytest.main([*defaults, *args])


if __name__ == "__main__":
    raise SystemExit(main())
