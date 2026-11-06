"""Pinned disposable FEI research inputs; never redistribute portraits/embeddings."""
import hashlib
import sys
import urllib.request
import zipfile
from pathlib import Path

ARCHIVE_URL = "https://fei.edu.br/~cet/originalimages_part1.zip"
ARCHIVE_SHA256 = "1d0b0ebe94278a5677edc450251f91450dadb821d9774edc111ad1be0e7475ed"
ARCHIVE_BYTES = 86857988
root = Path(sys.argv[1])
root.mkdir(parents=True, exist_ok=True)
inputs = {
    "fei-part1.zip": (ARCHIVE_URL, ARCHIVE_SHA256, 96 * 1024 * 1024, ARCHIVE_BYTES),
    "coffee.png": (
        "https://raw.githubusercontent.com/scikit-image/scikit-image/e8a42ba85aaf5fd9322ef9ca51bc21063b22fcae/skimage/data/coffee.png",
        "cc02f8ca188b167c775a7101b5d767d1e71792cf762c33d6fa15a4599b5a8de7",
        2 * 1024 * 1024,
        None,
    ),
}
for name, (url, digest, limit, exact_size) in inputs.items():
    target = root / name
    partial = root / (name + ".part")
    try:
        total, sha = 0, hashlib.sha256()
        with urllib.request.urlopen(url, timeout=60) as response, partial.open("xb") as output:
            while chunk := response.read(1024 * 1024):
                total += len(chunk)
                if total > limit:
                    raise RuntimeError("Research input size bound exceeded")
                sha.update(chunk)
                output.write(chunk)
        if sha.hexdigest() != digest or exact_size is not None and total != exact_size:
            raise RuntimeError("Research input integrity mismatch")
        partial.replace(target)
    finally:
        partial.unlink(missing_ok=True)

with zipfile.ZipFile(root / "fei-part1.zip") as archive:
    expected = {f"{subject}-{index:02}.jpg" for subject in range(1, 51) for index in range(1, 15)}
    files = archive.infolist()
    if len(files) != 700 or {item.filename for item in files} != expected:
        raise RuntimeError("FEI research archive inventory mismatch")
    if sum(item.file_size for item in files) != 88178712 or any(item.file_size > 512 * 1024 for item in files):
        raise RuntimeError("FEI research archive entry bounds mismatch")
print("PASS: pinned FEI original archive and natural control verified; credit Centro Universitario da FEI / Thomaz and Giraldi; aggregate research metrics only")
