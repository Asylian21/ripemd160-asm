# Apple M5 Pro local optimization evidence

Measured on 2026-10-04 with Go 1.22.5, darwin/arm64, `GOMAXPROCS=1`, six
samples per case, and 200ms benchmark duration. Each comparison uses frozen
test binaries; the SIMD comparisons alternate before/after invocations to
reduce clock/thermal drift. No other agent ran throughput work during these
measurement windows. These are local results for the M5 Pro.

| Change | Case | Before | After | Time delta | benchstat |
| --- | --- | --- | --- | --- | --- |
| Unrolled scalar | scalar, n=2048 | 348.8us | 238.3us | -31.68% | p=0.002, n=6 |
| Shorter boolean sequences | NEON, n=2048 | 123.1us | 112.1us | -8.91% | p=0.002, n=6 |
| SHA3 versus original kernel | n=2048 | 123.1us | 103.9us | -15.59% | p=0.002, n=6 |
| Independent additive terms | NEON, n=2048 | 114.23us | 91.63us | -19.78% | p=0.002, n=6 |
| Independent additive terms | SHA3, n=2048 | 105.59us | 82.78us | -21.60% | p=0.002, n=6 |

The stages above are separate comparisons, not one same-window measurement of
the overall delta. The final kernel reaches 24.74M hashes/s at n=2048, with
zero allocation. Parent application benchmarks measure the whole pipeline
separately. Cases n=4 and n=5 also improved; the raw files retain those samples.

- [scalar-before.txt](scalar-before.txt) and [scalar-after.txt](scalar-after.txt):
  original kernel versus first shorter-boolean/unrolled-scalar candidate.
- [original-simd.txt](original-simd.txt) and [sha3-candidate.txt](sha3-candidate.txt):
  original NEON, shorter NEON, and SHA3 without the final additive reassociation.
- [sha3-comparable.txt](sha3-comparable.txt): the SHA3 subset with its backend
  label normalized to `neon` so benchstat can compare with original-simd.txt.
  All numerical sample values are unchanged.
- [chain-before.txt](chain-before.txt) and [chain-after.txt](chain-after.txt):
  accepted SHA3/NEON candidate before and after the final reassociation.

```sh
benchstat scalar-before.txt scalar-after.txt
benchstat original-simd.txt sha3-comparable.txt
benchstat chain-before.txt chain-after.txt
```

The final source passed the native suite, data-race detector, vet, staticcheck,
and the independent oracle. Library statement coverage was 100%. Ten-second
differential fuzz campaigns passed for Hash32 (162,780 executions), Sum
(1,120,193), and the HASH160 helper (322,265). Those fuzz campaigns preceded the
last algebraically equivalent instruction reordering, whose frozen candidate
and final source both passed the full oracle suite; the final source also
passed the race detector again. Linux amd64, arm64, 386, and ppc64le were
cross-compiled successfully; those binaries were not executed on those CPUs.
