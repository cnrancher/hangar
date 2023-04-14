#!/usr/bin/env python3

import utils as u
import os

def test_generate_list_help():
    print('test_generate_list_help')
    u.run_subprocess(u.hangar, ['generate-list', '-h'])
    u.run_subprocess(u.hangar, ['generate-list', '--help'])

def test_generate_list_gc_2_7():
    print('test_generate_list_gc_2_7')
    u.run_subprocess(u.hangar, args=[
        'generate-list',
        '--rancher=v2.7.2-ent',
        '--output=v2.7.2-ent-images.txt',
    ])
    f = open('v2.7.2-ent-images.txt', 'r')
    for _ in range(0, 5):
        print(f.readline(), end='')
    print('......')
    f.close()
    os.remove('v2.7.2-ent-images.txt')

def test_generate_list_gc_2_7_dev():
    print('test_generate_list_gc_2_7_dev')
    u.run_subprocess(u.hangar, args=[
        'generate-list',
        '--rancher=v2.7.2-ent',
        '--output=v2.7.2-ent-dev-images.txt',
        '--dev',
    ])
    f = open('v2.7.2-ent-dev-images.txt', 'r')
    for _ in range(0, 5):
        print(f.readline(), end='')
    print('......')
    f.close()
    os.remove('v2.7.2-ent-dev-images.txt')

def test_generate_list_gc_2_6():
    print('test_generate_list_gc_2_6')
    u.run_subprocess(u.hangar, args=[
        'generate-list',
        '--rancher=v2.6.11-ent',
        '--output=v2.6.11-ent-images.txt',
    ])
    f = open('v2.6.11-ent-images.txt', 'r')
    for _ in range(0, 5):
        print(f.readline(), end='')
    print('......')
    f.close()
    os.remove('v2.6.11-ent-images.txt')

def test_generate_list_gc_2_5():
    print('test_generate_list_gc_2_5')
    u.run_subprocess(u.hangar, args=[
        'generate-list',
        '--rancher=v2.5.16-ent',
        '--output=v2.5.16-ent-images.txt',
    ])
    f = open('v2.5.16-ent-images.txt', 'r')
    for _ in range(0, 5):
        print(f.readline(), end='')
    print('......')
    f.close()
    os.remove('v2.5.16-ent-images.txt')

def test_generate_list_2_7():
    print('test_generate_list_2_7')
    u.run_subprocess(u.hangar, args=[
        'generate-list',
        '--rancher=v2.7.2',
        '--output=v2.7.2-images.txt',
    ])
    f = open('v2.7.2-images.txt', 'r')
    for _ in range(0, 5):
        print(f.readline(), end='')
    print('......')
    f.close()
    os.remove('v2.7.2-images.txt')

# Test generate list from custom downloaded chart repo path and KDM data file
def test_generate_list_custom_files():
    print('test_generate_list_custom_files')
    # Clone and download custom files
    u.run_subprocess('git', args=[
        'clone',
        'https://github.com/cnrancher/pandaria-catalog',
    ], timeout=60)
    u.run_subprocess('git', args=[
        'clone',
        'https://github.com/cnrancher/system-charts',
    ], timeout=60)
    u.run_subprocess('git', args=[
        'clone',
        'https://github.com/rancher/charts',
        'rancher-charts',
    ], timeout=60)
    u.run_subprocess('wget', args=[
        'https://releases.rancher.com/kontainer-driver-metadata/release-v2.7/data.json',
        '-O',
        'data.json',
    ], timeout=60)

    # Run hangar generate-list
    u.run_subprocess(u.hangar, args=[
        'generate-list',
        '--rancher=v2.7.0',
        '--chart=pandaria-catalog',
        '--chart=rancher-charts',
        '--system-chart=system-charts',
        '--kdm=data.json',
        '--output=custom-generated-images.txt',
    ])
    f = open('custom-generated-images.txt', 'r')
    print('custom-generated-images.txt images: ')
    for _ in range(0, 5):
        print(f.readline(), end='')
    print('......')
    f.close()
    os.remove('custom-generated-images.txt')

# Test generate list from custom KDM URL
def test_generate_list_custom_kdm_url():
    print('test_generate_list_custom_kdm_url')
    # Run hangar generate-list
    u.run_subprocess(u.hangar, args=[
        'generate-list',
        '--rancher=v2.7.0',
        '--kdm=https://releases.rancher.com/kontainer-driver-metadata/dev-v2.7/data.json',
        '--output=custom-kdm-images.txt',
    ])
    f = open('custom-kdm-images.txt', 'r')
    print('custom-kdm-images.txt images: ')
    for _ in range(0, 5):
        print(f.readline(), end='')
    print('......')
    f.close()
    os.remove('custom-kdm-images.txt')
