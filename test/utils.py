#!/usr/bin/env python3

import subprocess, platform, glob, sys, os

class TestFailedException(Exception):
    def __init__(self, message):
        self.message = message
    def __str__(self):
        return self.message

# Find hangar path in build folder
def get_hangar_path(version):
    if platform.system() == "Darwin":
        system = "darwin"
    elif platform.system() == "Linux":
        system = "linux"
    else:
        print("unrecognized system: " + platform.system())
        return ""

    if platform.machine() == "arm64":
        arch = "arm64"
    elif platform.machine() == "aarch64":
        arch = "arm64"
    elif platform.machine() == "x86_64":
        arch = "amd64"
    else:
        print("unrecognized arch: " + platform.machine())
        return ""

    print("get_hangar_path: system: " + system)
    print("get_hangar_path: arch: " + arch)
    match = '../build/hangar-' + system + '-' + arch + '-' + version
    matchList = glob.glob(match)
    if not isinstance(matchList, list) or len(matchList) == 0:
        return ""
    return matchList[0]

# Run the subprocess and return its stdout, stderr, return code
def run_subprocess(path, args, timeout=300, stdin=None, stdout=None, stderr=None, env=None):
    print("run_subprocess: run: " + path)
    args.insert(0, path)
    if stdin is None:
        stdin = sys.stdin
    if stdout is None:
        stdout = sys.stdout
    if stderr is None:
        stderr = sys.stderr

    # Launch the program using subprocess
    process = subprocess.Popen(
        args,
        stdin = stdin,
        stdout = stdout,
        stderr = stderr,
        text=True,
        env=env)
    ret = process.wait(timeout=timeout)
    if ret != 0:
        raise TestFailedException("subprocess failed")
    return ret

# Check *-failed.txt image list output
def check_failed(p):
    if os.path.exists(p):
        f = open(p, "r")
        raise TestFailedException(p + ':\n' + f.read())

hangar = get_hangar_path("*")
if hangar == "":
    print("failed to get hangar path")
    exit(1)
