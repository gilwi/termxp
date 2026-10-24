#!/usr/bin/env python3
"""Linux packaged-app smoke test. Run under xvfb-run; requires libX11/libXtst.
Uses real keyboard/mouse events and isolated config, never the user's workspace.
"""
import ctypes as c
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time

x = c.CDLL("libX11.so.6")
t = c.CDLL("libXtst.so.6")
x.XOpenDisplay.argtypes = [c.c_char_p]
x.XOpenDisplay.restype = c.c_void_p
x.XDefaultRootWindow.argtypes = [c.c_void_p]
x.XDefaultRootWindow.restype = c.c_ulong
x.XQueryTree.argtypes = [c.c_void_p, c.c_ulong, c.POINTER(c.c_ulong), c.POINTER(c.c_ulong), c.POINTER(c.POINTER(c.c_ulong)), c.POINTER(c.c_uint)]
x.XFetchName.argtypes = [c.c_void_p, c.c_ulong, c.POINTER(c.c_char_p)]
x.XFree.argtypes = [c.c_void_p]
x.XStringToKeysym.argtypes = [c.c_char_p]
x.XStringToKeysym.restype = c.c_ulong
x.XKeysymToKeycode.argtypes = [c.c_void_p, c.c_ulong]
x.XKeysymToKeycode.restype = c.c_uint
x.XSetInputFocus.argtypes = [c.c_void_p, c.c_ulong, c.c_int, c.c_ulong]
x.XRaiseWindow.argtypes = [c.c_void_p, c.c_ulong]
x.XMoveWindow.argtypes = [c.c_void_p, c.c_ulong, c.c_int, c.c_int]
x.XGetGeometry.argtypes = [c.c_void_p, c.c_ulong, c.POINTER(c.c_ulong), c.POINTER(c.c_int), c.POINTER(c.c_int), c.POINTER(c.c_uint), c.POINTER(c.c_uint), c.POINTER(c.c_uint), c.POINTER(c.c_uint)]
x.XFlush.argtypes = [c.c_void_p]
x.XCloseDisplay.argtypes = [c.c_void_p]
t.XTestFakeKeyEvent.argtypes = [c.c_void_p, c.c_uint, c.c_int, c.c_ulong]
t.XTestFakeMotionEvent.argtypes = [c.c_void_p, c.c_int, c.c_int, c.c_int, c.c_ulong]
t.XTestFakeButtonEvent.argtypes = [c.c_void_p, c.c_uint, c.c_int, c.c_ulong]
display = x.XOpenDisplay(None)
if not display:
    raise SystemExit("No X display; run with xvfb-run -a")

def key(name, down):
    code = x.XKeysymToKeycode(display, x.XStringToKeysym(name.encode()))
    assert code, f"No keycode for {name}"
    t.XTestFakeKeyEvent(display, code, down, 0)
    x.XFlush(display)

def chord(*names):
    for name in names:
        key(name, True)
    for name in reversed(names):
        key(name, False)
    time.sleep(0.15)

def type_text(text):
    names = {" ": "space", "\n": "Return", "'": "apostrophe", '"': "quotedbl", "/": "slash", "-": "minus", "_": "underscore", ".": "period", ">": "greater", ";": "semicolon", "=": "equal", "\\": "backslash", ":": "colon", "(": "parenleft", ")": "parenright"}
    for char in text:
        shifted = char.isupper() or char in '_>"(): '
        if char == " ": shifted = False
        if shifted: key("Shift_L", True)
        name = names.get(char, char)
        key(name, True); key(name, False)
        if shifted: key("Shift_L", False)
        time.sleep(0.004)

def windows(window):
    yield window
    root, parent, count = c.c_ulong(), c.c_ulong(), c.c_uint()
    children = c.POINTER(c.c_ulong)()
    if x.XQueryTree(display, window, c.byref(root), c.byref(parent), c.byref(children), c.byref(count)):
        ids = [children[i] for i in range(count.value)]
        if children: x.XFree(children)
        for child in ids:
            yield from windows(child)

def find_window():
    for window in windows(x.XDefaultRootWindow(display)):
        name = c.c_char_p()
        if x.XFetchName(display, window, c.byref(name)) and name.value:
            title = name.value.decode(errors="replace")
            x.XFree(name)
            if "TermXP" in title: return window
    return None

def wait_for(predicate, message, seconds=12):
    until = time.monotonic() + seconds
    while time.monotonic() < until:
        if predicate(): return
        time.sleep(0.05)
    raise AssertionError(message)

binary = str(Path(sys.argv[1] if len(sys.argv) > 1 else "build/bin/termxp").resolve())
with tempfile.TemporaryDirectory(prefix="termxp-smoke-") as temp:
    root = Path(temp)
    shell = root / "test-shell"
    shell.write_text("#!/bin/sh\nexec /bin/bash --noprofile --norc\n")
    shell.chmod(0o700)
    env = dict(os.environ, SHELL=str(shell), XDG_CONFIG_HOME=str(root / "config"), XDG_CACHE_HOME=str(root / "cache"), LIBGL_ALWAYS_SOFTWARE="1", GDK_BACKEND="x11", WEBKIT_DISABLE_DMABUF_RENDERER="1")
    with (root / "app.log").open("w+") as log:
        app = subprocess.Popen([binary], env=env, stdout=log, stderr=log)
        try:
            wait_for(lambda: find_window() is not None, "No TermXP window")
            window = find_window()
            x.XMoveWindow(display, window, 0, 0)
            x.XRaiseWindow(display, window)
            x.XSetInputFocus(display, window, 1, 0)
            x.XFlush(display)
            time.sleep(2)
            marker = root / "first"
            type_text(f"printf first > {marker}\n")
            wait_for(marker.exists, "First shell did not accept input (check bridge/CSP)")
            chord("Control_L", "Shift_L", "e")
            time.sleep(0.5)
            second = root / "second"
            type_text(f"printf second > {second}\n")
            wait_for(second.exists, "Split shell did not accept input")
            type_text("exit 9\n")
            time.sleep(0.4)
            chord("Control_L", "p")
            type_text("reconnect exited")
            chord("Return")
            time.sleep(0.4)
            reconnected = root / "reconnected"
            type_text(f"printf reconnected > {reconnected}\n")
            wait_for(reconnected.exists, "Exited pane could not reconnect")
            chord("Control_L", "Shift_L", "t")
            time.sleep(0.5)
            third = root / "third"
            type_text(f"printf third > {third}\n")
            wait_for(third.exists, "New tab did not accept input")
            chord("Control_L", "Shift_L", "w")
            time.sleep(0.3)
            chord("Control_L", "p")
            type_text("theme dracula")
            chord("Return")
            time.sleep(0.3)
            # Simulate a storage failure at final save without relying on Unix
            # permissions (which differ when CI runs as root).
            config_dir = root / "config/termxp"
            preserved_dir = root / "preserved-config"
            config_dir.rename(preserved_dir)
            config_dir.write_text("temporarily unavailable")
            # Close relative to the actual window size (small Xvfb screens may clamp it).
            parent = c.c_ulong(); left = c.c_int(); top = c.c_int()
            width = c.c_uint(); height = c.c_uint(); border = c.c_uint(); depth = c.c_uint()
            x.XGetGeometry(display, window, c.byref(parent), c.byref(left), c.byref(top), c.byref(width), c.byref(height), c.byref(border), c.byref(depth))
            t.XTestFakeMotionEvent(display, -1, width.value - 30, 16, 0)
            t.XTestFakeButtonEvent(display, 1, True, 0)
            t.XTestFakeButtonEvent(display, 1, False, 0)
            x.XFlush(display)
            time.sleep(0.6)
            assert app.poll() is None, "Failed final save did not prevent close"
            config_dir.unlink()
            preserved_dir.rename(config_dir)
            t.XTestFakeButtonEvent(display, 1, True, 0)
            t.XTestFakeButtonEvent(display, 1, False, 0)
            x.XFlush(display)
            wait_for(lambda: app.poll() is not None, "Window close did not finish")
            config = json.loads((root / "config/termxp/config.json").read_text())
            assert config["theme"] == "dracula", config
            assert len(config["workspace"]) == 1, config
            assert len(config["workspace"][0]["root"]["children"]) == 2, config
            assert "sessionId" not in json.dumps(config)
            assert app.returncode == 0, app.returncode
            # Restore the saved two-pane workspace in a fresh application process.
            app = subprocess.Popen([binary], env=env, stdout=log, stderr=log)
            wait_for(lambda: find_window() is not None, "Restored window did not appear")
            window = find_window()
            x.XMoveWindow(display, window, 0, 0)
            x.XRaiseWindow(display, window)
            x.XSetInputFocus(display, window, 1, 0)
            x.XFlush(display)
            time.sleep(1)
            restored = root / "restored"
            type_text(f"printf restored > {restored}\n")
            wait_for(restored.exists, "Restored workspace did not create a fresh shell")
            t.XTestFakeMotionEvent(display, -1, width.value - 30, 16, 0)
            t.XTestFakeButtonEvent(display, 1, True, 0)
            t.XTestFakeButtonEvent(display, 1, False, 0)
            x.XFlush(display)
            wait_for(lambda: app.poll() is not None, "Restored window did not close")
            # Unsupported config must survive startup and autosave until the
            # user explicitly invokes recovery; recovery preserves raw bytes.
            original = '{"version":99,"future":"must survive"}'
            config_path = root / "config/termxp/config.json"
            config_path.write_text(original)
            app = subprocess.Popen([binary], env=env, stdout=log, stderr=log)
            wait_for(lambda: find_window() is not None, "Recovery window did not appear")
            window=find_window()
            x.XMoveWindow(display,window,0,0);x.XRaiseWindow(display,window);x.XSetInputFocus(display,window,1,0);x.XFlush(display)
            time.sleep(1)
            assert config_path.read_text() == original, "Unsupported config was overwritten"
            chord("Control_L","p");type_text("recover configuration");chord("Return")
            wait_for(lambda: config_path.read_text() != original, "Explicit recovery did not save")
            backups=list(config_dir.glob("config-recovery-*.json"))
            assert any(path.read_text()==original for path in backups), "Recovery lost original bytes"
            t.XTestFakeMotionEvent(display,-1,width.value-30,16,0)
            t.XTestFakeButtonEvent(display,1,True,0);t.XTestFakeButtonEvent(display,1,False,0);x.XFlush(display)
            wait_for(lambda: app.poll() is not None,"Recovered window did not close")
            print("Native smoke passed: shell I/O, split, reconnect, tabs, palette, failed-save retry, restore, config recovery")
        except BaseException:
            log.flush(); log.seek(0)
            print(log.read(), file=sys.stderr)
            subprocess.run(["import", "-window", "root", "/tmp/termxp-smoke-failure.png"], check=False)
            raise
        finally:
            if app.poll() is None:
                app.terminate()
                try: app.wait(timeout=5)
                except subprocess.TimeoutExpired: app.kill(); app.wait()
x.XCloseDisplay(display)
