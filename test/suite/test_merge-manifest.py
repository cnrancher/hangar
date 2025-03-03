#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar merge-manifest"
"""

from .common import run_hangar, check, compare
from .common import REGISTRY_URL, REGISTRY_PASSWORD


def test_login():
    check(run_hangar([
        "login",
        REGISTRY_URL,
        "-u=admin",
        "-p", REGISTRY_PASSWORD,
        "--tls-verify=false",
    ]))


def test_merge_manifest_help():
    check(run_hangar(["merge-manifest", "--help"]))


def test_merge_manifest():
    log = run_hangar([
        "merge-manifest",
        REGISTRY_URL+"/library/manifest-v2-list:merged",
        REGISTRY_URL+"/library/manifest-v2-list:latest-linux-amd64",
        REGISTRY_URL+"/library/manifest-v2-list:latest-linux-arm64v8",
        "--dry-run",
        "--tls-verify=false",
    ], timeout=600)
    check(log)

    log = run_hangar([
        "merge-manifest",
        REGISTRY_URL+"/library/manifest-v2-list:merged",
        REGISTRY_URL+"/library/manifest-v2-list:latest-linux-amd64",
        REGISTRY_URL+"/library/manifest-v2-list:latest-linux-arm64v8",
        "--tls-verify=false",
    ], timeout=600)
    check(log)

    log = run_hangar([
        "inspect",
        "docker://"+REGISTRY_URL+"/library/manifest-v2-list:merged",
        "--raw",
        "--tls-verify=false",
    ])
    compare(log, "data/merge-manifest/inspect-1.log")
