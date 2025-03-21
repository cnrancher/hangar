#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar archive init"
    "hangar archive store chart"
    "hangar arcihve store file
    "hangar archive export file"
    "hangar archive ls"
"""

from .common import run_hangar, check, compare, prepare
from .common import REGISTRY_URL, REGISTRY_PASSWORD

ARCHIVE_NAME = "test_store_archive.zip"


def test_login():
    check(run_hangar([
        "login",
        REGISTRY_URL,
        "-u=admin",
        "-p", REGISTRY_PASSWORD,
        "--tls-verify=false",
    ]))


def test_archive_init():
    prepare(ARCHIVE_NAME)
    run_hangar([
        "archive",
        "init",
        ARCHIVE_NAME,
    ])


def test_store_chart():
    # Store chart from Helm HTTP Repo
    run_hangar([
        "archive",
        "store",
        "chart",
        "-f", ARCHIVE_NAME,
        "https://charts.rancher.cn/2.10-prime/latest/",
        "--name=rancher",
        "--version=2.10.3-ent",
    ])

    # Store chart from Tarball URL
    run_hangar([
        "archive",
        "store",
        "chart",
        "-f", ARCHIVE_NAME,
        "https://charts.rancher.cn/2.10-prime/latest/rancher-2.10.1-ent.tgz",
    ])

    # Store chart from OCI Repository
    run_hangar([
        "archive",
        "store",
        "chart",
        "-f", ARCHIVE_NAME,
        "oci://ghcr.io/nginx/charts",
        "--name=nginx-ingress",
        "--version=2.0.1",
    ])

    # Store file from local directory
    run_hangar([
        "archive",
        "store",
        "file",
        "-f", ARCHIVE_NAME,
        "./data/archive/save.txt",
    ])

    # Store file from remote URL
    run_hangar([
        "archive",
        "store",
        "file",
        "-f", ARCHIVE_NAME,
        "https://dl.rancher.cn/2.10/v2.10.3-ent_sha256sum.txt",
    ])

    # Extract custom file
    run_hangar([
        "archive",
        "export",
        "file",
        "-f", ARCHIVE_NAME,
        "-n", "v2.10.3-ent_sha256sum.txt",
        "-y",
    ])
    run_hangar([
        "archive",
        "export",
        "file",
        "-f", ARCHIVE_NAME,
        "-n", "save.txt",
        "-y",
    ])


def test_archive_ls():
    log = run_hangar([
        "archive",
        "ls",
        "-f", ARCHIVE_NAME,
        "--hide-log-time",
    ])
    compare(log, "data/archive/store_ls.log")
