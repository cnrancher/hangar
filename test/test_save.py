#!/usr/bin/env python3

# Automatin test for 'hangar save', 'hangar save-validate'

import utils as u
import os, glob
import shutil

CACHE_DIRECTORY = 'saved-image-cache'
SAVE_FAILED_LIST = 'save-failed.txt'

# Save image from the harbor registry server
def get_save_env():
    print('get_save_env')
    e = os.environ.copy()
    e['SOURCE_REGISTRY'] = e['DEST_REGISTRY']
    e['SOURCE_USERNAME'] = e['DEST_USERNAME']
    e['SOURCE_PASSWORD'] = e['DEST_PASSWORD']
    e['DEST_REGISTRY'] = e['DEST_USERNAME'] = e['DEST_PASSWORD'] = ''
    return e

def process_saved_files(saved_file='', delete_saved_file=False):
    print('process_saved_files')
    # Delete cache folder if exists
    if os.path.exists(CACHE_DIRECTORY):
        shutil.rmtree('saved-image-cache')
    # Check save failed image list
    u.check_failed(SAVE_FAILED_LIST)

    if saved_file != '':
        assert os.path.exists(saved_file)
        if delete_saved_file:
            if os.path.isdir(saved_file):
                shutil.rmtree(saved_file)
            else:
                os.remove(saved_file)

def test_save_help():
    print('test_save_help')
    # hangar save -h
    u.run_subprocess(u.hangar, ['save', '-h'])
    # hangar save --help
    u.run_subprocess(u.hangar, ['save', '--help'])

# Save jobs 20
def test_save_jobs_20():
    print('test_save_jobs_20')
    if os.path.exists(CACHE_DIRECTORY):
        shutil.rmtree(CACHE_DIRECTORY)
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'save',
        '-f', './data/save_test.txt',
        '-j', '20',
        '--debug'
    ], timeout=600, env=e)
    process_saved_files('saved-images.tar.gz', True)

# Save jobs 1
def test_save_jobs_1():
    print('test_save_jobs_1')
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'save',
        '-f', './data/save_test_one.txt',
    ], timeout=300, env=e)
    process_saved_files('saved-images.tar.gz', True)

# Save jobs less than 1
def test_save_jobs_lt_1():
    print('test_save_jobs_lt_1')
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'save',
        '-f', './data/save_test_one.txt',
        '-j', '0',
    ], timeout=300, env=e)
    process_saved_files('saved-images.tar.gz', True)

# Save jobs more than 20
def test_save_jobs_gt_20():
    print('test_save_jobs_gt_20')
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'save',
        '-f', './data/save_test_one.txt',
        '-j', '100',
    ], timeout=300, env=e)
    process_saved_files('saved-images.tar.gz', True)

# Save format zstd (jobs 10)
def test_save_zstd():
    print('test_save_zstd')
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'save',
        '-f', './data/save_test_one.txt',
        '-j', '10',
        '--compress', 'zstd',
    ], timeout=300, env=e)
    process_saved_files('saved-images.tar.zstd', True)

# Save without compression, compress=dir
def test_save_dir():
    print('test_save_dir')
    if os.path.exists('saved-images'):
        shutil.rmtree('saved-images')
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'save',
        '-f', './data/save_test_one.txt',
        '-j', '10',
        '--compress', 'dir',
    ], timeout=300, env=e)
    process_saved_files('saved-images', True)

# Save with segment compress (part size 100M)
def test_save_part():
    print('test_save_part')
    e = get_save_env()
    u.run_subprocess(u.hangar, args=[
        'save',
        '-f', './data/save_test_one.txt',
        '-j', '10',
        '--compress', 'gzip',
        '--part', '--part-size=50M',
        ], timeout=300, env=e)
    process_saved_files()
    saved_file_list = glob.glob('saved-images.tar.gz.part*')
    assert len(saved_file_list) != 0
    # Cleanup
    for i in saved_file_list:
        os.remove(i)
