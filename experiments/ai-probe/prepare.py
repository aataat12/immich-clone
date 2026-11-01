"""Fetch public, checksum-pinned experiment inputs; no user media or credentials."""
import hashlib
import json
from pathlib import Path
import sys
import urllib.request

ZOO = 'https://media.githubusercontent.com/media/opencv/opencv_zoo/48b10fa505aeb51fbd127fc60a5c660040e823b7/models/'
DATA = 'https://raw.githubusercontent.com/scikit-image/scikit-image/e8a42ba85aaf5fd9322ef9ca51bc21063b22fcae/skimage/data/'
FILES = {
    'yunet.onnx': (ZOO+'face_detection_yunet/face_detection_yunet_2023mar.onnx', '8f2383e4dd3cfbb4553ea8718107fc0423210dc964f9f4280604804ed2552fa4'),
    'sface.onnx': (ZOO+'face_recognition_sface/face_recognition_sface_2021dec.onnx', '0ba9fbfa01b5270c96627c4ef784da859931e02f04419c829e83484087c34e79'),
    'yolox.onnx': (ZOO+'object_detection_yolox/object_detection_yolox_2022nov.onnx', 'c5c2d13e59ae883e6af3b45daea64af4833a4951c92d116ec270d9ddbe998063'),
    'astronaut.png': (DATA+'astronaut.png', '88431cd9653ccd539741b555fb0a46b61558b301d4110412b5bc28b5e3ea6cb5'),
    'chelsea.png': (DATA+'chelsea.png', '596aa1e7cb875eb79f437e310381d26b338a81c2da23439704a73c4651e8c4bb'),
    'coffee.png': (DATA+'coffee.png', 'cc02f8ca188b167c775a7101b5d767d1e71792cf762c33d6fa15a4599b5a8de7'),
    'grace_hopper.jpg': ('https://raw.githubusercontent.com/matplotlib/matplotlib/4aeb773422464799998d900198b35cb80e94b3e1/lib/matplotlib/mpl-data/sample_data/grace_hopper.jpg', 'a8ca6d734765703b09728ab47fe59f473d93ae3967fc24c7c0288c3c7adb7130'),
}

def verify(root):
    for name, (_, digest) in FILES.items():
        assert hashlib.sha256((root/name).read_bytes()).hexdigest() == digest, name+' integrity failed'

if __name__ == '__main__':
    root = Path(sys.argv[1]); root.mkdir(parents=True, exist_ok=True)
    for name, (url, digest) in FILES.items():
        with urllib.request.urlopen(url, timeout=60) as response:
            data = response.read(45*1024*1024)
        assert hashlib.sha256(data).hexdigest() == digest, name+' download integrity failed'
        (root/name).write_bytes(data)
    verify(root)
    print(json.dumps({'verified_public_inputs': len(FILES)}))
