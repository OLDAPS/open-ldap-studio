"""Interrupt one test's TCP sessions while forwarding the real LDAP protocol.

TLS bytes pass through unchanged. This proxy never generates LDAP responses or
restarts the directory shared by other pytest workers.
"""

import select
import socket
import threading


class LDAPProxy:
    def __init__(self, upstream_port: int) -> None:
        self._upstream_port = upstream_port
        self._listener = socket.socket()
        self._listener.bind(("127.0.0.1", 0))
        self._listener.listen()
        self._listener.settimeout(0.1)
        self.port = self._listener.getsockname()[1]
        self._changed = threading.Condition()
        self._pairs: set[tuple[socket.socket, socket.socket]] = set()
        self._threads: list[threading.Thread] = []
        self._connections = 0
        self._stopped = threading.Event()
        self._forwarding = threading.Event()
        self._forwarding.set()
        self._accept_thread = threading.Thread(target=self._accept, daemon=True)

    def __enter__(self) -> "LDAPProxy":
        self._accept_thread.start()
        return self

    def __exit__(self, *_: object) -> None:
        self._stopped.set()
        self._listener.close()
        self.resume()
        self.drop()
        self._accept_thread.join(timeout=2)
        for thread in self._threads:
            thread.join(timeout=2)
        assert not self._accept_thread.is_alive(), "LDAP proxy accept thread did not stop"
        assert not any(thread.is_alive() for thread in self._threads), "LDAP proxy left an open session"

    @property
    def connections(self) -> int:
        with self._changed:
            return self._connections

    def wait_for_connections(self, count: int) -> None:
        with self._changed:
            assert self._changed.wait_for(lambda: self._connections >= count, timeout=5), (
                f"Expected {count} LDAP connections, received {self._connections}"
            )

    def wait_for_idle(self) -> None:
        with self._changed:
            assert self._changed.wait_for(lambda: not self._pairs, timeout=5), (
                "Application did not close its LDAP transport"
            )

    def pause(self) -> None:
        """Accept TCP, but withhold traffic in both directions."""
        self._forwarding.clear()

    def resume(self) -> None:
        self._forwarding.set()

    def drop(self) -> None:
        """Close current sessions; later connection attempts still reach slapd."""
        with self._changed:
            pairs = tuple(self._pairs)
        for pair in pairs:
            self._close_pair(pair)

    def _accept(self) -> None:
        while not self._stopped.is_set():
            try:
                client, _ = self._listener.accept()
            except socket.timeout:
                continue
            except OSError:
                return
            try:
                server = socket.create_connection(("127.0.0.1", self._upstream_port), timeout=1)
            except OSError:
                client.close()
                continue
            client.settimeout(1)
            server.settimeout(1)
            pair = (client, server)
            with self._changed:
                if self._stopped.is_set():
                    client.close()
                    server.close()
                    return
                self._pairs.add(pair)
                self._connections += 1
                for source, destination in (pair, pair[::-1]):
                    thread = threading.Thread(
                        target=self._copy, args=(source, destination, pair), daemon=True,
                    )
                    self._threads.append(thread)
                    thread.start()
                self._changed.notify_all()

    def _copy(self, source: socket.socket, destination: socket.socket,
              pair: tuple[socket.socket, socket.socket]) -> None:
        try:
            while not self._stopped.is_set():
                if not self._forwarding.wait(timeout=0.1):
                    continue
                readable, _, _ = select.select([source], [], [], 0.1)
                if not readable:
                    continue
                data = source.recv(65536)
                if not data:
                    return
                destination.sendall(data)
        except (OSError, ValueError):
            # Resetting and closing sockets is the fault this fixture creates.
            pass
        finally:
            self._close_pair(pair)

    def _close_pair(self, pair: tuple[socket.socket, socket.socket]) -> None:
        for connection in pair:
            try:
                connection.shutdown(socket.SHUT_RDWR)
            except OSError:
                pass
            connection.close()
        with self._changed:
            self._pairs.discard(pair)
            self._changed.notify_all()
