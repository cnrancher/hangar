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

import os
from .common import run_hangar, check, REGISTRY_URL

SAVE_FAILED_LIST = "save-failed.txt"
SYNC_FAILED_LIST = "sync-failed.txt"
LOAD_FAILED_LIST = "load-failed.txt"
MERGE_FAILED_LIST = "merge-failed.txt"
EXPORT_FAILED_LIST = "export-failed.txt"


def prepare():
    lists = [
        SAVE_FAILED_LIST,
        SYNC_FAILED_LIST,
        LOAD_FAILED_LIST,
    ]
    for list in lists:
        if os.path.exists(list):
            os.remove(list)


def test_save_help():
    check(run_hangar(["save", "--help"]))
    check(run_hangar(["save", "validate", "--help"]))


def test_sync_help():
    check(run_hangar(["sync", "--help"]))
    check(run_hangar(["sync", "validate", "--help"]))


def test_load_help():
    check(run_hangar(["load", "--help"]))
    check(run_hangar(["load", "validate", "--help"]))


def test_save():
    ret = run_hangar([
        "save",
        "-f=data/default_format.txt",
        "-d", "saved_test.zip",
        "-j=3",
        "-y",
        "--tls-verify=false",
    ])
    check(ret, SAVE_FAILED_LIST)


def test_save_validate():
    ret = run_hangar([
        "save",
        "validate",
        "-f=data/default_format.txt",
        "-d", "saved_test.zip",
        "-j=3",
        "-y",
        "--tls-verify=false",
    ])
    check(ret, SAVE_FAILED_LIST)


def test_sync():
    ret = run_hangar([
        "sync",
        "-f=data/sync.txt",
        "-d", "saved_test.zip",
        "-j=3",
        "--tls-verify=false",
    ])
    check(ret, SYNC_FAILED_LIST)


def test_sync_validate():
    ret = run_hangar([
        "sync",
        "validate",
        "-f=data/sync.txt",
        "-d", "saved_test.zip",
        "-j=3",
        "--tls-verify=false",
    ])
    check(ret, SYNC_FAILED_LIST)


def test_load():
    ret = run_hangar([
        "load",
        "-s", "saved_test.zip",
        "-d", REGISTRY_URL,
        "-j=3",
        "--tls-verify=false",
    ])
    check(ret, LOAD_FAILED_LIST)


def test_load_validate():
    ret = run_hangar([
        "load",
        "validate",
        "-s", "saved_test.zip",
        "-d", REGISTRY_URL,
        "-j=3",
        "--tls-verify=false",
    ])
    check(ret, LOAD_FAILED_LIST)


def test_archive_ls():
    check(run_hangar([
        "archive",
        "ls",
        "-f", "saved_test.zip",
    ]))


def test_archive_export():
    ret = run_hangar([
        "archive",
        "export",
        "-f", "data/export1.txt",
        "-s", "saved_test.zip",
        "-d", "export1.zip",
        "--auto-yes",
    ])
    check(ret, EXPORT_FAILED_LIST)

    ret = run_hangar([
        "archive",
        "export",
        "-f", "data/export2.txt",
        "-s", "saved_test.zip",
        "-d", "export2.zip",
        "--auto-yes",
    ])
    check(ret, EXPORT_FAILED_LIST)

    # Check the exported archive files.
    check(run_hangar([
        "archive",
        "ls",
        "-f", "export1.zip",
    ]))
    check(run_hangar([
        "archive",
        "ls",
        "-f", "export2.zip",
    ]))

    # Load the exported archive file
    ret = run_hangar([
        "load",
        "-s", "export1.zip",
        "-d", REGISTRY_URL,
        "-j=10",
        "--tls-verify=false",
    ])
    check(ret, LOAD_FAILED_LIST)


def test_archive_merge():
    ret = run_hangar([
        "archive",
        "merge",
        "-f", "export1.zip",
        "-f", "export2.zip",
        "-o", "merge.zip",
        "--auto-yes",
    ])
    check(ret, MERGE_FAILED_LIST)

    # Check the merged archive files.
    check(run_hangar([
        "archive",
        "ls",
        "-f", "merge.zip",
    ]))

    # Load the merged archive file
    ret = run_hangar([
        "load",
        "-s", "merge.zip",
        "-d", REGISTRY_URL,
        "--project=mirror-test",  # Auto create 'mirror-test' project
        "-j=10",
        "--tls-verify=false",
    ])
    check(ret, LOAD_FAILED_LIST)
