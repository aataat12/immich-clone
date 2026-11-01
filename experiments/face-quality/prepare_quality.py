"""Acquire original research archive and reuse previously pinned face model IDs."""
import hashlib
from pathlib import Path
import sys
import urllib.request

sys.path.insert(0, str(Path(__file__).resolve().parents[1]/'ai-probe'))
from prepare import FILES
ARCHIVE_URL = 'https://www.cl.cam.ac.uk/research/dtg/attarchive/pub/data/att_faces.zip'
ARCHIVE_SHA256 = '509d763bd2276aa054d260d7f1000e8be212738134f2f83d8ff13bc2adf1c8ec'

def inputs():
    return {name: FILES[name] for name in ['yunet.onnx', 'sface.onnx']} | {'att_faces.zip': (ARCHIVE_URL, ARCHIVE_SHA256)}

def verify(root):
    for name, (_, digest) in inputs().items():
        assert hashlib.sha256((root/name).read_bytes()).hexdigest() == digest, 'input integrity failed'

if __name__ == '__main__':
    root = Path(sys.argv[1]);root.mkdir(parents=True, exist_ok=True)
    for name,(url,digest) in inputs().items():
        with urllib.request.urlopen(url,timeout=60) as response:
            data = response.read(45*1024*1024)
        assert hashlib.sha256(data).hexdigest() == digest, 'input download integrity failed'
        (root/name).write_bytes(data)
    verify(root)
    print('Three pinned public research inputs verified; credit AT&T Laboratories Cambridge / Olivetti Research Laboratory')
