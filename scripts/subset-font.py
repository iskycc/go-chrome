#!/usr/bin/env python3
"""Subset embedded CJK fonts to reduce runtime memory footprint.

The project embeds Maple Mono CN Regular and Medium (~18 MB each). At runtime
Fyne loads the chosen font resources into memory, which is a major contributor
to the 600+ MB RSS seen on startup.

This script keeps only the glyphs needed for the Chinese UI (ASCII, common
CJK, punctuation and a few symbols). Typical output size is 2-4 MB per font,
which cuts the font memory cost by ~80%.

Usage:
    pip install fonttools brotli
    python scripts/subset-font.py \
        assets/fonts/MapleMono-CN-Regular.ttf \
        assets/fonts/MapleMono-CN-Regular.subset.ttf

Then replace the source font in assets/fonts/ and rebuild.
"""

import argparse
import sys

# Character coverage for the UI: ASCII printable, common CJK, punctuation and
# a small safety set of control/formatting characters.
SUBSET_TEXT = (
    # ASCII printable
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    " !\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
    # Common UI symbols
    "·•→←↑↓✓✗⚠ℹ✕✔▶■□◀…"
    # Common CJK Unified Ideographs (cover most modern Chinese UI text)
    "的一是在不了有和人这中大为上个国我以要他时来用们生到作地于出就分对成会可主发年动同工也能下过子说产种面而方后多定行学法所民得经十三之进着等部度家电力里如水化高自二理起小物现实加量都两体制机当使点从业本去把性好应开它合还因由其些然前外天政四日那社义事平形相全表间样与关各重新线内数正心反你明看原又么利比或但质气第向道命此变条只没结解问意建月公无系军很情者最立代想已通并提直题党程展五果料象员革位入常文总次品式活设及管特件长求老头基资边流路级少图山统接知较将组见计别她手角期根论运农指几九区强放决西被干做必战先回则任取完举色"
    # CJK punctuation / numerals / symbols
    "，。、；：？！\"\"''（）【】《》〈〉〔〕［］｛｝「」『』—…～·•￥"
    "零一二三四五六七八九十百千万亿两"
    "甲乙丙丁戊己庚辛壬癸子丑寅卯辰巳午未申酉戌亥"
    # Full-width ASCII used by the UI in places
    "０１２３４５６７８９ＡＢＣＤＥＦＧＨＩＪＫＬＭＮＯＰＱＲＳＴＵＶＷＸＹＺ"
    "ａｂｃｄｅｆｇｈｉｊｋｌｍｎｏｐｑｒｓｔｕｖｗｘｙｚ"
)


def subset_font(input_path: str, output_path: str) -> None:
    try:
        from fontTools.subset import Subsetter, Options
        from fontTools.ttLib import TTFont
    except ImportError as exc:
        print("fontTools is required. Install it with: pip install fonttools brotli", file=sys.stderr)
        raise SystemExit(1) from exc

    options = Options()
    options.layout_features = ["*"]
    options.name_IDs = ["*"]
    options.name_legacy = True
    options.notdef_outline = True
    options.recommended_glyphs = True
    options.desubroutinize = True
    options.hinting = False  # Hinting tables are large and Fyne doesn't use them.
    options.glyph_names = True

    font = TTFont(input_path)
    subsetter = Subsetter(options=options)
    subsetter.populate(text=SUBSET_TEXT)
    subsetter.subset(font)

    # Drop tables that Fyne's text shaper does not need.
    drop_tables = {"DSIG", "LTSH", "PCLT", "VDMX", "VORG", "EBDT", "EBLC", "EBSC", "MATH"}
    for tag in list(font.keys()):
        if tag in drop_tables:
            del font[tag]

    font.flavor = "woff2"
    font.save(output_path)

    in_size = input_path.stat().st_size
    out_size = output_path.stat().st_size
    print(f"Subsetted {input_path} -> {output_path}")
    print(f"  {in_size:,} bytes -> {out_size:,} bytes ({out_size * 100 / in_size:.1f}%)")


def main() -> None:
    parser = argparse.ArgumentParser(description="Subset embedded CJK fonts.")
    parser.add_argument("input", help="Input TTF/OTF font file")
    parser.add_argument("output", help="Output subset font file")
    args = parser.parse_args()
    from pathlib import Path
    subset_font(Path(args.input), Path(args.output))


if __name__ == "__main__":
    main()
