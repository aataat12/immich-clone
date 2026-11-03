"""Create controlled public-domain face media; no user content and no runtime dependency."""
import hashlib,json,sys,urllib.request
from pathlib import Path
from PIL import Image
import pillow_heif
pillow_heif.register_heif_opener()
r=Path(sys.argv[1]);r.mkdir(parents=True,exist_ok=True)
base='https://raw.githubusercontent.com/scikit-image/scikit-image/e8a42ba85aaf5fd9322ef9ca51bc21063b22fcae/skimage/data/'
for name,sha in [('astronaut.png','88431cd9653ccd539741b555fb0a46b61558b301d4110412b5bc28b5e3ea6cb5'),('coffee.png','cc02f8ca188b167c775a7101b5d767d1e71792cf762c33d6fa15a4599b5a8de7')]:
 b=urllib.request.urlopen(base+name,timeout=60).read();assert hashlib.sha256(b).hexdigest()==sha;(r/name).write_bytes(b)
a=Image.open(r/'astronaut.png').convert('RGB')
a.save(r/'face.heic',quality=95)
large=Image.new('RGB',(4032,3024));large.paste(a.resize((3024,3024)),(504,0))
large.save(r/'face-grid-reference.png');large.save(r/'face-grid.heic',quality=90,tile_size=512)
for name,size in [('face.heic',(512,512)),('face-grid.heic',(4032,3024))]:
 with Image.open(r/name) as check:assert check.size==size;check.load()
frames=r/'frames';frames.mkdir(exist_ok=True)
c=Image.open(r/'coffee.png').convert('RGB').resize((512,512))
for i in range(32):(a if i<16 else c).save(frames/('%03d.png'%i))
(r/'bad.heic').write_bytes(b'not a HEIC image')
print('PASS: pinned public face inputs and single/grid HEIC fixtures encoded and read back')
