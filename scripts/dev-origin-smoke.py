#!/usr/bin/env python3
"""Verify development IPC refuses foreign and missing browser origins.
Run under xvfb-run with the pinned Wails CLI on PATH. Uses an isolated config.
"""
import http.client
import os
from pathlib import Path
import signal
import socket
import subprocess
import tempfile
import time

with socket.socket() as probe:
    probe.bind(("127.0.0.1", 0))
    port = probe.getsockname()[1]
with tempfile.TemporaryDirectory(prefix="termxp-dev-smoke-") as temp:
    shell = Path(temp) / "test-shell"
    shell.write_text("#!/bin/sh\nexec /bin/bash --noprofile --norc\n")
    shell.chmod(0o700)
    env = dict(os.environ, SHELL=str(shell), GDK_BACKEND="x11", WEBKIT_DISABLE_DMABUF_RENDERER="1", XDG_CONFIG_HOME=temp + "/config", XDG_CACHE_HOME=temp + "/cache")
    with (Path(temp) / "dev.log").open("w+") as log:
        process = subprocess.Popen(["wails", "dev", "-tags", "webkit2_41", "-devserver", f"127.0.0.1:{port}", "-s", "-m", "-nosyncgomod", "-skipbindings", "-noreload", "-nogorebuild"], env=env, stdout=log, stderr=log, start_new_session=True)
        try:
            until = time.monotonic() + 50
            ready = False
            cookie = ""
            while time.monotonic() < until:
                if process.poll() is not None:
                    raise AssertionError("Development server exited")
                try:
                    connection = http.client.HTTPConnection("127.0.0.1", port, timeout=1)
                    connection.request("GET", "/")
                    response = connection.getresponse()
                    cookie = (response.getheader("Set-Cookie") or "").split(";", 1)[0]
                    response.read()
                    ready = response.status == 200
                    connection.close()
                    if ready: break
                except OSError:
                    pass
                time.sleep(0.2)
            assert ready, "Development server did not become ready"
            assert cookie, "Development page did not issue its capability cookie"
            for origin in [None, "https://untrusted.invalid", f"http://127.0.0.1:{port}"]:
                headers = {"Upgrade": "websocket", "Connection": "Upgrade", "Sec-WebSocket-Version": "13", "Sec-WebSocket-Key": "dGhlIHNhbXBsZSBub25jZQ=="}
                headers["Cookie"] = cookie
                if origin: headers["Origin"] = origin
                connection = http.client.HTTPConnection("127.0.0.1", port, timeout=3)
                connection.request("GET", "/wails/ipc", headers=headers)
                response = connection.getresponse()
                if origin == f"http://127.0.0.1:{port}":
                    assert response.status == 101, (origin, response.status)
                else:
                    assert response.status == 403, (origin, response.status)
                response.read(); connection.close()
            print("Development IPC smoke passed: authenticated foreign/missing origins refused; same origin accepted")
        except BaseException:
            log.flush(); log.seek(0)
            print(log.read())
            raise
        finally:
            os.killpg(process.pid, signal.SIGTERM)
            try: process.wait(timeout=5)
            except subprocess.TimeoutExpired: os.killpg(process.pid, signal.SIGKILL); process.wait()
