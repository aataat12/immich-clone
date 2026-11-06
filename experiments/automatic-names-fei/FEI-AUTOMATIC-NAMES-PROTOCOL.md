# Independent FEI automatic-name policy evaluation

Frozen before FEI native inference. Policy remains unchanged at cosine0.65,
competitor lead0.15 and two manually confirmed, distinct source originals.
No rerun of ORL is needed; its already reported result is reused separately.

## Source, permitted use and integrity

[FEI's official source](https://fei.edu.br/~cet/facedatabase.html) grants research
use while prohibiting redistribution/reproduction. This is a disposable research
evaluation, not product training or redistribution. Credit: Centro Universitario
da FEI, Carlos Eduardo Thomaz and Gilson Antonio Giraldi; see their2010 article
linked by the source. No portraits, thumbnails, ZIPs, embeddings, per-image
predictions or subject-level outputs enter Git, release packages or CI artifacts.
Only aggregate anonymous metrics are logged. No contact or registration is sent.

Use official HTTPS `https://fei.edu.br/~cet/originalimages_part1.zip` only:
**86857988 bytes**, SHA256
`1d0b0ebe94278a5677edc450251f91450dadb821d9774edc111ad1be0e7475ed`.
This pin was established by downloading and inspecting archive names/lengths
only, without native inference or rendering any portrait. The archive has exactly
700 JPEGs,50 subjects,14 original640x480 color images each;88178712 uncompressed
bytes. No mirror or transformed/aligned/morphed corpus is substituted.

The official corpus describes controlled backgrounds, broad profile rotation
and adult appearance/expression variation. Its author
[tutorial](https://fei.edu.br/~cet/tutorial_reconhecimentofaces.pdf), pp19–20,
describes the numbered acquisition sweep, intermediate left/right query indices
4–7 and frontal non-neutral index12. We infer the11/12 neutral/smile frontal pair
from that acquisition order and the official two-frontal-image description; this
mapping is an explicit inference, not a measured pose-angle annotation.

## Frozen enrollment, queries and execution

- Analyze every original, subject IDs1–50, configurations01–14:700 operations.
- Enroll IDs1–30 using exactly configurations11 and12:60 fixed references. No
  alternate image is chosen on failed detection; no auto output becomes a ref.
- Known queries: every other configuration for IDs1–30:360 queries.
- Unknown queries: IDs31–50, all14 configurations:280 queries. Never enroll them.
- Report central sweep4–7 separately:120 known queries. Outer sweep1–3/8–10:
  180 known queries. Additional configurations13/14:60 known queries. Unknowns
  include all these groups plus their40 frontal11/12 observations.
- Use the existing pinned native provider/model hashes and Go production JPEG
  decode, EXIF orientation and `Provider.Analyze` processing. No manual crops,
  resizing, fabricated landmarks, equalization, GPU, training or new package.
- Exactly one detected face is required; zero/multiple detections remain
  unresolved and in every original denominator. Invoke the candidate matcher
  directly with the same provenance, compatibility and independent-source rules.
- Verify archive before/after, validate its exact entry inventory and bounded
  dimensions, and require empty provider scratch. Run three flat no-face controls
  plus the already pinned public natural coffee control through the same provider.

## Predeclared gates

All conditions must pass without any policy/reference/input adaptation:

1. All60 fixed references detected; at least420/700 originals have exactly one
   detection. This avoids reporting rejection caused merely by pipeline failure.
2. Central sweep: at least108/120 exactly-one-face detections and90/120 correct
   automatic identities. All-known: at least144/360 correct assignments. These
   gates require useful retrieval while allowing uncertainty/extreme pose misses.
3. Unknowns: at least160/280 exactly-one-face detections; **zero false automatic
   identities** across all280 queries, including outer profiles and extra views.
4. **Zero wrong known identities** across all360 queries. Unresolved observations
   stay unnamed. Report subgroup coverage and outcomes even if the gate fails.
5. Three flat and one natural control have no faces; originals unchanged and
   scratch empty. Synthetic durability/provenance/actual publication-hook tests
   and older-writer compatibility are separate required release evidence.

`scripts/Test-FEIAutomaticNames.ps1` runs on the established Windows Server2022
runner. Download limit96MiB; each entry512KiB; no extraction. Native context24min,
Go test26min and job30min bound execution. The corpus/provider temporary directory
is removed even on failure. No image/embedding artifact upload step exists.

## Meaning and limits

FEI has not been used to choose this project's candidate policy. It supplies
independent source/color/pose/configuration evidence beyond ORL. Its first archive
is a fixed subset, not a demographically balanced sample of all200 subjects.
It remains a controlled adult acquisition. It does not establish performance for
children, age progression, family resemblance, every ancestry, severe occlusion
or unconstrained lighting/backgrounds. Similarity is not a confidence probability.
Record actual failed/uncertain cases in aggregate; do not tune to this screen.
Passing supports only the declared reversible prototype behavior and does not
automatically change the disabled production gate or certify Windows10.

Independence is from this project's policy selection and ORL evaluation. Possible
overlap between public FEI and upstream model pretraining is not established.
