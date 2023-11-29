#!/usr/bin/env python3

"""
Automatin tests for "hangar version" command.
"""

from .common import run_hangar, check


def test_version():
    # hangar version
    check(run_hangar(["version"]))
    # hangar -v
    check(run_hangar(["-v"]))
    # hangar --version
    check(run_hangar(["--version"]))
