# Position corpus sources

The harness (`cmd/poseval`) compares only the **board field** of each FEN (the part
before the first space); the side-to-move / castling fields are recorded for
completeness and are not scored.

## Physical (photos of real 3D boards) — ChessReD

All 30 physical images are **real smartphone photographs** of a physical chess
board (`physical/01.jpg` .. `physical/30.jpg`). They come from the
**Chess Recognition Dataset (ChessReD)** by Athanasios Masouris & Jan van Gemert
(TU Delft), published on 4TU.ResearchData.

- Dataset DOI: `10.4121/99b5c721-280b-450b-b058-b2900b69a90f.v2`
- Landing page: https://data.4tu.nl/datasets/99b5c721-280b-450b-b058-b2900b69a90f
- Code / paper repo: https://github.com/ThanosM97/end-to-end-chess-recognition
  (paper: "End-to-End Chess Recognition", VISIGRAPP 2024, arXiv:2310.04086)
- License: **CC BY-NC-SA 4.0**

**Ground truth = dataset annotations (not hand-labeling).** ChessReD ships an
`annotations.json` with, for every image, the exact piece **category** (white/black
× pawn/rook/knight/bishop/queen/king) and its **algebraic square** (`a1`..`h8`).
The board field of each FEN below was **assembled deterministically from those
per-square annotations** — the most reliable possible ground truth, independent of
camera angle. Each selected photo was additionally viewed with an image tool to
confirm it is a clear, real board photo and that the annotation-derived position is
plausible (correct piece counts/colours, captured pieces set aside, etc.).

All 30 photos are from ChessReD **game 0** (`images/0/G000_IMG###.jpg`, captured on
a Huawei P40 Pro). One game was used because it provides a clean opening →
middlegame → endgame progression with one distinct position per image, and because
the dataset is distributed only as large ZIP archives (the 4TU server does **not**
honour HTTP range requests, so selective extraction of arbitrary games is not
possible without downloading a multi-GB archive — the full set is 24 GB, the
`chessred2k` subset 4.6 GB). 30 move numbers were chosen spread across the whole
game to maximise position variety (32 pieces down to 8). The images were downscaled
to 1024×1024 (from the original 3072×3072) to keep the corpus small and uploads
fast; they are otherwise unmodified real photographs.

The `orientation` field is recorded as `white_bottom` for all physical images as a
best-effort default; ChessReD photos are shot from varying angles (corner / player /
low / top view) so the literal on-screen orientation varies. This field is **not
used by the scorer** — the FEN board field uses absolute `a1`..`h8` coordinates, so
ground truth is correct regardless of how the photo is rotated.

Note: image index ≠ move number (move 67 was a position-level duplicate of move 63
and was replaced by move 65). The source frame for each image is listed below.

| File | Source frame (ChessReD game 0) | Phase | FEN board field (from annotations) |
|------|-------------------------------|-------|------------------------------------|
| physical/01.jpg | G000_IMG000.jpg | opening | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR` |
| physical/02.jpg | G000_IMG002.jpg | opening | `rnbqkbnr/ppp1pppp/8/3p4/1P6/8/P1PPPPPP/RNBQKBNR` |
| physical/03.jpg | G000_IMG004.jpg | opening | `rn1qkbnr/ppp1pppp/8/3p1b2/1P6/8/PBPPPPPP/RN1QKBNR` |
| physical/04.jpg | G000_IMG006.jpg | opening | `rn1qkbnr/ppp2ppp/4p3/3p1b2/1P6/5N2/PBPPPPPP/RN1QKB1R` |
| physical/05.jpg | G000_IMG009.jpg | opening | `rn1qkb1r/ppp2ppp/4pn2/3p1b2/1P6/P3PN2/1BPP1PPP/RN1QKB1R` |
| physical/06.jpg | G000_IMG012.jpg | middlegame | `r2qkb1r/pp1n1ppp/2p1pn2/3p1b2/1PP5/P3PN2/1B1P1PPP/RN1QKB1R` |
| physical/07.jpg | G000_IMG015.jpg | middlegame | `r2qk2r/pp1n1ppp/2pbpn2/3p1b2/1PP5/P1N1PN2/1B1PBPPP/R2QK2R` |
| physical/08.jpg | G000_IMG018.jpg | middlegame | `r2q1rk1/pp1n1pp1/2pbpn1p/3p1b2/1PP5/P1N1PN2/1B1PBPPP/R2Q1RK1` |
| physical/09.jpg | G000_IMG021.jpg | middlegame | `r2q1rk1/1p1n1pp1/p1pbpn1p/3p1b2/1PPP4/P1NBPN2/1B3PPP/R2Q1RK1` |
| physical/10.jpg | G000_IMG024.jpg | middlegame | `r4rk1/1p1nqpp1/p1pbpn1p/3P4/1P1P4/P1NbPN2/1B3PPP/R2Q1RK1` |
| physical/11.jpg | G000_IMG027.jpg | middlegame | `r4rk1/1p1nqpp1/p1pb1n1p/3p4/1P1P4/P1NQP3/1B1N1PPP/R4RK1` |
| physical/12.jpg | G000_IMG031.jpg | middlegame | `r3r1k1/1p1nqpp1/p1p2n1p/3p4/1P1P4/P1NQP3/1B1N1PPK/R3R3` |
| physical/13.jpg | G000_IMG035.jpg | middlegame | `r3r1k1/1p1n1pp1/p1p4p/3p2q1/1P1P1Pn1/P1NQP1K1/1B1N2P1/R3R3` |
| physical/14.jpg | G000_IMG039.jpg | middlegame | `r3r1k1/1p1n1pp1/p1p4p/3p2P1/NP1P4/P2nP1K1/1B1N2P1/R3R3` |
| physical/15.jpg | G000_IMG043.jpg | middlegame | `r3r1k1/5pp1/p1p4p/1pPp2P1/1P6/P2nP1K1/1B1N2P1/R3R3` |
| physical/16.jpg | G000_IMG047.jpg | endgame | `r3r1k1/5p2/p1p4p/1pPp4/1P6/P3PNK1/1n4P1/R3R3` |
| physical/17.jpg | G000_IMG051.jpg | endgame | `r5k1/5p2/p1p3rp/1pPp4/1P1N4/P3P3/1n4PK/R3R3` |
| physical/18.jpg | G000_IMG055.jpg | endgame | `4r1k1/5p2/p1p3rp/1pPp4/PPnN4/4P3/4R1PK/R7` |
| physical/19.jpg | G000_IMG059.jpg | endgame | `6k1/5p2/2p3rp/1pPp4/1Pn1r3/4P3/2N1R1PK/R7` |
| physical/20.jpg | G000_IMG063.jpg | endgame | `6k1/5p2/2p3rp/1pP5/1P1Nr3/4n3/3R2PK/R7` |
| physical/21.jpg | G000_IMG065.jpg | endgame | `6k1/5p2/2p3rp/1pP5/1PnNr3/8/6PK/R2R4` |
| physical/22.jpg | G000_IMG071.jpg | endgame | `6k1/5p2/2p3r1/1pP2N1p/1P2r1n1/8/3R2P1/R5K1` |
| physical/23.jpg | G000_IMG075.jpg | endgame | `6k1/5p2/2p2r2/1pP4N/1r4n1/8/3R2P1/R5K1` |
| physical/24.jpg | G000_IMG079.jpg | endgame | `6k1/5p2/2p5/1pr5/1r4n1/6N1/3R2P1/5RK1` |
| physical/25.jpg | G000_IMG083.jpg | endgame | `6k1/5p2/8/1p1p1N2/1r4n1/8/6P1/5RK1` |
| physical/26.jpg | G000_IMG087.jpg | endgame | `6k1/5p2/8/1p3N2/1r4n1/3p4/6P1/3R2K1` |
| physical/27.jpg | G000_IMG091.jpg | endgame | `6k1/5p2/8/8/1pN3n1/3p4/1r4P1/3R2K1` |
| physical/28.jpg | G000_IMG095.jpg | endgame | `6k1/5p2/8/8/1p6/3pn3/2rN2P1/4R1K1` |
| physical/29.jpg | G000_IMG099.jpg | endgame | `4R1k1/5p2/8/8/8/1p1p4/3r2P1/6K1` |
| physical/30.jpg | G000_IMG102.jpg | endgame | `R7/5pk1/8/8/8/3p4/1p1r2P1/6K1` |

The previous starter `physical/01-04` (Wikimedia "Staunton No. 6" shots, mostly the
starting position) were **replaced** because they lacked positional variety. Other
games in the ChessReD `chessred2k` subset (games 6, 19, 22, …) were not used: the
ZIP is stored sequentially and the 4TU host ignores HTTP Range, so reaching later
games would mean downloading multiple GB; game 0 alone already covers all phases.

## Digital (2D diagrams) — unchanged

Rendered with the public lichess/web-boardimage SVG renderer
(https://github.com/niklasf/web-boardimage), hosted at
https://backscattering.de/web-boardimage/ — the same renderer lichess uses. Each
image is produced from a known FEN passed in the URL, so the FEN is exact ground
truth; the renderer output was still viewed to confirm it matched. Piece graphics
are the lichess "cburnett" set (GPLv2+); the renderer is GPLv3.

| File | Position / opening | FEN (board field) |
|------|--------------------|-------------------|
| digital/01.png | Starting position | rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR |
| digital/02.png | Ruy Lopez, after 3.Bb5 | r1bqkbnr/pppp1ppp/2n5/1B2p3/4P3/5N2/PPPP1PPP/RNBQK2R |
| digital/03.png | Italian Game, white castled | r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/3P1N2/PPP2PPP/RNBQ1RK1 |
| digital/04.png | Sicilian Defence, 2.Nc3 | rnbqkb1r/pp1ppppp/5n2/2p5/4P3/2N5/PPPP1PPP/R1BQKBNR |
| digital/05.png | K+Q vs K endgame fragment | 8/8/8/8/8/5k2/6q1/7K |
| digital/06.png | Symmetric Giuoco-Pianissimo middlegame | r3k2r/pppq1ppp/2np1n2/2b1p3/2B1P1b1/2NP1N2/PPPQ1PPP/R3K2R |
| digital/07.png | R + 3 pawns endgame | 6k1/5ppp/8/8/8/8/5PPP/R5K1 |
| digital/08.png | Queen's-side-castled middlegame | 2kr3r/ppp2ppp/2n5/3qp3/8/2PP4/PP3PPP/RNBQ1RK1 |

Example diagram URL (digital/01.png):
`https://backscattering.de/web-boardimage/board.png?fen=rnbqkbnr%2Fpppppppp%2F8%2F8%2F8%2F8%2FPPPPPPPP%2FRNBQKBNR`
