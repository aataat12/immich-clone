"""Acquire official pinned PICS research inputs; aggregate evaluation only."""
import hashlib
import sys
import urllib.error
import urllib.request
import zipfile
from pathlib import Path

ARCHIVE_URL = "https://pics.stir.ac.uk/zips/nottingham_originals.zip"
ARCHIVE_SHA256 = "b97fecf7c3e8250518a190aa265e617fbd916e14a2bbbcd79a58c65bf1da190e"
KNOWN = [3, 8, 11, 23, 34, 37, 39, 44, 48, 50, 51, 52, 53, 56, 58, 59, 60, 62, 63, 64, 66, 67, 70, 71, 73, 75, 76, 81, 85, 90, 91, 92, 93, 95, 99, 100]
UNKNOWN = [1, 2, 4, 5, 10, 13, 17, 25, 30, 32, 36, 38, 41, 42, 43, 45, 47, 49, 54, 55, 57, 65, 68, 69, 72, 74, 77, 78, 80, 83, 84, 86, 88, 94, 96]
root = Path(sys.argv[1])
root.mkdir(parents=True, exist_ok=True)
inputs = {
    "nottingham-originals.zip": (ARCHIVE_URL, ARCHIVE_SHA256, 64*1024*1024, 38347802),
    "coffee.png": (
        "https://raw.githubusercontent.com/scikit-image/scikit-image/e8a42ba85aaf5fd9322ef9ca51bc21063b22fcae/skimage/data/coffee.png",
        "cc02f8ca188b167c775a7101b5d767d1e71792cf762c33d6fa15a4599b5a8de7",
        2*1024*1024, None,
    ),
}
for name, (url, digest, limit, exact_size) in inputs.items():
    partial, target = root/(name+".part"), root/name
    try:
        total, sha = 0, hashlib.sha256()
        request = urllib.request.Request(url, headers={"User-Agent":"ImmichClone-QualityResearch/0.1.43 (public benchmark research; aggregate metrics only)"})
        with urllib.request.urlopen(request, timeout=60) as response, partial.open("xb") as output:
            while chunk := response.read(1024*1024):
                total += len(chunk)
                if total > limit:
                    raise RuntimeError("Research input size bound exceeded")
                sha.update(chunk)
                output.write(chunk)
        if sha.hexdigest() != digest or exact_size is not None and total != exact_size:
            raise RuntimeError("Research input integrity mismatch")
        partial.replace(target)
    except urllib.error.HTTPError as error:
        raise SystemExit(f"Official pinned research input refused access (HTTP {error.code}); no fallback or inference attempted") from None
    finally:
        partial.unlink(missing_ok=True)
expected = {f"m{subject}-{configuration}.gif" for subject in KNOWN+UNKNOWN for configuration in range(1,8)}
expected -= {"m5-4.gif", "m5-5.gif", "m13-4.gif"}
expected.add("m39-8.gif")
with zipfile.ZipFile(root/"nottingham-originals.zip") as archive:
    files = archive.infolist()
    if len(files)!=495 or {entry.filename for entry in files}!=expected:
        raise RuntimeError("Nottingham original inventory mismatch")
    if sum(entry.file_size for entry in files)!=38832533 or any(entry.file_size>512*1024 for entry in files):
        raise RuntimeError("Nottingham entry bounds mismatch")
print("PASS: official pinned PICS/University of Stirling Nottingham originals and natural control verified; aggregate research metrics only")
