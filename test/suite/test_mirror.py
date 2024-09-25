#!/usr/bin/env python3

"""
Automatin tests for commands:
    - hangar mirror
    - hangar mirror validate
"""

import pytest
from .common import run_hangar, check, compare, prepare
from .common import REGISTRY_URL, REGISTRY_PASSWORD, SOURCE_REGISTRY_URL


def test_login():
    check(run_hangar([
        "login",
        REGISTRY_URL,
        "-u=admin",
        "-p", REGISTRY_PASSWORD,
        "--tls-verify=false",
    ]))


def test_mirror_help():
    check(run_hangar(["mirror", "--help"]))
    check(run_hangar(["mirror", "validate", "--help"]))


@pytest.mark.dependency()
def test_mirror_default_format():
    FAILED = "mirror-failed-1.txt"
    prepare(FAILED)
    log = run_hangar([
        "mirror",
        "-f=data/mirror/default_format.txt",
        "-j=4",
        "-s", SOURCE_REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=amd64",
        "--os=linux",
        "--destination-project=library",
        "--tls-verify=false",
        "--provenance=false",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "mirror",
        "validate",
        "-f=data/mirror/default_format.txt",
        "-j=4",
        "-s", SOURCE_REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=amd64",
        "--os=linux",
        "--destination-project=library",
        "--tls-verify=false",
        "--provenance=false",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "inspect",
        "docker://"+REGISTRY_URL+"/library/slsa-provenance-test:latest",
        "--raw",
        "--tls-verify=false",
    ])
    # The inspected image should only have AMD64 architecture
    # and does not have SLSA Provenance
    compare(log, "data/mirror/inspect-1.log")


@pytest.mark.dependency(depends=['test_mirror_default_format'])
def test_mirror_mirror_format():
    FAILED = "mirror-failed-2.txt"
    prepare(FAILED)
    log = run_hangar([
        "mirror",
        "-f=data/mirror/mirror_format.txt",
        "-j=4",
        "-s", SOURCE_REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=arm64",
        "--os=linux",
        "--destination-project=library",
        "--tls-verify=false",
        "--provenance=true",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "mirror",
        "validate",
        "-f=data/mirror/mirror_format.txt",
        "-j=4",
        "-s", SOURCE_REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=arm64",
        "--os=linux",
        "--destination-project=library",
        "--tls-verify=false",
        "--provenance=true",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "inspect",
        "docker://"+REGISTRY_URL+"/library/slsa-provenance-test:latest",
        "--raw",
        "--tls-verify=false",
    ])
    # The inspected image should have AMD64,ARM64 architecture
    # and have SLSA Provenances
    compare(log, "data/mirror/inspect-2.log")
