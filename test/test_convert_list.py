#!/usr/bin/env python3

import utils as u
import os

def test_convert_list_help():
    print('test_convert_list_help')
    u.run_subprocess(u.hangar, ['convert-list', '-h'])
    u.run_subprocess(u.hangar, ['convert-list', '--help'])

def test_convert_list():
    # Convert list
    # Set source registry to docker.io
    # Set destination registry to env DEST_REGISTRY
    print('test_convert_list')
    u.run_subprocess(u.hangar, args=[
        'convert-list',
        '-i', './data/save_test.txt',
        '-o', './converted.txt',
        '-s', 'docker.io',
        '-d', os.environ['DEST_REGISTRY'],
    ])
    f = open('converted.txt', 'r')
    cv = f.read()
    print(cv)
    f.close()
    os.remove('converted.txt')
