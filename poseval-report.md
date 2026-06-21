# poseval — board photo → FEN quality report

_Generated 2026-06-21 02:02 • 38 images_

Backends: ollama, gemini

## Aggregate (mean per-square accuracy · exact-match rate · n · repaired · errors)

_sq-acc = mean fraction of the 64 squares correct. exact = clean FEN equals truth. repaired = model grid was malformed (rejected by AssembleFEN) and scored best-effort. err = no usable grid at all._

| Category | ollama sq-acc | ollama exact | ollama n | ollama repaired | ollama err | gemini sq-acc | gemini exact | gemini n | gemini repaired | gemini err |
|---|---|---|---|---|---|---|---|---|---|---|
| digital | 45.9% | 0% (0/8) | 8 | 6 | 2 | 87.9% | 12% (1/8) | 8 | 2 | 0 |
| physical | 53.5% | 0% (0/30) | 30 | 30 | 0 | 56.1% | 3% (1/30) | 30 | 10 | 0 |
| overall | 51.9% | 0% (0/38) | 38 | 36 | 2 | 62.8% | 5% (2/38) | 38 | 12 | 0 |

## Per-image breakdown

| Image | Category | ollama sq-acc | ollama result | gemini sq-acc | gemini result |
|---|---|---|---|---|---|
| physical/01.jpg | physical | 45.3% | repaired | 100.0% | exact |
| physical/02.jpg | physical | 43.8% | repaired | 87.5% | partial |
| physical/03.jpg | physical | 42.2% | repaired | 37.5% | repaired |
| physical/04.jpg | physical | 28.1% | repaired | 82.8% | partial |
| physical/05.jpg | physical | 42.2% | repaired | 73.4% | partial |
| physical/06.jpg | physical | 43.8% | repaired | 35.9% | partial |
| physical/07.jpg | physical | 42.2% | repaired | 57.8% | partial |
| physical/08.jpg | physical | 40.6% | repaired | 48.4% | partial |
| physical/09.jpg | physical | 34.4% | repaired | 40.6% | partial |
| physical/10.jpg | physical | 40.6% | repaired | 42.2% | partial |
| physical/11.jpg | physical | 43.8% | repaired | 43.8% | partial |
| physical/12.jpg | physical | 48.4% | repaired | 35.9% | repaired |
| physical/13.jpg | physical | 50.0% | repaired | 29.7% | partial |
| physical/14.jpg | physical | 50.0% | repaired | 34.4% | partial |
| physical/15.jpg | physical | 50.0% | repaired | 39.1% | partial |
| physical/16.jpg | physical | 59.4% | repaired | 45.3% | repaired |
| physical/17.jpg | physical | 56.2% | repaired | 45.3% | repaired |
| physical/18.jpg | physical | 57.8% | repaired | 32.8% | partial |
| physical/19.jpg | physical | 62.5% | repaired | 53.1% | partial |
| physical/20.jpg | physical | 65.6% | repaired | 50.0% | repaired |
| physical/21.jpg | physical | 64.1% | repaired | 64.1% | repaired |
| physical/22.jpg | physical | 64.1% | repaired | 45.3% | partial |
| physical/23.jpg | physical | 60.9% | repaired | 60.9% | repaired |
| physical/24.jpg | physical | 67.2% | repaired | 64.1% | partial |
| physical/25.jpg | physical | 59.4% | repaired | 68.8% | partial |
| physical/26.jpg | physical | 68.8% | repaired | 70.3% | partial |
| physical/27.jpg | physical | 71.9% | repaired | 67.2% | repaired |
| physical/28.jpg | physical | 67.2% | repaired | 71.9% | repaired |
| physical/29.jpg | physical | 64.1% | repaired | 76.6% | repaired |
| physical/30.jpg | physical | 71.9% | repaired | 79.7% | partial |
| digital/01.png | digital | 0.0% | error | 100.0% | exact |
| digital/02.png | digital | 46.9% | repaired | 89.1% | partial |
| digital/03.png | digital | 46.9% | repaired | 82.8% | repaired |
| digital/04.png | digital | 45.3% | repaired | 76.6% | partial |
| digital/05.png | digital | 95.3% | repaired | 96.9% | partial |
| digital/06.png | digital | 46.9% | repaired | 89.1% | partial |
| digital/07.png | digital | 85.9% | repaired | 92.2% | repaired |
| digital/08.png | digital | 0.0% | error | 76.6% | partial |

## Ground truth vs. recognized board field

### ollama

| Image | Truth | Got (board field) | Raw grid (repaired/error) |
|---|---|---|---|
| physical/01.jpg | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` | `1K2B2N/2R3Q1/2P5/2P5/2P5/2P5/2P5/2P5` | `.K..B..N/..R...Q/..P...../..P...../..P...../..P...../..P...../..P.....` |
| physical/02.jpg | `rnbqkbnr/ppp1pppp/8/3p4/1P6/8/P1PPPPPP/RNBQKBNR` | `1K2B2N/2R3Q1/2P5/2N5/2B5/2Q5/2R5/2K5` | `.K..B..N/..R...Q/..P...../..N...../..B...../..Q...../..R...../..K.....` |
| physical/03.jpg | `rn1qkbnr/ppp1pppp/8/3p1b2/1P6/8/PBPPPPPP/RN1QKBNR` | `1K2B2N/2R3Q1/2P5/2N5/2B5/2Q5/2R5/2K5` | `.K..B..N/..R...Q/..P...../..N...../..B...../..Q...../..R...../..K.....` |
| physical/04.jpg | `rn1qkbnr/ppp2ppp/4p3/3p1b2/1P6/5N2/PBPPPPPP/RN1QKB1R` | `1K2N1P1/2B1R1B1/2Q1N1Q1/2R1B1K1/2P1B1P1/2B1B1B1/2B1B1B1/2B1B1B1` | `.K..N.P./..B.R.B./..Q.N.Q./..R.B.K./..P.B.P./..B.B.B./..B.B.B./..B.B.B.` |
| physical/05.jpg | `rn1qkb1r/ppp2ppp/4pn2/3p1b2/1P6/P3PN2/1BPP1PPP/RN1QKB1R` | `1K2B2N/2R3Q1/2P5/2N5/2B5/2Q5/2R5/2k5` | `.K..B..N/..R...Q/..P...../..N...../..B...../..Q...../..R...../..k.....` |
| physical/06.jpg | `r2qkb1r/pp1n1ppp/2p1pn2/3p1b2/1PP5/P3PN2/1B1P1PPP/RN1QKB1R` | `1k2r3/2q4n/2b5/2r4b/8/8/8/8` | `.k..r...p/..q....n/..b...../..r....b/..e....a/..d....c/..f....g/..h....i` |
| physical/07.jpg | `r2qk2r/pp1n1ppp/2pbpn2/3p1b2/1PP5/P1N1PN2/1B1PBPPP/R2QK2R` | `1K2B2N/2R3Q1/2P4N/2N4B/2P5/2N5/2B5/2Q5` | `.K..B..N./..R...Q./..P....N/..N....B./..P...../..N...../..B...../..Q.....` |
| physical/08.jpg | `r2q1rk1/pp1n1pp1/2pbpn1p/3p1b2/1PP5/P1N1PN2/1B1PBPPP/R2Q1RK1` | `1K2B2N/2R3Q1/2P4N/2N4B/2P5/2N5/2B5/2Q5` | `.K..B..N./..R...Q./..P....N/..N....B./..P...../..N...../..B...../..Q.....` |
| physical/09.jpg | `r2q1rk1/1p1n1pp1/p1pbpn1p/3p1b2/1PPP4/P1NBPN2/1B3PPP/R2Q1RK1` | `1K2BQ1R/2P3N1/2P3N1/2P3N1/2P3N1/2P3N1/2P3N1/2P3N1` | `.K..BQ.RN.P./..P...N.BR../..P...N.BR../..P...N.BR../..P...N.BR../..P...N.BR../..P...N.BR../..P...N.BR..` |
| physical/10.jpg | `r4rk1/1p1nqpp1/p1pbpn1p/3P4/1P1P4/P1NbPN2/1B3PPP/R2Q1RK1` | `1K2B2N/2R3Q1/2P5/2N5/2B5/2Q5/2R5/2K5` | `.K..B..N/..R...Q/..P...../..N...../..B...../..Q...../..R...../..K.....` |
| physical/11.jpg | `r4rk1/1p1nqpp1/p1pb1n1p/3p4/1P1P4/P1NQP3/1B1N1PPP/R4RK1` | `1K2B2N/2R3Q1/2P4N/2N4B/2P5/2N5/2B5/2Q5` | `.K..B..N./..R...Q./..P....N/..N....B./..P...../..N...../..B...../..Q.....` |
| physical/12.jpg | `r3r1k1/1p1nqpp1/p1p2n1p/3p4/1P1P4/P1NQP3/1B1N1PPK/R3R3` | `1K2B2N/2R3Q1/2P4N/2P5/2P5/2P5/2P5/2P5` | `.K..B..N./..R...Q./..P....N./..P...../..P...../..P...../..P...../..P.....` |
| physical/13.jpg | `r3r1k1/1p1n1pp1/p1p4p/3p2q1/1P1P1Pn1/P1NQP1K1/1B1N2P1/R3R3` | `1k2r3/2p4b/2q5/2r5/2b5/2n5/2p5/2K5` | `.k..r...n/..p....b/..q...../..r...../..b...../..n...../..p...../..K.....` |
| physical/14.jpg | `r3r1k1/1p1n1pp1/p1p4p/3p2P1/NP1P4/P2nP1K1/1B1N2P1/R3R3` | `1K2B2N/2R3Q1/2P5/2P5/2P5/2P5/2P5/2P5` | `.K..B..N/..R...Q/..P...../..P...../..P...../..P...../..P...../..P.....` |
| physical/15.jpg | `r3r1k1/5pp1/p1p4p/1pPp2P1/1P6/P2nP1K1/1B1N2P1/R3R3` | `1K2B2N/2R3Q1/2P4N/2N3B1/2B3Q1/2N3R1/2P4K/2B3Q1` | `.K..B..N./..R...Q./..P....N./..N...B./..B...Q./..N...R./..P....K./..B...Q.` |
| physical/16.jpg | `r3r1k1/5p2/p1p4p/1pPp4/1P6/P3PNK1/1n4P1/R3R3` | `1K2B2N/2R3Q1/2P4N/2P5/2P5/2P5/2P5/2P5` | `.K..B..N./..R...Q./..P....N./..P...../..P...../..P...../..P...../..P...../..P.....` |
| physical/17.jpg | `r5k1/5p2/p1p3rp/1pPp4/1P1N4/P3P3/1n4PK/R3R3` | `1K2B2N/2R3Q1/2P5/2N5/2B5/2Q5/2R5/2K5` | `.K..B..N/..R...Q/..P...../..N...../..B...../..Q...../..R...../..K.....` |
| physical/18.jpg | `4r1k1/5p2/p1p3rp/1pPp4/PPnN4/4P3/4R1PK/R7` | `1K2B2N/2R3Q1/2P5/2N4B/2Q5/2P5/2N5/2K5` | `.K..B..N/..R...Q/..P...../..N....B/..Q...../..P.....B/..N.....A/..K......` |
| physical/19.jpg | `6k1/5p2/2p3rp/1pPp4/1Pn1r3/4P3/2N1R1PK/R7` | `1K2B2N/2Q3R1/2P5/2N5/2B5/2R5/2Q4k/2K5` | `.K..B..N/..Q...R/..P...../..N...../..B...../..R...../..Q....k/..K.....` |
| physical/20.jpg | `6k1/5p2/2p3rp/1pP5/1P1Nr3/4n3/3R2PK/R7` | `1K2B3/2R3N1/2Q5/2P5/2P5/2P5/2P5/2P5` | `.K..B./..R...N/..Q..../..P...../..P...../..P...../..P...../..P.....` |
| physical/21.jpg | `6k1/5p2/2p3rp/1pP5/1PnNr3/8/6PK/R2R4` | `1K2B2N/2R3Q1/2P5/2N5/2B5/2Q5/2R5/2K5` | `.K..B..N/..R...Q/..P...../..N...../..B...../..Q...../..R...../..K.....` |
| physical/22.jpg | `6k1/5p2/2p3r1/1pP2N1p/1P2r1n1/8/3R2P1/R5K1` | `1K2B2N/2R3Q1/2P5/2P5/2P5/2P5/2P5/2P5` | `.K..B..N/..R...Q/..P...../..P...../..P...../..P...../..P...../..P.....` |
| physical/23.jpg | `6k1/5p2/2p2r2/1pP4N/1r4n1/8/3R2P1/R5K1` | `1K2B2N/2R3Q1/2P4N/2P4B/2P5/2P4B/2P4N/2K4Q` | `.K..B..N./..R...Q./..P....N/..P....B/..P....W/..P....B/..P....N/..K....Q.` |
| physical/24.jpg | `6k1/5p2/2p5/1pr5/1r4n1/6N1/3R2P1/5RK1` | `1K2B2N/2R3Q1/2P5/2P5/2P5/2P5/2P5/2P5` | `.K..B..N/..R...Q/..P...../..P...../..P...../..P...../..P...../..P.....` |
| physical/25.jpg | `6k1/5p2/8/1p1p1N2/1r4n1/8/6P1/5RK1` | `1K2B3/2N3P1/2R1B3/2Q1R3/2N1P3/2N1N3/2N1B3/2N1Q3` | `.K..B./..N...P/..R.B./..Q.R./..N.P./..N.N./..N.B./..N.Q.` |
| physical/26.jpg | `6k1/5p2/8/1p3N2/1r4n1/3p4/6P1/3R2K1` | `1K2B3/2R3N1/2Q5/2P5/2P5/2P5/2P5/2P5` | `.K..B./..R...N/..Q..../..P...../..P...../..P...../..P...../..P.....` |
| physical/27.jpg | `6k1/5p2/8/8/1pN3n1/3p4/1r4P1/3R2K1` | `1K2B3/2N3P1/2R5/2Q5/2N5/2B5/2R5/2Q5` | `.K..B./..N...P/..R..../..Q...../..N...../..B...../..R...../..Q.....` |
| physical/28.jpg | `6k1/5p2/8/8/1p6/3pn3/2rN2P1/4R1K1` | `1K2B2N/2R3Q1/2P5/2N5/2B5/2Q4k/2R5/2K5` | `.K..B..N/..R...Q/..P...../..N...../..B...../..Q....k/..R.....U/..K......` |
| physical/29.jpg | `4R1k1/5p2/8/8/8/1p1p4/3r2P1/6K1` | `1K2B3/2N1P3/2R1B3/2Q1R3/2P1N3/2N1Q3/2B1K3/2P1K3` | `.K..B./..N.P./..R.B./..Q.R./..P.N./..N.Q./..B.K./..P.K.` |
| physical/30.jpg | `R7/5pk1/8/8/8/3p4/1p1r2P1/6K1` | `1k2r3/2p4b/2q5/2r5/2b5/2n5/2K5/2Q5` | `.k..r...n/..p....b/..q...../..r...../..b...../..n...../..K...../..Q.....` |
| digital/01.png | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` | `ERROR: recognize: decode model json: unexpected end of JSON input` | `` |
| digital/02.png | `r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R` | `1K2N3/2B3R1/2Q5/2P5/8/8/8/8` | `.K..N../..B...R/..Q..../..P...../..H..../..C..../..D..../..E....` |
| digital/03.png | `r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQ1RK1` | `1K2N3/2B3R1/2Q5/8/8/2P5/8/8` | `.K..N../..B...R/..Q..../..C..../..H..../..P..../..E..../..D....` |
| digital/04.png | `rnbqkb1r/pp1ppppp/5n2/2p5/4P3/2N5/PPPP1PPP/R1BQKBNR` | `1k2p3/2q3r1/2b4n/8/8/8/8/8` | `.k..p./..q...r/..b....n/..h...../..g...../..f...../..e...../..d.....` |
| digital/05.png | `8/8/8/8/8/5k2/6q1/7K` | `8/8/8/8/8/8/8/8` | `./../../../../../../..` |
| digital/06.png | `r3k2r/pppq1ppp/2np1n2/2b1p3/2B1P1b1/2NP1N2/PPPQ1PPP/R3K2R` | `1k2p3/2q3r1/2b4n/8/8/8/8/6b1` | `.k..p./..q...r/..b....n/..h...../..g....m/..f....e/..d....c/..a....b` |
| digital/07.png | `6k1/5ppp/8/8/8/8/5PPP/R5K1` | `8/8/8/8/8/8/8/8` | `./../../../../../../..` |
| digital/08.png | `2kr3r/ppp2ppp/2n5/3qp3/8/2PP4/PP3PPP/RNBQ1RK1` | `ERROR: recognize: decode model json: unexpected end of JSON input` | `` |

### gemini

| Image | Truth | Got (board field) | Raw grid (repaired/error) |
|---|---|---|---|
| physical/01.jpg | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| physical/02.jpg | `rnbqkbnr/ppp1pppp/8/3p4/1P6/8/P1PPPPPP/RNBQKBNR` | `rnbqkbnr/pppp1ppp/4p3/4P3/8/8/PPPP1PPP/RNBQKBNR` |  |
| physical/03.jpg | `rn1qkbnr/ppp1pppp/8/3p1b2/1P6/8/PBPPPPPP/RN1QKBNR` | `RKBQKB1R/PPPP1PPP/2N1P3/3P4/4b1p1/4N1P1/pp1pp1P1/r1bqk1nr` | `RKBQKB.R/PPPP.PPP/..N.P.../...P..../....b.p./....N.P./pp.pp.P./r.bqk.nr` |
| physical/04.jpg | `rn1qkbnr/ppp2ppp/4p3/3p1b2/1P6/5N2/PBPPPPPP/RN1QKB1R` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| physical/05.jpg | `rn1qkb1r/ppp2ppp/4pn2/3p1b2/1P6/P3PN2/1BPP1PPP/RN1QKB1R` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| physical/06.jpg | `r2qkb1r/pp1n1ppp/2p1pn2/3p1b2/1PP5/P3PN2/1B1P1PPP/RN1QKB1R` | `R1BQKBNR/PPPPPPPP/8/8/8/8/pppppppp/rnbqkbnr` |  |
| physical/07.jpg | `r2qk2r/pp1n1ppp/2pbpn2/3p1b2/1PP5/P1N1PN2/1B1PBPPP/R2QK2R` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| physical/08.jpg | `r2q1rk1/pp1n1pp1/2pbpn1p/3p1b2/1PP5/P1N1PN2/1B1PBPPP/R2Q1RK1` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| physical/09.jpg | `r2q1rk1/1p1n1pp1/p1pbpn1p/3p1b2/1PPP4/P1NBPN2/1B3PPP/R2Q1RK1` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| physical/10.jpg | `r4rk1/1p1nqpp1/p1pbpn1p/3P4/1P1P4/P1NbPN2/1B3PPP/R2Q1RK1` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| physical/11.jpg | `r4rk1/1p1nqpp1/p1pb1n1p/3p4/1P1P4/P1NQP3/1B1N1PPP/R4RK1` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| physical/12.jpg | `r3r1k1/1p1nqpp1/p1p2n1p/3p4/1P1P4/P1NQP3/1B1N1PPK/R3R3` | `1P1k1r1R/2p1p1p1/2p1p1p1/2p1p1p1/2p1p1p1/2p1p1p1/2p1p1p1/R1k1r1R1` | `.P.k.r.R/..p.p.p./..p.p.p./..p.p.p./..p.p.p./..p.p.p./..p.p.p./R.k.r.R.` |
| physical/13.jpg | `r3r1k1/1p1n1pp1/p1p4p/3p2q1/1P1P1Pn1/P1NQP1K1/1B1N2P1/R3R3` | `R1BQKBNR/PPPPPPPP/8/8/8/8/pppppppp/rnbqkbnr` |  |
| physical/14.jpg | `r3r1k1/1p1n1pp1/p1p4p/3p2P1/NP1P4/P2nP1K1/1B1N2P1/R3R3` | `1R3K1R/PPP1Q1PP/1B1N1N2/1P1P1B2/1P1P1P2/2n1b1n1/pp1p1p1p/r2k1q1r` |  |
| physical/15.jpg | `r3r1k1/5pp1/p1p4p/1pPp2P1/1P6/P2nP1K1/1B1N2P1/R3R3` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| physical/16.jpg | `r3r1k1/5p2/p1p4p/1pPp4/1P6/P3PNK1/1n4P1/R3R3` | `r1b1k2r/p1p1p1pp/1p1P1n2/2P1P1P1/2P1P1P1/2P1P1P1/2P1P1P1/2P1P1P1` | `r.b.k..r/p.p.p.pp/.p.P.n../..P.P.P./..P.P.P./..P.P.P./..P.P.P./..P.P.P.` |
| physical/17.jpg | `r5k1/5p2/p1p3rp/1pPp4/1P1N4/P3P3/1n4PK/R3R3` | `3K1P1R/4P3/1p1N1P1p/p1p1p1p1/1P1P1P1P/P1P1N1P1/P1P1P1P1/R1BQK1BR` | `...K.P.R/....P.../.p.N.P.p/p.p.p.p./.P.P.P.P/P.P.N.P./P.P.P.P./R.BQK.BR` |
| physical/18.jpg | `4r1k1/5p2/p1p3rp/1pPp4/PPnN4/4P3/4R1PK/R7` | `1R1K1R2/PPPQ1P1P/1N1B1N2/3P1P2/3p1p2/1n1b1n2/pppq1p1p/1r1k1r2` |  |
| physical/19.jpg | `6k1/5p2/2p3rp/1pPp4/1Pn1r3/4P3/2N1R1PK/R7` | `3K1R2/1P1P1P2/R1P1P3/2P1P3/3p1p2/1p1r1p2/p1p1p3/5k2` |  |
| physical/20.jpg | `6k1/5p2/2p3rp/1pP5/1P1Nr3/4n3/3R2PK/R7` | `2p1k2r/p1p1p1pp/2n1p3/3P1P1N/2P1N1P1/3P4/PP1K4/R3Q3` | `..p.k..r/p.p.p.pp/..n.p.../...P.P.N/..P.N.P./...P..../PP.K..../R...Q...` |
| physical/21.jpg | `6k1/5p2/2p3rp/1pP5/1PnNr3/8/6PK/R2R4` | `K7/3r1n1R/2p1p1p1/4P3/3P1P2/2p1P1P1/3P4/3R4` | `K......./...r.n.R/..p.p.p./....P.../...P.P../..p.P.P./...P..../...R....` |
| physical/22.jpg | `6k1/5p2/2p3r1/1pP2N1p/1P2r1n1/8/3R2P1/R5K1` | `1R1Q1B1R/3N4/P1P1P1P1/3p4/3P4/b1n1b1n1/1p1p1p1p/Kr1q1r1k` |  |
| physical/23.jpg | `6k1/5p2/2p2r2/1pP4N/1r4n1/8/3R2P1/R5K1` | `3k4/1p1p1p2/2r1b1n1/p1n1p3/1R1P4/2P1R3/4B1P1/3K1P1P` | `...k..../.p.p.p../..r.b.n./p.n.p.../.R.P..../..P.R.../....B.P./...K.P.P` |
| physical/24.jpg | `6k1/5p2/2p5/1pr5/1r4n1/6N1/3R2P1/5RK1` | `K7/5P2/4r3/3p1p2/3R4/1P1P1p2/1P1n3p/7k` |  |
| physical/25.jpg | `6k1/5p2/8/1p1p1N2/1r4n1/8/6P1/5RK1` | `8/4P3/3R4/3P4/8/3p4/1P1p1N1p/K2r1n1k` |  |
| physical/26.jpg | `6k1/5p2/8/1p3N2/1r4n1/3p4/6P1/3R2K1` | `3N4/4k3/1P6/1r1p4/3P4/1r1p1n2/5P2/2R1K3` |  |
| physical/27.jpg | `6k1/5p2/8/8/1pN3n1/3p4/1r4P1/3R2K1` | `k1n5/p1p1P1Q1/8/4p1p1/4N3/4p1R1/8/8` | `k.n...../p.p.P.Q./......../....p.p./....N.../....p.R./......../........` |
| physical/28.jpg | `6k1/5p2/8/8/1p6/3pn3/2rN2P1/4R1K1` | `3r1p2/2n1p3/3p4/2N1P3/8/8/K1P5/4k1q1` | `...r.p../..n.p.../...p..../..N.P.../......../......../K.P...../....k.q.` |
| physical/29.jpg | `4R1k1/5p2/8/8/8/1p1p4/3r2P1/6K1` | `2k5/R1p5/8/8/3p1p2/8/3P1P1Q/8` | `..k...../R.p...../......../......../...p.p../......../...P.P.Q/........` |
| physical/30.jpg | `R7/5pk1/8/8/8/3p4/1p1r2P1/6K1` | `K7/8/8/2p1p3/pk6/8/8/7R` |  |
| digital/01.png | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |  |
| digital/02.png | `r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R` | `rnbqkbnr/pppp1ppp/1n6/B3p3/4P3/6N1/PPPP1PPP/RNBQK2R` |  |
| digital/03.png | `r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQ1RK1` | `rnbqkbnr/pppppppp/2n1n3/2b1p3/2B1P3/3P1N2/PPP1PPPP/RNBQKBNR` | `rnbqkbnr/pppppppp/..n.n.../..b.p.../..B.P.../...P.N../PPP.PPPPP/RNBQKBNR` |
| digital/04.png | `rnbqkb1r/pp1ppppp/5n2/2p5/4P3/2N5/PPPP1PPP/R1BQKBNR` | `rnbqkbnr/pppppppp/8/4n3/2p1P3/8/2N5/RNBQKBNR` |  |
| digital/05.png | `8/8/8/8/8/5k2/6q1/7K` | `8/8/8/8/8/3k4/6q1/7K` |  |
| digital/06.png | `r3k2r/pppq1ppp/2np1n2/2b1p3/2B1P1b1/2NP1N2/PPPQ1PPP/R3K2R` | `r3k2r/ppppq1pp/2np1n2/1b1p4/2B1P1b1/2NP1N2/PPPQ1PPP/R3K2R` |  |
| digital/07.png | `6k1/5ppp/8/8/8/8/5PPP/R5K1` | `7k/6pp/8/8/8/8/5PPP/R6K` | `.......k/......ppp/......../......../......../......../.....PPP/R......K` |
| digital/08.png | `2kr3r/ppp2ppp/2n5/3qp3/8/2PP4/PP3PPP/RNBQ1RK1` | `r1k1qr1r/pppp1ppp/1n6/2q1p3/8/2PP4/PP2PPPP/RNBBQKNR` |  |

