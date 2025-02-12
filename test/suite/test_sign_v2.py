#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar generate-sigstore-key"
    "hangar sign"
    "hangar sign validate"
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


def test_signv2_help():
    check(run_hangar(["generate-sigstore-key", "--help"]))
    check(run_hangar(["sign", "--help"]))
    check(run_hangar(["sign", "validate", "--help"]))


def test_generate_sigstore_key():
    log = run_hangar([
        "generate-sigstore-key",
        "--prefix=cosign",
        "--passphrase-file=data/sign/sigstore_passphrase.txt",
        "--auto-yes",
    ])
    check(log)
    f = open("cosign.pub")
    pub = f.read()
    f.close
    f = open("cosign.key")
    key = f.read()
    f.close()
    assert 'BEGIN PUBLIC KEY' in pub
    assert 'BEGIN ENCRYPTED COSIGN PRIVATE KEY' in key


def test_sign_v2():
    FAILED = "sign-failed-v2-1.txt"
    prepare(FAILED)
    log = run_hangar([
        "mirror",
        "-f=data/sign/sign.txt",
        "-j=4",
        "-s", SOURCE_REGISTRY_URL,
        "-d", REGISTRY_URL,
        "--arch=amd64",
        "--os=linux",
        "--destination-project=sign-test-v2",
        "--tls-verify=false",
        "--provenance=true",
        "--failed", FAILED,
    ], timeout=600)
    check(log, FAILED)

    log = run_hangar([
        "sign",
        "--jobs=4",
        "--file=data/sign/sign.txt",
        "--registry", REGISTRY_URL,
        "--project", "sign-test-v2",
        "--key=cosign.key",
        "--passphrase-file=data/sign/sigstore_passphrase.txt",
        "--tls-verify=false",
        "--failed", FAILED,
        "--auto-yes",
    ])
    check(log, FAILED)

    log = run_hangar([
        "sign",
        "validate",
        "--jobs=4",
        "--file=data/sign/sign.txt",
        "--registry", REGISTRY_URL,
        "--project", "sign-test-v2",
        "--key=cosign.pub",
        "--tls-verify=false",
        "--failed", FAILED,
        "--auto-yes",
    ])
    check(log, FAILED)
