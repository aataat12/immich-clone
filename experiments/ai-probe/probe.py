"""Real CPU inference only. A tiny feasibility set, NOT an accuracy benchmark."""
import sys
from pathlib import Path
# Explicit candidate package directory works under a service's restricted environment.
root = Path(sys.argv[1])
sys.path.insert(0, str(root/'deps'))
import gc
import json
import platform
import time
import cv2 as cv
import numpy as np
from prepare import verify

verify(root)
assert cv.__version__ == '4.12.0' and np.__version__ == '2.2.6'
cv.setNumThreads(1)
cv.ocl.setUseOpenCL(False)
started = time.monotonic()
# YuNet consumes BGR input and emits box + five landmarks + confidence.
detector = cv.FaceDetectorYN.create(str(root/'yunet.onnx'), '', (512, 512), 0.9, 0.3, 5000,
                                  cv.dnn.DNN_BACKEND_OPENCV, cv.dnn.DNN_TARGET_CPU)
recognizer = cv.FaceRecognizerSF.create(str(root/'sface.onnx'), '',
                                      cv.dnn.DNN_BACKEND_OPENCV, cv.dnn.DNN_TARGET_CPU)

def load(name):
    image = cv.imread(str(root/name))
    assert image is not None and image.size, 'fixture decode'
    return image

def embedding(image):
    detector.setInputSize((image.shape[1], image.shape[0]))
    _, faces = detector.detect(image)
    assert faces is not None and len(faces) == 1, 'expected exactly one face'
    aligned = recognizer.alignCrop(image, faces[0])
    assert aligned.shape == (112, 112, 3), 'five-landmark alignment'
    vector = recognizer.feature(aligned).copy()
    assert vector.shape == (1, 128) and np.isfinite(vector).all(), 'embedding bounds'
    vector /= np.linalg.norm(vector)
    return vector

a = load('astronaut.png'); b = load('grace_hopper.jpg')
va = embedding(a); vb = embedding(b)
# This is a transformed SAME PHOTO, not independent cross-photo identity evidence.
transformed = cv.convertScaleAbs(cv.resize(a, (448, 448)), alpha=0.8, beta=15)
vt = embedding(transformed)
same = (va @ vt.T).item(); different = (va @ vb.T).item()
assert same > 0.363 and different < 0.363 and same > different + 0.2, 'identity separation'
for name in ['coffee.png', 'chelsea.png']:
    image = load(name); detector.setInputSize((image.shape[1], image.shape[0]))
    _, faces = detector.detect(image)
    assert faces is None or len(faces) == 0, 'non-face negative'
# Small grouping proof with saved vectors; names come from the user, not the model.
vectors = np.concatenate([va, vt, vb]); similarity = vectors @ vectors.T
assert similarity[0, 1] > 0.363 and similarity[0, 2] < 0.363 and similarity[1, 2] < 0.363
face_ms = round((time.monotonic()-started)*1000, 1)
del detector, recognizer; gc.collect()

# YOLOX: RGB raw 0..255 float32, upper-left letterbox 114, NCHW; no /255.
# Decode 8/16/32 grid outputs; class-aware NMS as in pinned upstream demo.
net = cv.dnn.readNet(str(root/'yolox.onnx'))
net.setPreferableBackend(cv.dnn.DNN_BACKEND_OPENCV); net.setPreferableTarget(cv.dnn.DNN_TARGET_CPU)
grids = []; strides = []
for stride in [8, 16, 32]:
    x, y = np.meshgrid(np.arange(640//stride), np.arange(640//stride))
    grid = np.stack((x, y), axis=2).reshape(-1, 2)
    grids.append(grid); strides.append(np.full((len(grid), 1), stride))
grid = np.concatenate(grids); stride = np.concatenate(strides)
object_results = []
for name, expected in [('astronaut.png', 0), ('chelsea.png', 15), ('coffee.png', 41)]:
    image = cv.cvtColor(load(name), cv.COLOR_BGR2RGB)
    ratio = min(640/image.shape[0], 640/image.shape[1])
    w, h = int(image.shape[1]*ratio), int(image.shape[0]*ratio)
    padded = np.full((640, 640, 3), 114, dtype=np.float32)
    padded[:h, :w] = cv.resize(image, (w, h))
    start = time.monotonic()
    net.setInput(padded.transpose(2, 0, 1)[None])
    output = net.forward()[0]
    assert output.shape == (8400, 85) and np.isfinite(output).all(), 'object tensor'
    centers = (output[:, :2] + grid)*stride
    sizes = np.exp(output[:, 2:4])*stride
    boxes = np.concatenate([centers-sizes/2, sizes], axis=1)
    scores = output[:, 4:5]*output[:, 5:]
    classes = np.argmax(scores, axis=1); confidence = scores.max(axis=1)
    keep = cv.dnn.NMSBoxesBatched(boxes.tolist(), confidence.tolist(), classes.tolist(), 0.5, 0.5)
    detected = sorted(set(int(classes[i]) for i in keep))
    assert expected in detected, name+' expected object missing: '+str(detected)
    assert np.all(sizes[keep] > 0), 'invalid box size'
    object_results.append({'fixture': name, 'expected_coco_class': expected,
                           'detected_classes': detected, 'inference_ms': round((time.monotonic()-start)*1000, 1)})
verify(root)  # Neither media nor model files may be rewritten by inference.
print(json.dumps({'scope':'tiny public-fixture CPU feasibility; not general recognition accuracy',
                  'python':platform.python_version(), 'opencv':cv.__version__, 'numpy':np.__version__,
                  'same_photo_transformed_cosine':round(same, 6), 'different_people_cosine':round(different, 6),
                  'face_pipeline_ms':face_ms, 'objects':object_results, 'originals_unchanged':True}))
