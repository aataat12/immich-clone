#6A face suggestions and filter paint checks

Candidate retains exact inference provenance but permits review-only comparisons
between the three serviced native BGR960 provider IDs listed explicitly in
recognition/compatibility.go. Models/preprocessing are unchanged; actual parity
is recorded in dependency-evidence.md (native, zlib servicing) and opencv-nr01.md.
Unknown/different providers remain separate. Similarity0.363 and ranking margin0.05
are unchanged. No automatic naming, forced reanalysis or provenance rewriting.

Synthetic regression coverage checks older named references against current video
observations, including Unnamed-only filtering, and rejects an unknown provider.
The known-photo same-provider video suggestion regression is retained.

The People body now paints an explicit dark background rather than relying on a
transparent body over the root. Edge tests sample rendered background pixels
before and after repeated checkbox toggles. A passing cloud paint test does not
establish the cause of a compositor/embedded-browser overlay elsewhere.

Windows execution is pending. No new installer, model, binary provider, security
mitigation, private media publication or every-frame Full Scan is part of this
candidate.
