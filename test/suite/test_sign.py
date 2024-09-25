#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar generate-sigstore-key"
    "hangar sign"
    "hangar sign validate"
    "hangar mirror (with --sigstore-private-key option provided)"
    "hangar load (with --sigstore-private-key option provided)"
"""

from .common import run_hangar, check, prepare
from .common import REGISTRY_URL, REGISTRY_PASSWORD, SOURCE_REGISTRY_URL


def test_login():
    check(run_hangar([
        "login",
        REGISTRY_URL,
        "-u=admin",
        "-p", REGISTRY_PASSWORD,
        "--tls-verify=false",
    ]))


def test_sign_help():
    check(run_hangar(["generate-sigstore-key", "--help"]))
    check(run_hangar(["sign", "--help"]))
    check(run_hangar(["sign", "validate", "--help"]))


def test_generate_sigstore_key():
    log = run_hangar([
        "generate-sigstore-key",
        "--prefix=sigstore",
        "--passphrase-file=data/sign/sigstore_passphrase.txt",
        "--auto-yes",
    ])
    check(log)
    f = open("sigstore.pub")
    pub = f.read()
    f.close
    f = open("sigstore.key")
    key = f.read()
    f.close()
    assert 'BEGIN PUBLIC KEY' in pub
    assert 'BEGIN ENCRYPTED COSIGN PRIVATE KEY' in key


def test_load_sigstore_sign():
    FAILED = "sign-failed-1.txt"
    prepare(FAILED)
    log = run_hangar([
        "save",
        "--file=data/sign/save.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", "sign-save-1.zip",
        "--arch=amd64,arm64",
        "--os=linux",
        "--tls-verify=false",
        "--auto-yes",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "load",
        "--file", "data/sign/save.txt",
        "-s", "sign-save-1.zip",
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--project=sign-test",
        "--source-registry", SOURCE_REGISTRY_URL,
        "--sigstore-private-key=sigstore.key",
        "--sigstore-passphrase-file=data/sign/sigstore_passphrase.txt",
        "--tls-verify=false",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "sign",
        "validate",
        "--jobs=4",
        "--file=data/sign/save.txt",
        "--project=sign-test",
        "--registry", REGISTRY_URL,
        "--sigstore-pubkey=sigstore.pub",
        "--tls-verify=false",
        "--failed", FAILED,
    ])
    check(log, FAILED)


def test_mirror_sigstore_sign():
    FAILED = "sign-failed-2.txt"
    prepare(FAILED)
    log = run_hangar([
        "mirror",
        "--file=data/sign/mirror.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--destination-project=sign-test",
        "--sigstore-private-key=sigstore.key",
        "--sigstore-passphrase-file=data/sign/sigstore_passphrase.txt",
        "--tls-verify=false",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "mirror",
        "validate",
        "--file=data/sign/mirror.txt",
        "-s", SOURCE_REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=amd64,arm64",
        "--os=linux",
        "--destination-project=sign-test",
        "--sigstore-private-key=sigstore.key",
        "--sigstore-passphrase-file=data/sign/sigstore_passphrase.txt",
        "--tls-verify=false",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "sign",
        "validate",
        "--jobs=4",
        "--file=data/sign/mirror.txt",
        "--registry", REGISTRY_URL,
        "--project", "sign-test",
        "--sigstore-pubkey=sigstore.pub",
        "--tls-verify=false",
        "--failed", FAILED,
    ])
    check(log, FAILED)


def test_sign():
    FAILED = "sign-failed-3.txt"
    prepare(FAILED)
    log = run_hangar([
        "sign",
        "--jobs=4",
        "--file=data/sign/sign.txt",
        "--registry", REGISTRY_URL,
        "--project", "sign-test",
        "--sigstore-key=sigstore.key",
        "--sigstore-passphrase-file=data/sign/sigstore_passphrase.txt",
        "--tls-verify=false",
        "--failed", FAILED,
    ])
    check(log, FAILED)

    log = run_hangar([
        "sign",
        "validate",
        "--jobs=4",
        "--file=data/sign/sign.txt",
        "--registry", REGISTRY_URL,
        "--project", "sign-test",
        "--sigstore-pubkey=sigstore.pub",
        "--tls-verify=false",
        "--failed", FAILED,
    ])
    check(log, FAILED)
