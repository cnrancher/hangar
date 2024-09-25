#!/usr/bin/env python3

"""
Automatin tests for "hangar convert-list" command.
"""

from .common import run_hangar, check, compare, prepare
from .common import SOURCE_REGISTRY_URL


def test_convert_list_help():
    check(run_hangar(["convert-list", "--help"]))


def test_convert_list():
    prepare("converted.txt")
    check(run_hangar([
        "convert-list",
        "--input=data/convert-list/default.txt",
        "--output=converted.txt",
        "-s", "example.io",
        "-d", SOURCE_REGISTRY_URL,
    ]))
    f = open("converted.txt")
    s = f.read()
    f.close()
    compare(s, "data/convert-list/converted.txt")
