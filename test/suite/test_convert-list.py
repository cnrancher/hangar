#!/usr/bin/env python3

"""
Automatin tests for "hangar convert-list" command.
"""

from .common import run_hangar, check, REGISTRY_URL


def test_convert_list_help():
    check(run_hangar(["convert-list", "--help"]))


def test_convert_list():
    check(run_hangar([
        "convert-list",
        "--input=data/default_format.txt",
        "--output=converted.txt",
        "-s", "docker.io",
        "-d", REGISTRY_URL,
    ]))
    cf = open("converted.txt")
    c = cf.read()
    cf.close()
    print("")
    print("Converted image list:\n"+c)
