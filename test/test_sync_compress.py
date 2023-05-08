#!/usr/bin/env python3

# Automatin test for 'hangar sync', 'hangar compress' and
# 'hangar decompress'

import utils as u
import os, glob
import shutil

CACHE_DIRECTORY = 'saved-image-cache'
SYNC_FAILED_LIST = 'sync-failed.txt'

# Save image from the harbor registry server
def get_save_env():
    e = os.environ.copy()
    e['SOURCE_REGISTRY'] = e['DEST_REGISTRY']
    e['SOURCE_USERNAME'] = e['DEST_USERNAME']
    e['SOURCE_PASSWORD'] = e['DEST_PASSWORD']
    e['DEST_REGISTRY'] = e['DEST_USERNAME'] = e['DEST_PASSWORD'] = ''
    return e

def test_sync_help():
    print('test_sync_help')
    # hangar sync -h
    u.run_subprocess(u.hangar, ['sync', '-h'])
    # hangar sync --help
    u.run_subprocess(u.hangar, ['sync', '--help'])

def prepare_sync_files():
    if os.path.isdir('sync-directory'):
        shutil.rmtree('sync-directory')

    print('prepare_sync_files')
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'save',
        '-f', './data/save_test_one.txt',
        '--compress', 'dir',
        '-d', 'sync-directory'
    ], env=e)
    u.check_failed('save-failed.txt')

# Sync jobs 20
def test_sync_jobs_20():
    print('test_sync_jobs_20')
    prepare_sync_files()
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'sync',
        '-f', './data/sync_test.txt',
        '-d', 'sync-directory',
        '-j', '20',
        '--debug'
    ], timeout=600, env=e)
    u.check_failed(SYNC_FAILED_LIST)

# Sync jobs less than 1
def test_sync_jobs_lt_1():
    print('test_sync_jobs_lt_1')
    prepare_sync_files()
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'sync',
        '-f', './data/sync_test.txt',
        '-d', 'sync-directory',
        '-j', '0',
    ], timeout=600, env=e)
    u.check_failed(SYNC_FAILED_LIST)

# Sync jobs larger than 20
def test_sync_jobs_gt_20():
    print('test_sync_jobs_gt_20')
    prepare_sync_files()
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'sync',
        '-f', './data/sync_test.txt',
        '-d', 'sync-directory',
        '-j', '30',
    ], timeout=600, env=e)
    u.check_failed(SYNC_FAILED_LIST)

# Load synced directory onto harbor server
def test_sync_load():
    print('test_sync_load')
    u.run_subprocess(u.hangar, args=[
        'load',
        '-s', 'sync-directory',
        '-j', '5',
        '--compress=dir',
    ], timeout=600)
    u.check_failed('load-failed.txt')

# Validate load synced directory
def test_sync_load_validate():
    print('test_sync_load_validate')
    u.run_subprocess(u.hangar, args=[
        'load-validate',
        '-s', './sync-directory',
        '-j', '5',
        '--compress=dir',
    ], timeout=300)
    u.check_failed('load-validate-failed.txt')

# Compress
def test_compress_help():
    print('test_compress_help')
    u.run_subprocess(u.hangar, args=['compress', '-h'])
    u.run_subprocess(u.hangar, args=['compress', '--help'])

# Compress gzip
def test_compress_gzip():
    print('test_compress_gzip')
    u.run_subprocess(u.hangar, args=[
        'compress',
        '-f', 'sync-directory',
    ])

# Compress zstd
def test_compress_zstd():
    print('test_compress_zstd')
    u.run_subprocess(u.hangar, args=[
        'compress',
        '-f', CACHE_DIRECTORY,
        '--format', 'zstd',
    ])

# Compress gzip segment compress
def test_compress_gzip_segment():
    print('test_compress_gzip_segment')
    u.run_subprocess(u.hangar, args=[
        'compress',
        '-f', CACHE_DIRECTORY,
        '--format', 'gzip',
        '--part',
        '--part-size=100M',
        '--destination=gzip-segment.tar.gz',
    ])
    saved_file_list = glob.glob('gzip-segment.tar.gz*')
    assert len(saved_file_list) > 1
    print('created part files:')
    counter = 0
    for i in range(len(saved_file_list)):
        print(saved_file_list[i])
        info = os.stat(saved_file_list[i])
        print("File:", saved_file_list[i],
              "size:", info.st_size / 1024 / 1024, "MB")
        if info.st_size == 100 * 1024 * 1024:
            counter += 1
    shutil.rmtree(CACHE_DIRECTORY)
    assert counter >= len(saved_file_list) - 1

# Decompress gzip
def test_decompress_gzip():
    print('test_decompress_gzip')
    u.run_subprocess(u.hangar, args=[
        'decompress',
        '-f', 'saved-images.tar.gz',
    ])
    shutil.rmtree(CACHE_DIRECTORY)

# Decompress zstd
def test_decompress_zstd():
    print('test_decompress_zstd')
    u.run_subprocess(u.hangar, args=[
        'decompress',
        '-f', 'saved-images.tar.zstd',
        '--format=zstd',
    ])
    shutil.rmtree(CACHE_DIRECTORY)

# Decompress segment gzip
def test_decompress_gzip_segment():
    print('test_decompress_gzip_segment')
    u.run_subprocess(u.hangar, args=[
        'decompress',
        '-f', 'gzip-segment.tar.gz',
    ])
    shutil.rmtree(CACHE_DIRECTORY)
