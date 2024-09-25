#!/usr/bin/env python3

"""
Automatin tests for following commands:
    "hangar scan"
"""

import os
from .common import run_hangar, check
from .common import SOURCE_REGISTRY_URL, TRIVY_DB_REPO, TRIVY_JAVA_DB_REPO

SCAN_FAILED_LIST = "scan-failed.txt"
SCAN_REPORT_PREFIX = "scan-report"
SCAN_REPORT_JSON = SCAN_REPORT_PREFIX + ".json"
SCAN_REPORT_YAML = SCAN_REPORT_PREFIX + ".yaml"
SCAN_REPORT_CSV = SCAN_REPORT_PREFIX + ".csv"
SCAN_REPORT_SPDX_JSON = SCAN_REPORT_PREFIX + ".spdx.json"
SCAN_REPORT_SPDX_CSV = SCAN_REPORT_PREFIX + ".spdx.csv"
SCAM_REPORT_CUSTOM_NAME = "custom-report.csv"


def prepare():
    lists = [
        SCAN_FAILED_LIST,
        SCAN_REPORT_JSON,
        SCAN_REPORT_YAML,
        SCAN_REPORT_CSV,
        SCAN_REPORT_SPDX_JSON,
        SCAM_REPORT_CUSTOM_NAME,
    ]
    for list in lists:
        if os.path.exists(list):
            os.remove(list)


def check_report(name):
    if not os.path.exists(name):
        raise Exception("scan report file not found:", name)

    with open(name, "r") as file:
        i = 0
        print("scan report output [" + name + "]: ")
        for line in file.readlines():
            if i >= 10:
                break
            i += 1
            print(line.strip())
        if i == 10:
            print("......")


def test_scan_help():
    prepare()
    check(run_hangar(["scan", "--help"]))


def test_scan_image_csv():
    log = run_hangar([
        "scan",
        "--file", "data/scan/scan.txt",
        "--format", "csv",
        "--registry", SOURCE_REGISTRY_URL,
        "--tls-verify=false",
        "--auto-yes",
        "--trivy-db-repo", TRIVY_DB_REPO,
        "--trivy-java-db-repo", TRIVY_JAVA_DB_REPO,
    ])
    check(log, SCAN_FAILED_LIST)
    check_report(SCAN_REPORT_CSV)


def test_scan_image_json():
    log = run_hangar([
        "scan",
        "--file", "data/scan/scan.txt",
        "--format", "json",
        "--registry", SOURCE_REGISTRY_URL,
        "--tls-verify=false",
        "--auto-yes",
        "--trivy-db-repo", TRIVY_DB_REPO,
        "--trivy-java-db-repo", TRIVY_JAVA_DB_REPO,
    ])
    check(log, SCAN_FAILED_LIST)
    check_report(SCAN_REPORT_JSON)


def test_scan_image_yaml():
    log = run_hangar([
        "scan",
        "--file", "data/scan/scan.txt",
        "--format", "yaml",
        "--registry", SOURCE_REGISTRY_URL,
        "--tls-verify=false",
        "--auto-yes",
        "--trivy-db-repo", TRIVY_DB_REPO,
        "--trivy-java-db-repo", TRIVY_JAVA_DB_REPO,
    ])
    check(log, SCAN_FAILED_LIST)
    check_report(SCAN_REPORT_YAML)


def test_scan_image_spdx_json():
    log = run_hangar([
        "scan",
        "--file", "data/scan/scan.txt",
        "--format", "spdx-json",
        "--registry", SOURCE_REGISTRY_URL,
        "--tls-verify=false",
        "--auto-yes",
        "--trivy-db-repo", TRIVY_DB_REPO,
        "--trivy-java-db-repo", TRIVY_JAVA_DB_REPO,
    ])
    check(log, SCAN_FAILED_LIST)
    check_report(SCAN_REPORT_SPDX_JSON)


def test_scan_image_spdx_csv():
    log = run_hangar([
        "scan",
        "--file", "data/scan/scan.txt",
        "--format", "spdx-csv",
        "--registry", SOURCE_REGISTRY_URL,
        "--tls-verify=false",
        "--auto-yes",
        "--trivy-db-repo", TRIVY_DB_REPO,
        "--trivy-java-db-repo", TRIVY_JAVA_DB_REPO,
    ])
    check(log, SCAN_FAILED_LIST)
    check_report(SCAN_REPORT_SPDX_CSV)


def test_scan_image_custom_report():
    log = run_hangar([
        "scan",
        "--file", "data/scan/scan.txt",
        "--report", SCAM_REPORT_CUSTOM_NAME,
        "--registry", SOURCE_REGISTRY_URL,
        "--tls-verify=false",
        "--auto-yes",
        "--trivy-db-repo", TRIVY_DB_REPO,
        "--trivy-java-db-repo", TRIVY_JAVA_DB_REPO,
    ])
    check(log, SCAN_FAILED_LIST)
    check_report(SCAM_REPORT_CUSTOM_NAME)


def test_scan_image_custom_scanners():
    log = run_hangar([
        "scan",
        "--file", "data/scan/scan.txt",
        "--scanner", "vuln,seclog",
        "--registry", SOURCE_REGISTRY_URL,
        "--tls-verify=false",
        "--auto-yes",
        "--trivy-db-repo", TRIVY_DB_REPO,
        "--trivy-java-db-repo", TRIVY_JAVA_DB_REPO,
    ])
    check(log, SCAN_FAILED_LIST)
    check_report(SCAN_REPORT_CSV)


def test_scan_image_skip_db_update():
    log = run_hangar([
        "scan",
        "--file", "data/scan/scan.txt",
        "--skip-db-update",
        "--skip-java-db-update",
        "--registry", SOURCE_REGISTRY_URL,
        "--tls-verify=false",
        "--auto-yes",
        "--trivy-db-repo", TRIVY_DB_REPO,
        "--trivy-java-db-repo", TRIVY_JAVA_DB_REPO,
    ])
    check(log, SCAN_FAILED_LIST)
    check_report(SCAN_REPORT_CSV)
