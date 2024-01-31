#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar generate-sigstore-key"
    "hangar sign"
    "hangar sign validate"
    "hangar mirror (with --sigstore-private-key option provided)"
    "hangar load (with --sigstore-private-key option provided)"
"""

import os
from .common import run_hangar, check, REGISTRY_URL

SIGN_FAILED_LIST = "sign-failed.txt"
MIRROR_FAILED_LIST = "mirror-failed.txt"
LOAD_FAILED_LIST = "load-failed.txt"


def prepare():
    lists = [
        SIGN_FAILED_LIST,
        MIRROR_FAILED_LIST,
    ]
    for list in lists:
        if os.path.exists(list):
            os.remove(list)


def test_generate_sigstore_key_help():
    check(run_hangar(["generate-sigstore-key", "--help"]))


def test_sign_help():
    check(run_hangar(["sign", "--help"]))
    check(run_hangar(["sign", "validate", "--help"]))


def test_generate_sigstore_key():
    ret = run_hangar([
        "generate-sigstore-key",
        "--prefix=sigstore",
        "--passphrase-file=data/sigstore_passphrase.txt",
        "--auto-yes",
    ])
    check(ret)


def test_sign():
    ret = run_hangar([
        "sign",
        "--jobs=4",
        "--file=data/sigstore_sign.txt",
        "--registry", REGISTRY_URL,
        "--sigstore-key=sigstore.key",
        "--sigstore-passphrase-file=data/sigstore_passphrase.txt",
        "--tls-verify=false",
    ])
    check(ret, SIGN_FAILED_LIST)


def test_sign_validate():
    ret = run_hangar([
        "sign",
        "validate",
        "--jobs=4",
        "--file=data/sigstore_sign.txt",
        "--registry", REGISTRY_URL,
        "--sigstore-pubkey=sigstore.pub",
        "--tls-verify=false",
    ])
    check(ret, SIGN_FAILED_LIST)


def test_mirror_sigstore_sign():
    ret = run_hangar([
        "mirror",
        "--file=data/mirror_sigstore_sign.txt",
        "-s", REGISTRY_URL,  # use private registry to avoid rate limit.
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--sigstore-private-key=sigstore.key",
        "--sigstore-passphrase-file=data/sigstore_passphrase.txt",
        "--tls-verify=false",
    ], timeout=600)
    check(ret, MIRROR_FAILED_LIST)


def test_validate_mirror_sigstore_sign():
    ret = run_hangar([
        "sign",
        "validate",
        "--jobs=4",
        "--file=data/mirror_sigstore_sign.txt",
        "--registry", REGISTRY_URL,
        "--sigstore-pubkey=sigstore.pub",
        "--tls-verify=false",
    ])
    check(ret, SIGN_FAILED_LIST)


def test_load_sigstore_sign():
    ret = run_hangar([
        "load",
        "--file=data/load_sigstore_sign.txt",
        "-s", "saved_test.zip",
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--source-registry", REGISTRY_URL,
        "--sigstore-private-key=sigstore.key",
        "--sigstore-passphrase-file=data/sigstore_passphrase.txt",
        "--tls-verify=false",
    ], timeout=600)
    check(ret, LOAD_FAILED_LIST)


def test_validate_load_sigstore_sign():
    ret = run_hangar([
        "sign",
        "validate",
        "--jobs=4",
        "--file=data/load_sigstore_sign.txt",
        "--registry", REGISTRY_URL,
        "--sigstore-pubkey=sigstore.pub",
        "--tls-verify=false",
    ])
    check(ret, SIGN_FAILED_LIST)
