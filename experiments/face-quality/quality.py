"""Frozen-protocol model screen; outputs aggregate metrics only, no embeddings."""
import sys
from pathlib import Path
root=Path(sys.argv[1]);sys.path.insert(0,str(root/'deps'))
import json
import time
import zipfile
import cv2 as cv
import numpy as np
from prepare_quality import verify
from metrics import freeze_threshold, evaluate

verify(root)
assert cv.__version__=='4.12.0' and np.__version__=='2.2.6'
cv.setNumThreads(1);cv.ocl.setUseOpenCL(False)
detector=cv.FaceDetectorYN.create(str(root/'yunet.onnx'),'',(276,336),0.9,0.3,5000,
                                cv.dnn.DNN_BACKEND_OPENCV,cv.dnn.DNN_TARGET_CPU)
recognizer=cv.FaceRecognizerSF.create(str(root/'sface.onnx'),'',cv.dnn.DNN_BACKEND_OPENCV,cv.dnn.DNN_TARGET_CPU)
start=time.monotonic()

def measure(subjects, archive):
    records=[]
    for subject in subjects:
        for index in range(1,11):
            data=archive.read(f's{subject}/{index}.pgm')
            image=cv.imdecode(np.frombuffer(data,dtype=np.uint8),cv.IMREAD_COLOR)
            assert image is not None and image.shape==(112,92,3),'source image shape'
            image=cv.resize(image,(276,336),interpolation=cv.INTER_LINEAR)
            _,faces=detector.detect(image)
            vector=None
            if faces is not None and len(faces)==1:
                aligned=recognizer.alignCrop(image,faces[0])
                assert aligned.shape==(112,112,3),'alignment'
                vector=recognizer.feature(aligned).reshape(-1).copy()
                assert vector.shape==(128,) and np.isfinite(vector).all(),'embedding'
                norm=np.linalg.norm(vector);assert norm>0
                vector/=norm
            records.append((subject,index,vector))
    return records

with zipfile.ZipFile(root/'att_faces.zip') as archive:
    calibration=measure(range(1,11),archive)
    threshold,maximum=freeze_threshold(calibration)
    # Threshold is fixed before reading ANY held-out images.
    heldout=measure(range(11,41),archive)
result=evaluate(heldout,threshold)
detected=sum(v is not None for _,_,v in calibration+heldout)
quality_pass=detected>=360 and result['known_correct']>=144 and result['known_wrong']==0 and result['unknown_false_accept']==0
verify(root)
print(json.dumps({'scope':'ORL narrow cross-photo screen; not broad photo-library acceptance',
 'credit':'AT&T Laboratories Cambridge / Olivetti Research Laboratory',
 'opencv':cv.__version__,'numpy':np.__version__,'images_total':400,'detected_exactly_one':detected,
 'calibration_detected':sum(v is not None for _,_,v in calibration),
 'heldout_detected':sum(v is not None for _,_,v in heldout),
 'calibration_max_negative':round(maximum,6),'frozen_threshold':threshold,
 'elapsed_seconds':round(time.monotonic()-start,2),'metrics':result,
 'narrow_quality_screen_passed':quality_pass,'originals_unchanged':True}))
