#!/usr/bin/env python3

import subprocess
import sys
import os
import shutil
import random
import string


HANGAR = shutil.which("hangar")
if HANGAR is None:
    if os.path.exists("../../bin/hangar"):
        HANGAR = "../../bin/hangar"
    elif os.path.exists("../bin/hangar"):
        HANGAR = "../bin/hangar"
    else:
        raise Exception("hangar executable not found")
    print('hangar is', HANGAR)


REGISTRY_URL = os.getenv("REGISTRY_URL")
if REGISTRY_URL is None:
    print("Please run validation test by executing 'scripts/run.sh'",
          file=sys.stderr)
    raise Exception("REGISTRY_URL env not specified")


SOURCE_REGISTRY_URL = os.getenv("SOURCE_REGISTRY_URL")
if SOURCE_REGISTRY_URL is None:
    SOURCE_REGISTRY_URL = "docker.io"

TRIVY_DB_REPO = os.getenv("TRIVY_DB_REPO")
if TRIVY_DB_REPO is None:
    TRIVY_DB_REPO = "ghcr.io/aquasecurity/trivy-db:2"

TRIVY_JAVA_DB_REPO = os.getenv("TRIVY_JAVA_DB_REPO")
if TRIVY_JAVA_DB_REPO is None:
    TRIVY_JAVA_DB_REPO = "ghcr.io/aquasecurity/trivy-java-db:1"


def run_hangar(args=[], timeout=1200) -> string:
    args.insert(0, HANGAR)
    result = subprocess.run(
        args,
        text=True,
        timeout=timeout,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    try:
        result.check_returncode()
    except subprocess.CalledProcessError as e:
        print(result.stdout)
        raise e
    return result.stdout


def check(log: string, p=None):
    print(log)
    if p is not None and os.path.exists(p):
        f = open(p, "r")
        images = f.read()
        f.close()
        os.remove(p)
        print("Hangar run failed:")
        print(log)
        print("")
        raise Exception("Failed images:", images, "Log", log)


def compare(output: string, expected_file: string):
    f = open(expected_file, "r")
    expected = f.read().strip()
    output = output.strip()
    f.close()
    expected = expected.replace("docker.io", SOURCE_REGISTRY_URL)
    try:
        assert output == expected
    except AssertionError:
        raise Exception("Expected:", expected, "Actual", output)


def prepare(name):
    if os.path.exists(name):
        os.remove(name)


REGISTRY_PASSWORD = os.getenv("REGISTRY_PASSWORD")
if REGISTRY_PASSWORD is None:
    print("registry password not specified, will use random string")
    REGISTRY_PASSWORD = ''.join(
        random.choices(string.ascii_uppercase + string.digits, k=8))
