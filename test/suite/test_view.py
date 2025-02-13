#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar view sbom"
    "hangar view provenance"
"""

from .common import run_hangar, check
from .common import REGISTRY_URL, REGISTRY_PASSWORD


def test_login():
    check(run_hangar([
        "login",
        REGISTRY_URL,
        "-u=admin",
        "-p", REGISTRY_PASSWORD,
        "--tls-verify=false",
    ]))


def test_view_help():
    check(run_hangar(["view", "--help"]))
    check(run_hangar(["view", "sbom", "--help"]))
    check(run_hangar(["view", "provenance", "--help"]))


def test_view():
    log = run_hangar([
        "view",
        "sbom",
        "registry.suse.com/bci/base:15.6",
    ], timeout=600)
    check(log)

    log = run_hangar([
        "view",
        "provenance",
        "registry.suse.com/bci/base:15.6",
    ], timeout=600)
    check(log)
