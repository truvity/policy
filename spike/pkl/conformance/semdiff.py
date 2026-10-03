#!/usr/bin/env python3
"""Semantic diff of generated JSON Schemas against the hand-written ones.

Normalises key order and the order of set-like lists (required, enum, a type
array), pairs documents by `$id`, and prints every difference as a path with
both values.  Usage: semdiff.py <generated-dir> [--json]
"""
import json, sys, glob, os

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))
HAND = (glob.glob(ROOT + "/schemas/*.json") + glob.glob(ROOT + "/schemas/fragments/*.json") +
        glob.glob(ROOT + "/examples/url-shortener/schemas/*.json") +
        [ROOT + "/examples/url-shortener/log/src/url_shortener_log/log.schema.json",
         ROOT + "/config/testdata/shortener.schema.json"])
SETLISTS = {"required", "enum", "type"}


def norm(v, key=None):
    if isinstance(v, dict):
        return {k: norm(x, k) for k, x in sorted(v.items())}
    if isinstance(v, list):
        items = [norm(x) for x in v]
        if key in SETLISTS and all(isinstance(i, str) for i in items):
            return sorted(items)
        if key == "allOf":
            return sorted(items, key=lambda i: json.dumps(i, sort_keys=True))
        return items
    if isinstance(v, float) and v.is_integer():
        return int(v)
    return v


def diff(a, b, path, out):
    if isinstance(a, dict) and isinstance(b, dict):
        for k in sorted(set(a) | set(b)):
            if k not in a:
                out.append((path + "/" + k, "<absent>", b[k]))
            elif k not in b:
                out.append((path + "/" + k, a[k], "<absent>"))
            else:
                diff(a[k], b[k], path + "/" + k, out)
    elif isinstance(a, list) and isinstance(b, list) and len(a) == len(b):
        for i, (x, y) in enumerate(zip(a, b)):
            diff(x, y, "%s/%d" % (path, i), out)
    elif a != b:
        out.append((path, a, b))


def helm(gdir, hdir):
    """Compare each chart's generated values.schema.json with the one chartschema.Compose builds."""
    total = 0
    for p in sorted(glob.glob(gdir + "/*/values.schema.json")):
        name = os.path.basename(os.path.dirname(p))
        out = []
        diff(norm(json.load(open(os.path.join(hdir, name, "values.schema.json")))), norm(json.load(open(p))), "", out)
        total += len(out)
        print("== helm/%s: %s" % (name, "IDENTICAL" if not out else "%d difference(s)" % len(out)))
        for path, x, y in out:
            print("   %s\n      hand: %s\n      gen : %s" % (path, json.dumps(x, ensure_ascii=False)[:160], json.dumps(y, ensure_ascii=False)[:160]))
    print("\nTOTAL helm: %d differences" % total)


def main():
    if sys.argv[1] == "--helm":
        return helm(sys.argv[2], sys.argv[3])
    gdir = sys.argv[1]
    hand = {}
    for p in HAND:
        d = json.load(open(p))
        hand[d["$id"]] = (os.path.relpath(p, ROOT), d)
    total = 0
    rows = []
    seen = set()
    for p in sorted(glob.glob(gdir + "/**/*.json", recursive=True)):
        g = json.load(open(p))
        seen.add(g["$id"])
        if g["$id"] not in hand:
            rows.append((os.path.relpath(p, gdir), "(no hand-written counterpart: new in the Pkl contract)", []))
            continue
        rel, h = hand[g["$id"]]
        out = []
        diff(norm(h), norm(g), "", out)
        rows.append((os.path.relpath(p, gdir), rel, out))
        total += len(out)
    for i, (rel, _) in sorted(hand.items()):
        if i not in seen:
            rows.append(("(none generated)", rel, [("(document)", "present", "<no generated counterpart>")]))
            total += 1
    if "--json" in sys.argv:
        print(json.dumps([{"generated": a, "hand": b, "diffs": [{"path": p, "hand": x, "gen": y} for p, x, y in o]} for a, b, o in rows], indent=1, ensure_ascii=False))
        return
    for a, b, out in rows:
        print("== %s  vs  %s: %s" % (a, b, "NEW" if b.startswith("(no") else "IDENTICAL" if not out else "%d difference(s)" % len(out)))
        for p, x, y in out:
            print("   %s\n      hand: %s\n      gen : %s" % (p, json.dumps(x, ensure_ascii=False)[:200], json.dumps(y, ensure_ascii=False)[:200]))
    print("\nTOTAL: %d documents, %d identical, %d differences" % (len(rows), sum(1 for r in rows if not r[2]), total))


main()
