"""Acquire bounded pinned evaluation inputs; no redistribution or user media."""
import hashlib,sys,urllib.request
from pathlib import Path
from prepare_quality import ARCHIVE_URL,ARCHIVE_SHA256
root=Path(sys.argv[1]);root.mkdir(parents=True,exist_ok=True)
inputs={
 'att_faces.zip':(ARCHIVE_URL,ARCHIVE_SHA256,8*1024*1024),
 'coffee.png':('https://raw.githubusercontent.com/scikit-image/scikit-image/e8a42ba85aaf5fd9322ef9ca51bc21063b22fcae/skimage/data/coffee.png','cc02f8ca188b167c775a7101b5d767d1e71792cf762c33d6fa15a4599b5a8de7',2*1024*1024),
}
for name,(url,digest,limit) in inputs.items():
 with urllib.request.urlopen(url,timeout=60) as response:data=response.read(limit+1)
 assert len(data)<=limit and hashlib.sha256(data).hexdigest()==digest,'native quality input integrity failed'
 (root/name).write_bytes(data)
print('PASS: pinned ORL research archive and public natural no-face control verified; credit AT&T Laboratories Cambridge / Olivetti Research Laboratory; no portraits/embeddings will be uploaded')
