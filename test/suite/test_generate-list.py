#!/usr/bin/env python3

"""
Automatin tests for "hangar generate-list" command.
"""

import os
from .common import run_hangar, check


def test_generate_list_help():
    check(run_hangar(["generate-list", "--help"]))


def test_generate_list_gc_28():
    check(run_hangar([
        "generate-list",
        "--rancher=v2.8.0-ent",
        "--output=v2.8.0-ent-images.txt",
    ]))
    f = open("v2.8.0-ent-images.txt", "r")
    print("")
    print("Generated image list of 'v2.8.0-ent'")
    for _ in range(0, 5):
        print(f.readline(), end="")
    print("......")
    f.close()
    os.remove("v2.8.0-ent-images.txt")

    f = open("v2.8.0-ent-versions.txt", "r")
    print("")
    print("Generated k8s version list of 'v2.8.0-ent'")
    for _ in range(0, 5):
        print(f.readline(), end="")
    print("......")
    f.close()
    os.remove("v2.8.0-ent-versions.txt")


def test_generate_list_gc_28_dev():
    check(run_hangar([
        "generate-list",
        "--rancher=v2.8.0-ent",
        "--output=v2.8.0-ent-dev-images.txt",
        "--dev"
    ]))
    f = open("v2.8.0-ent-dev-images.txt", "r")
    print("")
    print("Generated image list of 'v2.8.0-ent' from DEV branch")
    for _ in range(0, 5):
        print(f.readline(), end="")
    print("......")
    f.close()
    os.remove("v2.8.0-ent-dev-images.txt")

    f = open("v2.8.0-ent-versions.txt", "r")
    print("")
    print("Generated k8s version list of 'v2.8.0-ent'")
    for _ in range(0, 5):
        print(f.readline(), end="")
    print("......")
    f.close()
    os.remove("v2.8.0-ent-versions.txt")


def test_generate_list_28():
    check(run_hangar([
        "generate-list",
        "--rancher=v2.8.0",
        "--output=v2.8.0-images.txt",
    ]))
    f = open("v2.8.0-images.txt", "r")
    print("")
    print("Generated image list of 'v2.8.0'")
    for _ in range(0, 5):
        print(f.readline(), end="")
    print("......")
    f.close()
    os.remove("v2.8.0-images.txt")

    f = open("v2.8.0-versions.txt", "r")
    print("")
    print("Generated k8s version list of 'v2.8.0'")
    for _ in range(0, 5):
        print(f.readline(), end="")
    print("......")
    f.close()
    os.remove("v2.8.0-versions.txt")


def test_generate_list_28_dev():
    check(run_hangar([
        "generate-list",
        "--rancher=v2.8.0",
        "--output=v2.8.0-dev-images.txt",
    ]))
    f = open("v2.8.0-dev-images.txt", "r")
    print("")
    print("Generated image list of 'v2.8.0' from DEV branch")
    for _ in range(0, 5):
        print(f.readline(), end="")
    print("......")
    f.close()
    os.remove("v2.8.0-dev-images.txt")

    f = open("v2.8.0-versions.txt", "r")
    print("")
    print("Generated k8s version list of 'v2.8.0'")
    for _ in range(0, 5):
        print(f.readline(), end="")
    print("......")
    f.close()
    os.remove("v2.8.0-versions.txt")
