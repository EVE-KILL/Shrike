#!/usr/bin/env python3
"""Generate the common-script Exo 2 subset; keep the original for other scripts.

Run with Python and fonttools[woff]==4.64.0 installed:
    python web/scripts/subset-fonts.py

Keep these ranges aligned with the Exo 2 faces in app/assets/main.css.
Bump the output filename when the original font or subset settings change:
/fonts/** assets are cached immutably.
"""
from pathlib import Path
from fontTools import subset
from fontTools.ttLib import TTFont

fonts = Path(__file__).resolve().parents[1] / "public" / "fonts"
font = TTFont(fonts / "Exo2-Variable.woff2", recalcTimestamp=False)
options = subset.Options()
options.flavor = "woff2"
options.layout_features = ["*"]
subsetter = subset.Subsetter(options=options)
subsetter.populate(unicodes=list(range(0x0370)) + list(range(0x2000, 0x2400)))
subsetter.subset(font)
font.flavor = "woff2"
font.save(fonts / "Exo2-Latin-v1.woff2")
