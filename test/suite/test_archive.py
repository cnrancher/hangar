#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar save"
    "hangar save validate"
    "hangar sync"
    "hangar sync validate"
    "hangar load"
    "hangar load validate"
    "hangar archive ls"
    "hangar archive merge"
    "hangar archive export"
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


def test_save_help():
    check(run_hangar(["save", "--help"]))
    check(run_hangar(["save", "validate", "--help"]))


def test_sync_help():
    check(run_hangar(["sync", "--help"]))
    check(run_hangar(["sync", "validate", "--help"]))


def test_load_help():
    check(run_hangar(["load", "--help"]))
    check(run_hangar(["load", "validate", "--help"]))


@pytest.mark.dependency()
def test_save():
    FAILED = "1-failed.txt"
    prepare(FAILED)
    log = run_hangar([
        "save",
        "-f=data/archive/save.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", "1.zip",
        "-j=3",
        "--os=linux",
        "--arch=amd64",
        "-y",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)

    log = run_hangar([
        "save",
        "validate",
        "-f=data/archive/save.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", "1.zip",
        "-j=3",
        "--os=linux",
        "--arch=amd64",
        "-y",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)

    log = run_hangar([
        "archive",
        "ls",
        "-f", "1.zip",
        "--hide-log-time",
    ])
    compare(log, "data/archive/save_ls.log")

    log = run_hangar([
        "load",
        "-s", "1.zip",
        "-d", REGISTRY_URL,
        "-j=3",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)

    log = run_hangar([
        "load",
        "validate",
        "-s", "1.zip",
        "-d", REGISTRY_URL,
        "-j=3",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)


@pytest.mark.dependency()
def test_sync():
    FAILED = "2-failed.txt"
    prepare(FAILED)
    log = run_hangar([
        "save",
        "-f=data/archive/save.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", "2.zip",
        "-j=3",
        "--os=linux",
        "--arch=amd64",
        "-y",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)

    log = run_hangar([
        "sync",
        "-f=data/archive/sync.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", "2.zip",
        "-j=3",
        "--os=linux",
        "--arch=amd64",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)

    log = run_hangar([
        "sync",
        "validate",
        "-f=data/archive/sync.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", "2.zip",
        "-j=3",
        "--os=linux",
        "--arch=amd64",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)

    log = run_hangar([
        "archive",
        "ls",
        "-f", "2.zip",
        "--hide-log-time",
    ])
    compare(log, "data/archive/sync_ls.log")

    log = run_hangar([
        "load",
        "-s", "2.zip",
        "-d", REGISTRY_URL,
        "-j=3",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)

    log = run_hangar([
        "load",
        "validate",
        "-s", "2.zip",
        "-d", REGISTRY_URL,
        "-j=3",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)


@pytest.mark.dependency()
def test_archive_merge_export():
    FAILED = "3-failed.txt"
    prepare(FAILED)
    log = run_hangar([
        "save",
        "-f=data/archive/save.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", "3-1.zip",
        "-j=3",
        "--os=linux",
        "--arch=arm64",
        "-y",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)
    log = run_hangar([
        "save",
        "-f=data/archive/sync.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", "3-2.zip",
        "-j=3",
        "--os=linux",
        "--arch=arm64",
        "-y",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)
    run_hangar([
        "archive",
        "merge",
        "-f", "3-1.zip",
        "-f", "3-2.zip",
        "-o", "3-3.zip",
        "--auto-yes",
    ])
    log = run_hangar([
        "archive",
        "ls",
        "-f", "3-3.zip",
        "--hide-log-time",
    ])
    compare(log, "data/archive/merge_ls.log")

    # Load the merged archive file
    log = run_hangar([
        "load",
        "-s", "3-3.zip",
        "-d", REGISTRY_URL,
        "-j=10",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)
    log = run_hangar([
        "load",
        "validate",
        "-s", "3-3.zip",
        "-d", REGISTRY_URL,
        "-j=10",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)

    # Export image from archive
    log = run_hangar([
        "archive",
        "export",
        "-f", "data/archive/save.txt",
        "--source-registry", SOURCE_REGISTRY_URL,
        "-s", "3-3.zip",
        "-d", "3-4.zip",
        "--auto-yes",
        "--failed", FAILED,
    ])
    check(log, FAILED)

    log = run_hangar([
        "archive",
        "ls",
        "-f", "3-4.zip",
        "--hide-log-time",
    ])
    compare(log, "data/archive/export_ls.log")

    # Load the exported archive file
    log = run_hangar([
        "load",
        "-s", "3-4.zip",
        "-d", REGISTRY_URL,
        "-j=10",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)
    log = run_hangar([
        "load",
        "validate",
        "-s", "3-4.zip",
        "-d", REGISTRY_URL,
        "-j=10",
        "--tls-verify=false",
        "-o", FAILED,
    ])
    check(log, FAILED)


@pytest.mark.dependency(depends=['test_archive_merge_export', 'test_save'])
def test_inspect():
    log = run_hangar([
        "inspect",
        "docker://"+REGISTRY_URL+"/hxstarrys/slsa-provenance-test:latest",
        "--raw",
        "--tls-verify=false",
    ])
    compare(log, "data/archive/inspect.log")
