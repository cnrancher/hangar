#!/usr/bin/env python3

import subprocess
import sys
import os
import shutil


HANGAR = shutil.which("hangar")
if HANGAR is None:
    if os.path.exists("../../bin/hangar"):
        HANGAR = "../../bin/hangar"
    elif os.path.exists("../bin/hangar"):
        HANGAR = "../bin/hangar"
    else:
        raise Exception("hangar executable not found")


REGISTRY_URL = os.getenv("REGISTRY_URL")
if REGISTRY_URL is None:
    print("Please run validation test by executing 'scripts/entrypoint.sh'",
          file=sys.stderr)
    raise Exception("REGISTRY_URL env not specified")


def run_hangar(args=[], timeout=1200) -> int:
    args.insert(0, HANGAR)
    # args.append("--insecure-policy")
    process = subprocess.Popen(
        args,
        text=True,
    )
    return process.wait(timeout=timeout)


def check(ret: int, p=None):
    if ret == 0:
        return
    if p is not None and os.path.exists(p):
        f = open(p, "r")
        images = f.read()
        f.close()
        os.remove(p)
        raise Exception("Failed images:", images)
    else:
        raise Exception("hangar run failed:", ret)
