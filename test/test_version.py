#!/usr/bin/env python3

import utils as u

def test_version():
    print('test_version')
    # hangar version
    u.run_subprocess(u.hangar, ['version'])
    # hangar -v
    u.run_subprocess(u.hangar, ['-v'])
    # hangar --version
    u.run_subprocess(u.hangar, ['--version'])
