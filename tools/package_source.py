#!/usr/bin/env python3
"""Reproducible source ZIP with internal source hashes; no network or writes to source."""
import hashlib
import sys
import zipfile
from pathlib import Path

EXCLUDE_DIRS={'.git','__pycache__','.idea','.vscode','.local','.private','bin','dist'}
EXCLUDE_SUFFIXES={'.zip','.pyc','.log','.sqlite','.sqlite3','.db','.out','.prof','.pprof','.pem','.key','.secret'}
EXCLUDE_NAMES={'SHA256SUMS','gen-type-go','phasea-proof'}

def package(root: Path, output: Path):
    root=root.resolve();output=output.resolve()
    files={}
    for path in sorted(root.rglob('*')):
        if not path.is_file() or path.is_symlink():
            continue
        rel=path.relative_to(root)
        if any(x in EXCLUDE_DIRS for x in rel.parts) or path.suffix in EXCLUDE_SUFFIXES or path.name in EXCLUDE_NAMES or any(x.startswith('.env') for x in rel.parts):
            continue
        if path.resolve()==output:continue
        files[rel.as_posix()]=path.read_bytes()
    hashes=''.join(f'{hashlib.sha256(data).hexdigest()}  {name}\n' for name,data in sorted(files.items()))
    files['SHA256SUMS']=hashes.encode('utf-8')
    output.parent.mkdir(parents=True,exist_ok=True)
    with zipfile.ZipFile(output,'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as z:
        for name,data in sorted(files.items()):
            info=zipfile.ZipInfo(name,(1980,1,1,0,0,0))
            info.create_system=3;info.external_attr=(0o100644<<16)
            info.compress_type=zipfile.ZIP_DEFLATED
            z.writestr(info,data,compress_type=zipfile.ZIP_DEFLATED,compresslevel=9)
    b=output.read_bytes()
    print(f'{hashlib.sha256(b).hexdigest()}  {output.name}')
    print(f'files={len(files)} bytes={len(b)} (including internal SHA256SUMS)')

if __name__=='__main__':
    if len(sys.argv)!=2:raise SystemExit('usage: python3 tools/package_source.py OUTPUT.zip')
    package(Path(__file__).resolve().parent.parent,Path(sys.argv[1]))
