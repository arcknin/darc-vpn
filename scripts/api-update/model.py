#!/usr/bin/env python3
"""Preserve the resolved private runtime model; change only the API image."""
import copy
import json
import os
import pathlib
import re
import stat
import sys


def data_source(model):
    api = model['services']['api']
    if api.get('environment', {}).get('VPN_DB_PATH') != '/data/vpn.db':
        raise ValueError('Unsupported database location')
    mounts = [v for v in api.get('volumes', [])
              if isinstance(v, dict) and v.get('target') == '/data' and v.get('type') == 'bind']
    if len(mounts) != 1:
        raise ValueError('Persistent bind data volume required')
    source = mounts[0].get('source', '')
    if not pathlib.Path(source).is_absolute() or '..' in pathlib.Path(source).parts:
        raise ValueError('Absolute data volume required')
    return source


def prepare(model, image):
    if not re.fullmatch(r'sha256:[a-f0-9]{64}', image):
        raise ValueError('Pinned image required')
    if set(model.get('services', {})) != {'api', 'wireguard'}:
        raise ValueError('Unexpected stack services')
    if model['services']['api'].get('network_mode') != 'service:wireguard':
        raise ValueError('WireGuard namespace required')
    data_source(model)
    result = copy.deepcopy(model)
    result['services']['api']['image'] = image
    return result


def main():
    source, image, target = sys.argv[1:]
    path = pathlib.Path(source)
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != 0 or info.st_mode & 0o077:
        raise ValueError('Runtime model must be a private root-owned regular file')
    original = json.loads(path.read_text())
    updated = prepare(original, image)
    directory = pathlib.Path(data_source(updated))
    if directory.is_symlink() or not directory.is_dir():
        raise ValueError('Persistent data directory unavailable')
    if not (directory / 'vpn.db').is_file() or not (directory / 'master.hex').is_file():
        raise ValueError('Existing database and master key required')
    fd = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, 'w') as output:
        json.dump(updated, output, indent=2)
        output.write('\n')
    print(directory)


if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError):
        raise SystemExit('Runtime model/data validation failed; no private configuration printed.')
