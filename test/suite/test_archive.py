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
"""

import os
from .common import run_hangar, check, REGISTRY_URL

SAVE_FAILED_LIST = "save-failed.txt"
SYNC_FAILED_LIST = "sync-failed.txt"
LOAD_FAILED_LIST = "load-failed.txt"


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
        "-s", REGISTRY_URL,
        "-d", "saved_test.zip",
        "-j=10",
        "-y",
        "--tls-verify=false",
    ])
    check(ret, SAVE_FAILED_LIST)


def test_save_validate():
    ret = run_hangar([
        "save",
        "validate",
        "-f=data/default_format.txt",
        "-s", REGISTRY_URL,
        "-d", "saved_test.zip",
        "-j=10",
        "-y",
        "--tls-verify=false",
    ])
    check(ret, SAVE_FAILED_LIST)


def test_sync():
    ret = run_hangar([
        "sync",
        "-f=data/sync.txt",
        "-s", REGISTRY_URL,
        "-d", "saved_test.zip",
        "-j=10",
        "--tls-verify=false",
    ])
    check(ret, SYNC_FAILED_LIST)


def test_sync_validate():
    ret = run_hangar([
        "sync",
        "validate",
        "-f=data/sync.txt",
        "-s", REGISTRY_URL,
        "-d", "saved_test.zip",
        "-j=10",
        "--tls-verify=false",
    ])
    check(ret, SYNC_FAILED_LIST)


def test_load():
    ret = run_hangar([
        "load",
        "-s", "saved_test.zip",
        "-d", REGISTRY_URL,
        "-j=10",
        "--tls-verify=false",
    ])
    check(ret, LOAD_FAILED_LIST)


def test_load_validate():
    ret = run_hangar([
        "load",
        "validate",
        "-s", "saved_test.zip",
        "-d", REGISTRY_URL,
        "-j=10",
        "--tls-verify=false",
    ])
    check(ret, LOAD_FAILED_LIST)


def test_archive_ls():
    check(run_hangar([
        "archive",
        "ls",
        "-f", "saved_test.zip",
    ]))
