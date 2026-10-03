"""Runs every fixture through three Python-side validators:

  jsonschema-hand  python-jsonschema on the hand-written schemas (as the repo's loader does)
  jsonschema-gen   the same on the schemas generated from Pkl
  pydantic         the pydantic models generated from Pkl

usage: run.py <spike/pkl dir> <repo root> <generated schema dir> <generated pydantic .py>
"""
import importlib.util
import json
import sys
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator
from referencing import Registry, Resource
from referencing.jsonschema import DRAFT202012

spike, repo, gen_dir, pyd_file = (Path(a) for a in sys.argv[1:5])
manifest = json.loads((spike / "conformance/manifest.json").read_text())
docs = json.loads((spike / "conformance/documents.json").read_text())

spec = importlib.util.spec_from_file_location("contract_pyd", pyd_file)
mod = importlib.util.module_from_spec(spec)
sys.modules["contract_pyd"] = mod
spec.loader.exec_module(mod)


def registry(files):
    loaded = {f: json.loads(Path(f).read_text()) for f in files}
    reg = Registry().with_resources(
        (d["$id"], Resource.from_contents(d, default_specification=DRAFT202012)) for d in loaded.values()
    )
    return reg, loaded


hand_files = {str(repo / d["hand"]) for d in docs.values() if d["hand"]}
hand_files |= {str(repo / "schemas/service.json")} | {str(p) for p in (repo / "schemas/fragments").glob("*.json")}
hand_reg, hand_docs = registry(sorted(hand_files))
gen_reg, gen_docs = registry(sorted(str(p) for p in gen_dir.rglob("*.json")))

out = []
for e in manifest:
    doc = yaml.safe_load(Path(e["path"]).read_text())
    d = docs[e["schema"]]

    def emit(name, accept, detail=""):
        out.append({"id": e["id"], "validator": name, "accept": accept, "detail": detail})

    def via_schema(name, reg, loaded, file):
        v = Draft202012Validator(loaded[file], registry=reg)
        errs = sorted(v.iter_errors(doc), key=lambda x: list(x.absolute_path))
        emit(name, not errs, "" if not errs else f"{'/'.join(map(str, errs[0].absolute_path)) or '(root)'}: {errs[0].message[:80]}")

    if d["hand"]:
        via_schema("jsonschema-hand", hand_reg, hand_docs, str(repo / d["hand"]))
    via_schema("jsonschema-gen", gen_reg, gen_docs, str(gen_dir / d["gen"]))
    try:
        mod.SCHEMAS[d["gen"].removesuffix(".json")].model_validate(doc)
        emit("pydantic", True)
    except Exception as err:  # noqa: BLE001
        emit("pydantic", False, str(err).splitlines()[0][:80] + " " + (str(err).splitlines()[1] if len(str(err).splitlines()) > 1 else ""))

for r in out:
    print(json.dumps(r))
