#!/usr/bin/env python3

"""
Automatin tests for "hangar generate-list" command.
"""

import os
from .common import run_hangar, check


def handle_generate_file(name):
    f = open(name, "r")
    print("")
    print("Generated " + name)
    for _ in range(0, 5):
        print(f.readline(), end="")
    print("......")
    f.close()
    os.remove(name)


def test_generate_list_help():
    check(run_hangar(["generate-list", "--help"]))


def test_generate_list_gc_kdm_detailed_output():
    check(run_hangar([
        "generate-list",
        "--rancher=v2.9.99-ent",
        "--output=v2.9.99-ent-images.txt",
        "--rke-images=rke-images.txt",
        "--rke2-images=rke2-images.txt",
        "--k3s-images=k3s-images.txt",
        "--rke2-windows-images=rke2-windows-images.txt",
        "--kdm-remove-deprecated=false",
        "--auto-yes",
    ]))
    handle_generate_file("v2.9.99-ent-images.txt")
    handle_generate_file("v2.9.99-ent-versions.txt")
    handle_generate_file("rke-images.txt")
    handle_generate_file("rke2-images.txt")
    handle_generate_file("k3s-images.txt")
    handle_generate_file("rke2-windows-images.txt")


def test_generate_list_gc_29_dev():
    check(run_hangar([
        "generate-list",
        "--rancher=v2.9.99-ent",
        "--output=v2.9.99-ent-dev-images.txt",
        "--dev"
    ]))
    handle_generate_file("v2.9.99-ent-dev-images.txt")
    handle_generate_file("v2.9.99-ent-versions.txt")


def test_generate_list_29():
    check(run_hangar([
        "generate-list",
        "--rancher=v2.9.99",
        "--output=v2.9.99-images.txt",
    ]))
    handle_generate_file("v2.9.99-images.txt")
    handle_generate_file("v2.9.99-versions.txt")


def test_generate_list_29_dev():
    check(run_hangar([
        "generate-list",
        "--rancher=v2.9.99",
        "--output=v2.9.99-dev-images.txt",
    ]))
    handle_generate_file("v2.9.99-dev-images.txt")
    handle_generate_file("v2.9.99-versions.txt")
