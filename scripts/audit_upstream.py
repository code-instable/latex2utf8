#!/usr/bin/env python3
"""Compare the symbol database with the upstream unicode-latex table.

    python3 scripts/audit_upstream.py [path/to/latex.ts]

Without an argument the table is downloaded from
https://github.com/ojsheikh/unicode-latex. Prints, for the upstream entries
that lutf cannot resolve by name, which ones are plain naming differences (the
character exists under another name) and which characters are missing outright.
Names of the math alphabets (bf, it, bb, ...) differ on purpose and are skipped.

The script only reads; use it to decide what to add to symbols/data/*.tsv.
"""
import collections
import pathlib
import re
import sys
import unicodedata
import urllib.request

UPSTREAM = "https://raw.githubusercontent.com/ojsheikh/unicode-latex/master/src/latex.ts"
DATA = pathlib.Path(__file__).resolve().parent.parent / "symbols" / "data"


def upstream(text):
    table = {}
    entry = re.compile(r"""^'((?:[^'\\]|\\.)*)'\s*:\s*'((?:[^'\\]|\\.)*)'$""")
    for line in text.splitlines():
        m = entry.match(line.strip().rstrip(","))
        if m:
            name = m.group(1).replace("\\\\", "")
            value = m.group(2).replace("\\'", "'").replace("\\\\", "\\")
            table.setdefault(name, value)
    return table


def records(path):
    for line in path.read_text(encoding="utf-8").splitlines():
        if line and not line.startswith("#"):
            yield line.split("\t")


def ours():
    names = {name: value for name, value, *_ in records(DATA / "symbols.tsv")}
    for alias, target, *_ in records(DATA / "aliases.tsv"):
        names[alias] = names[target]
    for key, value in records(DATA / "fractions.tsv"):
        num, den = key.split("/")
        if len(num) == len(den) == 1:
            names.setdefault("frac" + num + den, value)
    marks = {name for name, kind, *_ in records(DATA / "commands.tsv") if kind in ("accent", "wideaccent")}
    return names, marks


def is_math_alphabet(value):
    return len(value) == 1 and (0x1D400 <= ord(value) <= 0x1D7FF or 0x2100 <= ord(value) <= 0x214F)


def main():
    if len(sys.argv) > 1:
        text = pathlib.Path(sys.argv[1]).read_text(encoding="utf-8")
    else:
        with urllib.request.urlopen(UPSTREAM, timeout=30) as resp:
            text = resp.read().decode("utf-8")
    up = upstream(text)
    names, accent_names = ours()
    by_value = collections.defaultdict(list)
    for name, value in names.items():
        by_value[value].append(name)

    renamed, missing = [], []
    for name, value in sorted(up.items()):
        if name in names or name in accent_names or "/" in name or is_math_alphabet(value):
            continue
        if len(value) == 1 and unicodedata.category(value) in ("Mn", "Me"):
            continue  # combining marks are commands, see commands.tsv
        (renamed if value in by_value else missing).append((name, value))

    print("upstream entries: %d, symbols known to lutf: %d\n" % (len(up), len(names)))
    print("known characters, upstream name not accepted (%d):" % len(renamed))
    for name, value in renamed:
        print("  %-24s %s  (lutf: %s)" % (name, value, by_value[value][0]))
    print("\ncharacters lutf does not have (%d):" % len(missing))
    for name, value in missing:
        print("  %-24s %s  U+%s" % (name, value, "+".join("%04X" % ord(c) for c in value)))


if __name__ == "__main__":
    main()
