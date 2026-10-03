#!/usr/bin/env python3
"""Conformance driver: generate, run every validator over every fixture, compare.

    uv run --no-project --with pyyaml python conformance/run.py [--skip-gen] [--no-kotlin]

Writes conformance/manifest.json (input), conformance/results.json and
conformance/results.md (output). Exit status is 0 whatever the verdicts are:
the table is the product, and a disagreement is a finding rather than a failure.
"""
import json
import subprocess
import sys
import time
from pathlib import Path

import yaml

SPIKE = Path(__file__).resolve().parent.parent
REPO = SPIKE.parent.parent
OUT = SPIKE / "out"
CONF = SPIKE / "conformance"
T = {}


def sh(cmd, cwd=SPIKE, **kw):
    t = time.time()
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, **kw)
    return r, time.time() - t


def build_manifest():
    entries = []

    def add(id_, schema, path, label):
        entries.append({"id": id_, "schema": schema, "path": str(path), "label": label})

    # Fixtures this repository already has.
    for name, label in [("valid", "ok"), ("empty", "bad"), ("missing-required", "bad"), ("secret-in-file", "bad"),
                        ("unknown-key", "bad"), ("wrong-type", "bad")]:
        add(f"repo/config/{name}", "shortener", REPO / "config/testdata" / f"{name}.yaml", label)
    for name in ["migrate", "prober", "redirect", "stat", "urls"]:
        add(f"repo/url-shortener/{name}", name, REPO / "examples/url-shortener/internal/config/testdata" / f"{name}.yaml", "ok")
    # The platform blocks of the library-convention example chart's values.
    derived = SPIKE / "fixtures/derived"
    derived.mkdir(exist_ok=True)
    td = REPO / "examples/url-shortener/charts/testdata"
    for name, src in [("service-example", td / "service-example/values.yaml"), ("service-example-everything", td / "service-example-everything.yaml")]:
        values = yaml.safe_load(src.read_text())
        p = derived / f"platform-{name}.yaml"
        p.write_text(yaml.safe_dump(values["platform"]) if values.get("platform") else "{}\n")
        add(f"repo/chart-values/{name}#platform", "platform", p, "ok")
    # New hand-written fixtures: <schema>/<ok|bad>-<name>.yaml
    for p in sorted((SPIKE / "fixtures").glob("*/*.yaml")):
        if p.parent.name == "derived":
            continue
        add(f"new/{p.parent.name}/{p.stem}", p.parent.name, p, p.stem.split("-")[0])
    # Probes generated from the vocabulary annotations.
    for p in sorted((OUT / "probes").glob("*.json")):
        add(f"probe/{p.stem}", "showcase", p, p.stem.split("-")[0])
    (CONF / "manifest.json").write_text(json.dumps(entries, indent=1))
    return entries


def jsonl(r, what):
    if r.returncode != 0:
        sys.exit(f"{what} failed:\n{r.stderr[-3000:]}\n{r.stdout[-500:]}")
    return [json.loads(line) for line in r.stdout.splitlines() if line.startswith("{")]


def main():
    skip_gen = "--skip-gen" in sys.argv
    if not skip_gen:
        r, T["generate (jsonschema, zod, pydantic, probes)"] = sh(["pkl", "run", "gen.pkl", "--", "--dir", "out"])
        if r.returncode:
            sys.exit(r.stderr)
    entries = build_manifest()
    results = []
    r, T["generate go (pkl-go pkl.golang 0.13.2)"] = sh(["bash", "conformance/go/generate.sh"])
    if r.returncode:
        sys.exit(r.stderr)
    r, T["go: jsonschema-hand/gen + pkl-eval + pkl-go"] = sh(["go", "run", ".", str(SPIKE), str(REPO), str(OUT / "jsonschema")], cwd=CONF / "go")
    results += jsonl(r, "go runner")
    # The generated module imports `zod`, which resolves from the directory it sits in.
    (CONF / "ts/contract.zod.ts").write_text((OUT / "ts/contract.zod.ts").read_text())
    r, T["ts: ajv-hand/gen + zod"] = sh(["node", "run.mjs", str(SPIKE), str(REPO), str(OUT / "jsonschema"), str(CONF / "ts/contract.zod.ts")], cwd=CONF / "ts")
    results += jsonl(r, "ts runner")
    r, T["py: jsonschema-hand/gen + pydantic"] = sh(
        ["uv", "run", "--no-project", "--with", "pydantic==2.12.5", "--with", "pyyaml", "--with", "jsonschema==4.25.1", "--with", "referencing",
         "python", str(CONF / "py/run.py"), str(SPIKE), str(REPO), str(OUT / "jsonschema"), str(OUT / "py/contract_pyd.py")])
    results += jsonl(r, "py runner")
    if "--no-kotlin" not in sys.argv:
        r, T["kotlin: networknt-hand/gen + pkl-kotlin"] = sh(["bash", "conformance/kotlin/run.sh"])
        results += jsonl(r, "kotlin runner")
    extras()
    report(entries, results)


def extras():
    """Equivalence, Helm composition, Helm lint and the gitops render."""
    py = [sys.executable, "conformance/semdiff.py"]
    eq, _ = sh(py + [str(OUT / "jsonschema")])
    hand = Path("/tmp/pklspike-helm-hand")
    comp, T["compose hand-written + validate values (Go, chartschema)"] = sh(["go", "run", "./spike/pkl/conformance/compose", ".", str(OUT / "helm"), str(hand)], cwd=REPO)
    if comp.returncode:
        sys.exit(comp.stderr)
    helm_eq, _ = sh(py + ["--helm", str(OUT / "helm"), str(hand)])
    lint, T["helm lint x7 charts"] = sh(["bash", "conformance/helm-lint.sh"])
    gitops, T["gitops render + helm lint + failing modules"] = sh(["bash", "gitops/render.sh"])
    (CONF / "equivalence.txt").write_text(
        "# Generated JSON Schemas vs the hand-written ones\n\n" + eq.stdout +
        "\n# Generated chart values.schema.json vs chartschema.Compose of the hand-written schemas\n\n" + helm_eq.stdout +
        "\n# Generated values.yaml validated against both (the library Helm validates with)\n\n" + "\n".join(sorted(comp.stdout.splitlines())) +
        "\n\n# helm lint of the generated charts\n\n" + lint.stdout +
        "\n# gitops: rendered environments and failing modules\n\n" + gitops.stdout)
    for k in ["jsonschema", "zod", "pydantic", "probes", "helm"]:
        _, T["generate " + k + " alone"] = sh(["pkl", "run", "gen.pkl", "--", "--dir", "/tmp/pklspike-gen", "--target", k])


COLS = ["jsonschema-hand", "ajv-hand", "py-hand", "kt-hand"]


def report(entries, results):
    # Name validators per language so that columns are unambiguous.
    rename = {("jsonschema-hand", "go"): "go-schema-hand"}
    by = {}
    for r in results:
        by.setdefault(r["id"], {})[r["validator"]] = r
    validators = ["ajv-hand", "ajv-gen", "zod", "jsonschema-hand", "jsonschema-gen", "pydantic",
                  "go-hand", "go-gen", "pkl-eval", "pkl-go", "kt-hand", "kt-gen", "pkl-kotlin"]
    # The Go runner and the Python runner both emit "jsonschema-*": the Go ones come first in `results`.
    fixed = {}
    for r in results:
        pass
    rows = []
    seen_go = {}
    for e in entries:
        v = {}
        for r in [x for x in results if x["id"] == e["id"]]:
            name = r["validator"]
            v.setdefault(name, []).append(r)
        row = {"id": e["id"], "schema": e["schema"], "label": e["label"], "v": {}}
        # jsonschema-hand/gen appear twice: first Go (santhosh), second Python (jsonschema).
        for name, lst in v.items():
            if name in ("jsonschema-hand", "jsonschema-gen"):
                row["v"]["go-" + name.split("-")[1]] = lst[0]
                if len(lst) > 1:
                    row["v"]["py-" + name.split("-")[1]] = lst[1]
            else:
                row["v"][name] = lst[0]
        rows.append(row)
    cols = ["ajv-hand", "ajv-gen", "go-hand", "go-gen", "py-hand", "py-gen", "kt-hand", "kt-gen", "zod", "pydantic", "pkl-eval", "pkl-go", "pkl-kotlin"]
    cols = [c for c in cols if any(c in r["v"] for r in rows)]
    # Struct-only decodes (no Pkl, no schema) are shown but are not validators: they are
    # expected to be lax, and are left out of the agreement count.
    struct_cols = [c for c in ["go-struct", "kt-struct"] if any(c in r["v"] for r in rows)]
    (CONF / "results.json").write_text(json.dumps({"columns": cols, "rows": rows, "timing_seconds": T}, indent=1))

    def mark(r, c):
        x = r["v"].get(c)
        return "-" if x is None else ("A" if x["accept"] else "R")

    dis = []
    for r in rows:
        verdicts = {c: mark(r, c) for c in cols if mark(r, c) != "-"}
        vals = set(verdicts.values())
        r["agree"] = len(vals) == 1
        r["agreed_with_label"] = r["agree"] and (("A" in vals) == (r["label"] == "ok"))
        if not r["agree"]:
            dis.append(r)
    probes = [r for r in rows if r["id"].startswith("probe/")]
    fx = [r for r in rows if not r["id"].startswith("probe/")]
    lines = ["# Conformance results", "", "Generated by `conformance/run.py`. A = accept, R = reject, - = not applicable.",
             "Columns: ajv/go/py/kt = JSON Schema validators (Ajv, santhosh-tekuri, python-jsonschema, networknt) on the hand-written (`-hand`) or the Pkl-generated (`-gen`) schemas; `zod`, `pydantic` = generated types; `pkl-eval` = Pkl itself; `pkl-go`, `pkl-kotlin` = the same evaluation decoded into the structs the official generators emit.",
             "", f"Fixtures: {len(fx)} service/platform fixtures + {len(probes)} vocabulary probes. Validators per fixture: up to {len(cols)}.",
             f"Fixtures on which every validator agrees: {sum(1 for r in rows if r['agree'])} of {len(rows)}. Disagreements: {len(dis)}.", ""]
    lines += ["## Time", ""] + ["| step | seconds |", "|---|---|"] + [f"| {k} | {v:.1f} |" for k, v in T.items()] + [""]
    lines += ["## Fixtures", "", "| fixture | label | " + " | ".join(cols) + " | agree |", "|---|---|" + "---|" * (len(cols) + 1)]
    for r in fx:
        lines.append(f"| {r['id']} | {r['label']} | " + " | ".join(mark(r, c) for c in cols) + f" | {'yes' if r['agree'] else '**NO**'} |")
    lines += ["", "## What the generated Go and Kotlin types accept on their own", "",
              "`go-struct` / `kt-struct`: the fixture decoded straight into the struct pkl-gen-go / pkl-codegen-kotlin emitted (encoding/json, Jackson), with no Pkl and no schema. "
              "Over the fixtures that should be REJECTED (label bad, non-probe) they accept:", ""]
    for c in struct_cols:
        bad = [r for r in fx if r["label"] == "bad" and c in r["v"]]
        acc = [r for r in bad if mark(r, c) == "A"]
        lines.append(f"- `{c}`: accepts {len(acc)} of {len(bad)} invalid fixtures; probes: accepts {sum(1 for r in probes if r['label']=='bad' and c in r['v'] and mark(r, c)=='A')} of {sum(1 for r in probes if r['label']=='bad' and c in r['v'])} invalid vocabulary values.")
    lines += ["", "## Vocabulary probes", "", f"{len(probes)} probes generated from the annotations; {sum(1 for r in probes if r['agree'])} agree everywhere.", ""]
    lines += ["| probe | label | " + " | ".join(cols) + " |", "|---|---|" + "---|" * len(cols)]
    for r in probes:
        if not r["agree"] or not r["agreed_with_label"]:
            lines.append(f"| {r['id']} | {r['label']} | " + " | ".join(mark(r, c) for c in cols) + " |")
    lines += ["", "## Disagreements", ""]
    for r in dis:
        acc = [c for c in cols if mark(r, c) == "A"]
        rej = [c for c in cols if mark(r, c) == "R"]
        why = next((r["v"][c]["detail"] for c in rej if r["v"][c]["detail"]), "")
        lines.append(f"- `{r['id']}` (label {r['label']}): accepted by {', '.join(acc)}; rejected by {', '.join(rej)}. First rejection: {why[:140]}")
    wrong = [r for r in rows if r["agree"] and not r["agreed_with_label"]]
    lines += ["", "## Unanimous verdict that differs from the label", ""] + [f"- `{r['id']}` (label {r['label']})" for r in wrong]
    (CONF / "results.md").write_text("\n".join(lines) + "\n")
    print("\n".join(lines[:12]))
    print("disagreements:", len(dis), " unanimous-but-unlabelled:", len(wrong))


main()
