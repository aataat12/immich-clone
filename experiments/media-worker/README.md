# Bounded media worker experiment

No component is linked/bundled into ImmichClone. Last delivered installer remains
0.1.7. Acceptance: source-built CPU decoding of synthetic H264, HEVC, rotated MOV
and one 8-bit single-image HEIC; decoded pixels/dimensions; BMP pipe output bounded
by the Go wrapper and PNG generation; original hashes; malformed rejection;
network disabled; native DLL import inventory and LocalService restart on Server.
This does not prove Windows 10, camera corpora, grid/burst HEIC, HDR, metadata,
encoding/transcoding, interruption/time-limit containment or production licensing.

Pinned FFmpeg 8.1.3 upstream mirror commit:
`1041abdc962f4cc4f394aa8de9dc5236c0c3b9e7` (tag n8.1.3).
Source is archived unmodified from that exact Git object. Build command, compiler
versions/copyright, LGPL text, source and binary hashes are retained together.
No GPL/nonfree flags, external codec libraries, network protocols, GPU code or
x86 assembly. CPU baseline x86-64/SSE2; Windows target macro 0x0A00. DLL imports
must be OS-supplied; UCRT/VC/GPU SDK installation is not assumed. LGPL and compiler
runtime license verification, update/security policy, actual dependency closure,
process/memory containment and installer integration are separate acceptance gates.
This experiment is not offered as an installable release.

Fixtures are generated solid red frames only, not user media. Base64 and SHA256
allow direct review and checksum verification before execution. Generator used
cloud Linux FFmpeg 7.1.5 (fixture producer only):

```sh
ffmpeg -f lavfi -i color=red:size=160x120:rate=5 -t 1 -c:v libx264 -threads 1 -pix_fmt yuv420p h264.mp4
ffmpeg -f lavfi -i color=red:size=160x120:rate=5 -t 1 -c:v libx265 -threads 1 -x265-params pools=none:frame-threads=1 -pix_fmt yuv420p hevc.mp4
ffmpeg -display_rotation:v:0 90 -i h264.mp4 -c copy rotated.mov
```

HEIC producer used Pillow 12.0.0 and pillow-heif 1.8.1, CI/development only:

```python
from PIL import Image
import pillow_heif
pillow_heif.register_heif_opener()
Image.new('RGB', (160,120), (255,0,0)).save('synthetic.heic', quality=95)
```

Synthetic fixture content may be used, modified and redistributed without
restriction. Encoder libraries are not shipped with the application or candidate.
The custom build removes the prior large third-party binary's transitive source
acquisition ambiguity; successful execution alone does not clear redistribution.
No source-to-binary reproducibility claim until independent rebuild comparison.

Initial run 41344707161 built the candidate and passed H264/HEVC service decode,
but failed the rotated-MOV dimension assertion. Investigation found the fixture
producer silently ignored the old rotate metadata command. Regenerated using
`-display_rotation:v:0 90` and independently confirmed its display matrix and
120x160 thumbnail using the fixture producer. This is a fixture correction, not
evidence that the candidate failed actual rotation. Retest Windows operations
using the unchanged checksum-verified binary artifact from that run.

Source build is manually dispatchable (`minimal-media.yml`); service retests use
`media-service.yml` with a pinned `run-id`. Reuse compiled artifacts only while
unexpired; rebuild pinned source if needed, then update that explicit run ID. No
floating third-party binary or mandatory paid/local build environment is used.
Completed service retest: run 41345139004, commit
`0cb29a1aa25546a4125aea25dbc7a42646e8010e`. Actual timeout termination and
output-cap tests passed, as did an enumeration permitting only file/pipe protocols.
Total process memory containment and broad/grid HEIC remain separate gates; the
FFmpeg per-allocation flag does not establish a total memory cap.
