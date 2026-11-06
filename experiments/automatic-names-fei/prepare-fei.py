"""Pinned disposable FEI research inputs; never redistribute portraits/embeddings."""
import hashlib
import http.cookiejar
import sys
import urllib.error
import urllib.request
import zipfile
from pathlib import Path

ARCHIVE_URL = "https://fei.edu.br/~cet/originalimages_part1.zip"
ARCHIVE_SHA256 = "1d0b0ebe94278a5677edc450251f91450dadb821d9774edc111ad1be0e7475ed"
ARCHIVE_BYTES = 86857988
SOURCE_PAGE = "https://fei.edu.br/~cet/facedatabase.html"
RESEARCH_USER_AGENT = "ImmichClone-QualityResearch/0.1.43 (public FEI benchmark evaluation; no image redistribution)"
root = Path(sys.argv[1])
root.mkdir(parents=True, exist_ok=True)
# Follow the publisher's normal public page-to-download flow. Accept only its
# anonymous session cookies; do not spoof a browser, bypass a challenge or retry
# a refused request through another host, identity, proxy or mirror.
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
page_request = urllib.request.Request(SOURCE_PAGE, headers={"User-Agent": RESEARCH_USER_AGENT})
try:
    with opener.open(page_request, timeout=60) as response:
        page = response.read(256 * 1024 + 1)
        if response.status != 200 or len(page) > 256 * 1024 or b"originalimages_part1.zip" not in page:
            raise SystemExit("FEI official public download page unavailable or unexpected; no inference attempted")
except urllib.error.HTTPError as error:
    raise SystemExit(f"FEI official public page refused access (HTTP {error.code}); no fallback or inference attempted") from None
print("FEI official public download page accepted; following its published archive link using the same anonymous session")
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
        headers = {"User-Agent": RESEARCH_USER_AGENT}
        if url == ARCHIVE_URL:
            headers["Referer"] = SOURCE_PAGE
        request = urllib.request.Request(url, headers=headers)
        with opener.open(request, timeout=60) as response, partial.open("xb") as output:
            while chunk := response.read(1024 * 1024):
                total += len(chunk)
                if total > limit:
                    raise RuntimeError("Research input size bound exceeded")
                sha.update(chunk)
                output.write(chunk)
        if sha.hexdigest() != digest or exact_size is not None and total != exact_size:
            raise RuntimeError("Research input integrity mismatch")
        partial.replace(target)
    except urllib.error.HTTPError as error:
        raise SystemExit(f"Pinned research input refused access (HTTP {error.code}); no fallback or inference attempted") from None
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
