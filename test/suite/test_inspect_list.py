#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar inspect list"
"""

import os
from .common import run_hangar, check


INSPECT_REPORT_TXT = "inspect-report.txt"
INSPECT_REPORT_JSON = "inspect-report.json"
INSPECT_REPORT_YAML = "inspect-report.yaml"
INSPECT_REPORT_CSV = "inspect-report.csv"
INSPECT_REPORT_CUSTOM_NAME = "custom-inspect-report.txt"
INSPECT_FAIL_LIST = "inspect-failed.txt"


def prepare():
    lists = [
        INSPECT_REPORT_TXT,
        INSPECT_REPORT_JSON,
        INSPECT_REPORT_YAML,
        INSPECT_REPORT_CSV,
        INSPECT_REPORT_CUSTOM_NAME,
    ]
    for list in lists:
        if os.path.exists(list):
            os.remove(list)


def check_report(name):
    if not os.path.exists(name):
        raise Exception("inspect report file not found:", name)

    with open(name, "r") as file:
        i = 0
        print("inspect report output [" + name + "]: ")
        for line in file.readlines():
            if i >= 10:
                break
            i += 1
            print(line.strip())
        if i == 10:
            print("......")


def test_inspect_list_help():
    prepare()
    check(run_hangar(["inspect", "list", "--help"]))


def test_inspect_list_txt():
    log = run_hangar([
        "inspect",
        "list",
        "--file", "data/inspect-list/images.txt",
        "--format", "txt",
        "--registry", "",
        "--tls-verify=false",
        "--auto-yes",
    ])
    check(log, INSPECT_FAIL_LIST)
    check_report(INSPECT_REPORT_TXT)


def test_inspect_list_json():
    log = run_hangar([
        "inspect",
        "list",
        "--file", "data/inspect-list/images.txt",
        "--format", "json",
        "--registry", "",
        "--tls-verify=false",
        "--auto-yes",
    ])
    check(log, INSPECT_FAIL_LIST)
    check_report(INSPECT_REPORT_JSON)


def test_inspect_list_yaml():
    log = run_hangar([
        "inspect",
        "list",
        "--file", "data/inspect-list/images.txt",
        "--format", "yaml",
        "--registry", "",
        "--tls-verify=false",
        "--jobs=10",
        "--auto-yes",
    ])
    check(log, INSPECT_FAIL_LIST)
    check_report(INSPECT_REPORT_YAML)


def test_inspect_list_csv():
    log = run_hangar([
        "inspect",
        "list",
        "--file", "data/inspect-list/images.txt",
        "--format", "csv",
        "--registry", "",
        "--tls-verify=false",
        "--jobs=10",
        "--auto-yes",
    ])
    check(log, INSPECT_FAIL_LIST)
    check_report(INSPECT_REPORT_CSV)


def test_inspect_list_custom():
    log = run_hangar([
        "inspect",
        "list",
        "--file", "data/inspect-list/images.txt",
        "--format", "json",
        "--registry", "",
        "--tls-verify=false",
        "--debug",
        "--report", INSPECT_REPORT_CUSTOM_NAME,
        "--auto-yes",
    ])
    check(log, INSPECT_FAIL_LIST)
    check_report(INSPECT_REPORT_CUSTOM_NAME)
