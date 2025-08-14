#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar list-tags"
    "hangar delete"
"""

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


def test_list_tags_delete_help():
    check(run_hangar(["list-tags", "--help"]))
    check(run_hangar(["delete", "--help"]))


def test_list_tags():
    FAILED = "mirror-list-tags-failed-1.txt"
    prepare(FAILED)
    log = run_hangar([
        "mirror",
        "-f=data/list-tags/mirror.txt",
        "-j=1",
        "-s", SOURCE_REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--destination-project=test-list-tags",
        "--tls-verify=false",
        "--provenance=false",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "list-tags",
        "docker://"+REGISTRY_URL+"/test-list-tags/manifest-v2-list",
        "--tls-verify=false"
    ], timeout=10)
    compare(log, "data/list-tags/list-tags-1.log")


def test_delete():
    log = run_hangar([
        "delete",
        REGISTRY_URL+"/test-list-tags/manifest-v2-list:latest-linux-amd64",
        "--auto-yes",
        "--tls-verify=false"
    ], timeout=10)
    check(log)

    log = run_hangar([
        "list-tags",
        "docker://"+REGISTRY_URL+"/test-list-tags/manifest-v2-list",
        "--tls-verify=false"
    ], timeout=10)
    compare(log, "data/list-tags/list-tags-2.log")

    log = run_hangar([
        "delete",
        REGISTRY_URL+"/test-list-tags/manifest-v2-list:latest-linux-arm64v8",
        "--auto-yes",
        "--tls-verify=false"
    ], timeout=10)
    check(log)

    log = run_hangar([
        "delete",
        REGISTRY_URL+"/test-list-tags/manifest-v2-list:latest",
        "--auto-yes",
        "--tls-verify=false"
    ], timeout=10)
    check(log)

    log = run_hangar([
        "list-tags",
        REGISTRY_URL+"/test-list-tags/manifest-v2-list",
        "--tls-verify=false"
    ], timeout=10)
    compare(log, "data/list-tags/list-tags-3.log")
