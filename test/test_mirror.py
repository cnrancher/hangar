#!/usr/bin/env python3

# Automatin test for 'hangar mirror', 'hangar mirror-validate'

import utils as u
import os

MIRROR_FAILED_LIST = 'mirror-failed.txt'
VALIDATE_FAILED_LIST = 'mirror-validate-failed.txt'

# Mirror from same image registry to prevent docker pull rate limit
def get_mirror_from_save_registry_env():
    e = os.environ.copy()
    e['SOURCE_REGISTRY'] = e['DEST_REGISTRY']
    e['SOURCE_USERNAME'] = e['DEST_USERNAME']
    e['SOURCE_PASSWORD'] = e['DEST_PASSWORD']
    return e

def test_mirror_help():
    print('test_mirror_help')
    # hangar mirror -h
    u.run_subprocess(u.hangar, ['mirror', '-h'])
    # hangar mirror --help
    u.run_subprocess(u.hangar, ['mirror', '--help'])

# Mirror jobs 10
def test_mirror_jobs_20():
    print('test_mirror_jobs_20')
    u.run_subprocess(u.hangar, [
        'mirror',
        '-f', './data/mirror_test.txt',
        '-j', '20',
        '--debug',
        '--repo-type=harbor'
    ], timeout=600)
    u.check_failed(MIRROR_FAILED_LIST)

# Mirror validate jobs 10
def test_mirror_validate():
    print('test_mirror_validate')
    u.run_subprocess(u.hangar, [
        'mirror-validate',
        '-f', './data/mirror_test.txt',
        '-j', '20'
    ], timeout=300)
    u.check_failed(VALIDATE_FAILED_LIST)

# Mirror jobs 1
def test_mirror_jobs_1():
    print('test_mirror_jobs_1')
    u.run_subprocess(u.hangar, args=[
        'mirror',
        '-f', './data/mirror_test_one.txt',
        '--repo-type=harbor'
    ], timeout=300, env=get_mirror_from_save_registry_env())
    u.check_failed(MIRROR_FAILED_LIST)

# Mirror jobs less than 1
def test_mirror_jobs_lt_1():
    print('test_mirror_jobs_lt_1')
    u.run_subprocess(u.hangar, args=[
        'mirror',
        '-f', './data/mirror_test_one.txt',
        '-j', '0',
        '--repo-type=harbor',
    ], timeout=300, env=get_mirror_from_save_registry_env())
    u.check_failed(MIRROR_FAILED_LIST)

# Mirror jobs more than 20
def test_mirror_jobs_gt_20():
    print('test_mirror_jobs_gt_20')
    u.run_subprocess(u.hangar, args=[
        'mirror',
        '-f', './data/mirror_test_one.txt',
        '-j', '100',
    ], timeout=300, env=get_mirror_from_save_registry_env())
    u.check_failed(MIRROR_FAILED_LIST)
