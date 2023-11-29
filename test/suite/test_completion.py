#!/usr/bin/env python3

"""
Automatin tests for "hangar completion" command.
"""

from .common import run_hangar, check


def test_completion_help():
    check(run_hangar(["completion", "--help"]))
    check(run_hangar(["completion", "bash", "--help"]))
    check(run_hangar(["completion", "zsh", "--help"]))
    check(run_hangar(["completion", "fish", "--help"]))
    check(run_hangar(["completion", "powershell", "--help"]))


def test_completion():
    check(run_hangar(["completion", "bash"]))
    check(run_hangar(["completion", "zsh"]))
    check(run_hangar(["completion", "fish"]))
    check(run_hangar(["completion", "powershell"]))
