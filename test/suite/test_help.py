#!/usr/bin/env python3

"""
Automation tests for "hangar help" command.
"""

from .common import run_hangar, check


def test_help():
    # hangar help
    check(run_hangar(["help"]))
    # hangar --help
    check(run_hangar(["--help"]))
