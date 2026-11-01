"""Public fixtures/goldens for native parity only; never included in artifacts."""
import hashlib
import json
from pathlib import Path
import struct
import sys
sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'ai-probe'))
from prepare import verify
import cv2 as cv
import numpy as np
root=Path(sys.argv[1]);verify(root)
cv.setNumThreads(1);cv.ocl.setUseOpenCL(False)
images={name:cv.imread(str(root/name)) for name in ['astronaut.png','grace_hopper.jpg','coffee.png','chelsea.png']}
images['dimmed']=cv.convertScaleAbs(cv.resize(images['astronaut.png'],(448,448)),alpha=.8,beta=15)
detector=cv.FaceDetectorYN.create(str(root/'yunet.onnx'),'',(512,512),.9,.3,5000,cv.dnn.DNN_BACKEND_OPENCV,cv.dnn.DNN_TARGET_CPU)
recognizer=cv.FaceRecognizerSF.create(str(root/'sface.onnx'),'',cv.dnn.DNN_BACKEND_OPENCV,cv.dnn.DNN_TARGET_CPU)
golden={};manifest={}
for name,image in images.items():
    assert image is not None
    frame=name+'.bgr';data=b'ICAI'+struct.pack('<II',image.shape[1],image.shape[0])+image.tobytes()
    (root/frame).write_bytes(data)
    detector.setInputSize((image.shape[1],image.shape[0]));_,faces=detector.detect(image)
    vectors=[]
    if faces is not None:
        for face in faces:
            aligned=recognizer.alignCrop(image,face);v=recognizer.feature(aligned).reshape(-1);v=v/np.linalg.norm(v);vectors.append(v.tolist())
    golden[frame]=vectors
for name,data in {'truncated.bgr':b'ICAI'+struct.pack('<II',512,512),
                  'oversized.bgr':b'ICAI'+struct.pack('<II',1601,512),
                  'invalid.bgr':b'NOT A FRAME!','trailing.bgr':(root/'astronaut.png.bgr').read_bytes()+b'\0'}.items():
    (root/name).write_bytes(data)
for p in root.iterdir():
    if p.is_file() and p.suffix in ['.exe','.onnx','.bgr','.png','.jpg']:
        manifest[p.name]=hashlib.sha256(p.read_bytes()).hexdigest()
(root/'golden.json').write_text(json.dumps(golden))
(root/'input-hashes.json').write_text(json.dumps(manifest))
print('Native working frames and private parity goldens prepared; no raw embeddings emitted')
