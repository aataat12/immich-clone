"""CI-only independent Pillow fixtures. No user files or runtime dependencies."""
import hashlib,json,sys,urllib.request,os
from pathlib import Path
from PIL import Image,ImageOps

root=Path(sys.argv[1]);root.mkdir(parents=True,exist_ok=True)
fixtures=[]
def record(name, camera=False):
    p=root/name
    with Image.open(p) as raw:
        raw.seek(0)
        exif=raw.getexif()
        sub=exif.get_ifd(0x8769) if 0x8769 in exif else {}
        date=(sub.get(36867) or exif.get(306) or '').strip('\x00 ')
        photo={'orientation':int(exif.get(274,1)),'make':str(exif.get(271,'' )).strip('\x00 '),'model':str(exif.get(272,'')).strip('\x00 '),'captured':date.replace(':','-',2).replace(' ','T')}
        oriented=ImageOps.exif_transpose(raw) if name.lower().endswith(('.jpg','.jpeg')) else raw.copy()
        oriented=oriented.convert('RGBA');w,h=oriented.size
        pixels=[list(oriented.getpixel((int(w*x),int(h*y)))) for x,y in [(.25,.25),(.75,.25),(.25,.75),(.75,.75)]]
    fixtures.append({'name':name,'sha256':hashlib.sha256(p.read_bytes()).hexdigest(),'width':w,'height':h,'photo':photo if name.lower().endswith(('.jpg','.jpeg')) else None,'pixels':pixels,'tolerance':12 if name.lower().endswith(('.jpg','.jpeg')) else 3,'camera':camera})

def save(name, image, **kwargs):
    image.save(root/name,**kwargs);record(name)

im=Image.new('RGB',(160,120))
for box,c in [((0,0,80,60),(240,30,20)),((80,0,160,60),(20,230,30)),((0,60,80,120),(20,30,240)),((80,60,160,120),(240,240,240))]:im.paste(c,box)
for orientation in range(1,9):
    e=Image.Exif();e[274]=orientation;e[271]='Synthetic camera';e[272]='Quadrants';e[306]='2000:01:01 00:00:00';e[0x8769]={36867:'2024:03:02 10:11:12'}
    save(f'orientation-{orientation}.jpg',im,quality=95,exif=e)
save('progressive.jpg',im,quality=95,progressive=True)
save('cmyk.jpg',im.convert('CMYK'),quality=95)
save('rgb.png',im)
rgba=im.convert('RGBA');rgba.putalpha(128);save('alpha.png',rgba)
save('palette.png',im.convert('P',palette=Image.Palette.ADAPTIVE,colors=4))
save('gray.png',im.convert('L'))
save('24bit.bmp',im)
save('animated.gif',im,save_all=True,append_images=[Image.new('RGB',im.size,'black')],duration=100,loop=0)
for compression in ['raw','tiff_lzw','tiff_adobe_deflate','packbits']:
    save('first-'+compression+'.tiff',im,compression=compression,save_all=True,append_images=[Image.new('RGB',im.size,'black')])
save('12mp.jpg',im.resize((4032,3024)),quality=85)

# These are public reduced camera samples retaining original camera EXIF, not
# claims of a full-resolution camera corpus. Download only; do not redistribute.
sha='5332a9cf0220e9c6c93f88a187daa90808051e10'
cameras={
 'Canon_40D.jpg':'6bfdabd4fc33d112283c147acccc574e770bbe6fbdbc3d4da968ba7b606ecc2f',
 'Nikon_D70.jpg':'8e2a627b96ca71c20129161f46bda3d338407da99bd11b1055adb27af27d7ef5',
 'Pentax_K10D.jpg':'146601c9d406410abdaa832508ee4ccddbc7ad54530e81d57962c1b7728e2e6d',
}
for name,expected in cameras.items():
    cache=os.environ.get('IMMICH_CAMERA_SOURCE')
    b=(Path(cache)/name).read_bytes() if cache else urllib.request.urlopen(f'https://raw.githubusercontent.com/ianare/exif-samples/{sha}/jpg/{name}',timeout=45).read()
    assert len(b)<100000 and hashlib.sha256(b).hexdigest()==expected
    (root/name).write_bytes(b);record(name,True)
(root/'expected.json').write_text(json.dumps(fixtures,indent=2),encoding='utf-8')
print(f'PASS: independent fixture expectations generated for {len(fixtures)} images; {sum(f["camera"] for f in fixtures)} reduced camera EXIF samples')
