#!/usr/bin/env python3

"""
Automatin tests for "hangar mirror", "hangar mirror validate" commands.
"""

import os
from .common import run_hangar, check, REGISTRY_URL

MIRROR_FAILED_LIST = "mirror-failed.txt"


def prepare():
    if os.path.exists(MIRROR_FAILED_LIST):
        os.remove(MIRROR_FAILED_LIST)


def test_mirror_help():
    check(run_hangar(["mirror", "--help"]))
    check(run_hangar(["mirror", "validate", "--help"]))


def test_mirror_default_format():
    ret = run_hangar([
        "mirror",
        "-f=data/default_format.txt",
        "-j=4",
        "-s", REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--tls-verify=false",
    ], timeout=600)
    check(ret, MIRROR_FAILED_LIST)


def test_mirror_validate_default_format():
    ret = run_hangar([
        "mirror",
        "validate",
        "-f=data/default_format.txt",
        "-j=4",
        "-s", REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--tls-verify=false",
    ], timeout=600)
    check(ret, MIRROR_FAILED_LIST)


def test_mirror_mirror_format():
    ret = run_hangar([
        "mirror",
        "-f=data/mirror_format.txt",
        "-j=4",
        "-s", REGISTRY_URL,  # Use private registry to avoid rate limit.
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--remove-signatures",  # Hangar does not support sign for harbor
        "--tls-verify=false",
    ], timeout=600)
    check(ret, MIRROR_FAILED_LIST)


def test_mirror_validate_mirror_format():
    ret = run_hangar([
        "mirror",
        "validate",
        "-f=data/mirror_format.txt",
        "-j=4",
        "-s", REGISTRY_URL,  # use private registry to avoid rate limit.
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--tls-verify=false",
    ], timeout=600)
    check(ret, MIRROR_FAILED_LIST)
