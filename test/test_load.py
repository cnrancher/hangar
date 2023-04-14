#!/usr/bin/env python3

import utils as u
import os, glob, shutil

CACHE_DIRECTORY = 'saved-image-cache'
LOAD_FAILED_LIST = 'save-failed.txt'

def get_save_env():
    e = os.environ.copy()
    e['SOURCE_REGISTRY'] = e['DEST_REGISTRY']
    e['SOURCE_USERNAME'] = e['DEST_USERNAME']
    e['SOURCE_PASSWORD'] = e['DEST_PASSWORD']
    e['DEST_REGISTRY'] = e['DEST_USERNAME'] = e['DEST_PASSWORD'] = ''
    return e

def ensure_load_file_exists():
    load_file_list = glob.glob('load-part.tar.gz.part*')
    if not os.path.isfile('load.tar.gz'):
        print('prepare load tgz file: ')
        e = get_save_env()
        u.run_subprocess(u.hangar, args=[
            'save',
            '-f', './data/save_test.txt',
            '-j', '10',
            '--compress', 'gzip',
            '-d', 'load.tar.gz'
        ], timeout=300, env=e)
        shutil.rmtree('saved-image-cache')
        u.check_failed('save-failed.txt')

    if len(load_file_list) == 0:
        print('prepare load part files')
        e = get_save_env()
        u.run_subprocess(u.hangar, args=[
            'save',
            '-f', './data/save_test_one.txt',
            '-j', '10',
            '--compress', 'gzip',
            '--part', '--part-size=50M',
            '-d', 'load-part'
        ], timeout=300, env=e)
        shutil.rmtree('saved-image-cache')
        u.check_failed('save-failed.txt')

    if not os.path.isfile('load.tar.zstd'):
        print('prepare load zstd file: ')
        e = get_save_env()
        u.run_subprocess(u.hangar, args=[
            'save',
            '-f', './data/save_test_one.txt',
            '-j', '10',
            '--compress', 'zstd',
            '-d', 'load'
        ], timeout=300, env=e)
        shutil.rmtree('saved-image-cache')
        u.check_failed('save-failed.txt')

    if not os.path.isdir('load-directory'):
        print('prepare load directory: ')
        e = get_save_env()
        u.run_subprocess(u.hangar, args=[
            'save',
            '-f', './data/save_test_one.txt',
            '-j', '10',
            '--compress', 'dir',
            '-d', 'load-directory'
        ], timeout=300, env=e)
        u.check_failed('save-failed.txt')

def cleanup_generated_files():
    if os.path.exists(CACHE_DIRECTORY):
        shutil.rmtree(CACHE_DIRECTORY)
    u.check_failed(LOAD_FAILED_LIST)

def test_load_help():
    print('test_load_help')
    # hangar load -h
    u.run_subprocess(u.hangar, ['load', '-h'])
    # hangar load --help
    u.run_subprocess(u.hangar, ['load', '--help'])

# Load jobs 20
def test_load_jobs_20():
    print('test_load_jobs_20')
    ensure_load_file_exists()
    u.run_subprocess(u.hangar, args=[
        'load',
        '-s', 'load.tar.gz',
        '-j', '20',
        '--compress', 'gzip'
    ], timeout=300)
    cleanup_generated_files()

# Load validate
def test_load_validate():
    print('test_load_validate')
    u.run_subprocess(u.hangar, args=[
        'load-validate',
        '-s', './load.tar.gz',
        '-j', '10'
    ], timeout=300)
    cleanup_generated_files()
    u.check_failed('load-validate-failed.txt')

# Load compress=dir
def test_load_dir():
    print('test_load_dir')
    ensure_load_file_exists()
    u.run_subprocess(u.hangar, args=[
        'load',
        '-s', 'load-directory',
        '-j', '10',
        '--compress', 'dir'
    ], timeout=300)
    cleanup_generated_files()

# Load validate compress=dir
def test_load_validate_dir():
    print('test_load_validate_dir')
    u.run_subprocess(u.hangar, args=[
        'load-validate',
        '-s', './load-directory',
        '-j', '10',
        '--compress=dir',
    ], timeout=300)
    cleanup_generated_files()
    u.check_failed('load-validate-failed.txt')

# Load zstd
def test_load_zstd():
    print('test_load_zstd')
    ensure_load_file_exists()
    u.run_subprocess(u.hangar, args=[
        'load',
        '-s', 'load.tar.zstd',
        '-j', '10',
        '--compress', 'zstd'
    ], timeout=300)
    cleanup_generated_files()

# Load validate compress=zstd
def test_load_validate_dir():
    print('test_load_validate_dir')
    u.run_subprocess(u.hangar, args=[
        'load-validate',
        '-s', 'load.tar.zstd',
        '-j', '10',
        '--compress=zstd',
    ], timeout=300)
    cleanup_generated_files()
    u.check_failed('load-validate-failed.txt')

# Load jobs less than 1
def test_load_jobs_1():
    print('test_load_jobs_1')
    ensure_load_file_exists()
    u.run_subprocess(u.hangar, args=[
        'load',
        '-s', 'load.tar.zstd',
        '-j', '0',
        '--compress', 'zstd'
    ], timeout=300)
    cleanup_generated_files()

# Load jobs more than 20
def test_load_jobs_lt_20():
    print('test_load_jobs_lt_20')
    ensure_load_file_exists()
    u.run_subprocess(u.hangar, args=[
        'load',
        '-s', 'load.tar.zstd',
        '-j', '30',
        '--compress', 'zstd'
    ], timeout=300)
    cleanup_generated_files()

# Load part
def test_load_part():
    print('test_load_part')
    ensure_load_file_exists()
    u.run_subprocess(u.hangar, args=[
        'load',
        '-s', 'load-part.tar.gz.part0',
        '-j', '10',
        '--compress', 'gzip'
    ], timeout=300)
    cleanup_generated_files()
