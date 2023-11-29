#!/usr/bin/env python3

"""
Automatin tests for 'hangar login', 'hangar logout' commands.
"""

from .common import run_hangar, check, REGISTRY_URL


def test_login_logout_help():
    check(run_hangar(["login", "--help"]))
    check(run_hangar(["logout", "--help"]))


def test_login_logout():
    # The test registry server can be login with any password.
    check(run_hangar([
        "login",
        REGISTRY_URL,
        "-u=admin",
        "-p=123",
        "--tls-verify=false",
    ]))
    check(run_hangar([
        "logout",
        REGISTRY_URL,
    ]))
    # Re-login for other tests use.
    check(run_hangar([
        "login",
        REGISTRY_URL,
        "-u=admin",
        "-p=000",
        "--tls-verify=false",
    ]))
