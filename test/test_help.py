#!/usr/bin/env python3

import utils as u

def test_help():
    print('test_help')
    # hangar help
    u.run_subprocess(u.hangar, ['help'])
    # hangar -h
    u.run_subprocess(u.hangar, ['-h'])
    # hangar --help
    u.run_subprocess(u.hangar, ['--help'])
