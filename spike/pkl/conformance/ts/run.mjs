// Runs every fixture in the manifest through three TypeScript-side validators:
//   ajv-hand  Ajv (2020-12, the options the repo's own loader uses) on the hand-written schemas
//   ajv-gen   the same on the schemas generated from Pkl
//   zod       the zod schemas generated from Pkl
// usage: node run.mjs <spike/pkl dir> <repo root> <generated schema dir> <generated zod .ts>
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import { Ajv2020 } from "ajv/dist/2020.js";
import { parse } from "yaml";

const [spike, repo, genDir, zodFile] = process.argv.slice(2);
const manifest = JSON.parse(readFileSync(join(spike, "conformance/manifest.json"), "utf8"));
const docs = JSON.parse(readFileSync(join(spike, "conformance/documents.json"), "utf8"));
const { schemas } = await import(pathToFileURL(zodFile).href);

const walk = (d) => readdirSync(d).flatMap((f) => (statSync(join(d, f)).isDirectory() ? walk(join(d, f)) : f.endsWith(".json") ? [join(d, f)] : []));
const load = (p) => JSON.parse(readFileSync(p, "utf8"));

function ajvFor(files) {
  // The options the repository's TypeScript loader uses (ts/src/config.ts).
  const ajv = new Ajv2020({ allErrors: true, strict: false });
  const byId = {};
  for (const f of files) {
    const d = load(f);
    ajv.addSchema(d, d.$id);
    byId[f] = d.$id;
  }
  return { ajv, byId };
}

const handFiles = [
  ...Object.values(docs).filter((d) => d.hand).map((d) => join(repo, d.hand)),
  join(repo, "schemas/service.json"),
  ...readdirSync(join(repo, "schemas/fragments")).map((f) => join(repo, "schemas/fragments", f)),
];
const hand = ajvFor([...new Set(handFiles)]);
const gen = ajvFor(walk(genDir));

const out = [];
for (const e of manifest) {
  const doc = parse(readFileSync(e.path, "utf8"));
  const d = docs[e.schema];
  const run = (name, fn) => {
    try {
      const r = fn();
      out.push({ id: e.id, validator: name, accept: r.ok, detail: r.detail ?? "" });
    } catch (err) {
      out.push({ id: e.id, validator: name, accept: false, detail: "error: " + String(err.message).split("\n")[0] });
    }
  };
  const viaAjv = (set, file) => () => {
    const validate = set.ajv.getSchema(set.byId[file]);
    const ok = validate(doc);
    return { ok, detail: ok ? "" : `${validate.errors[0].instancePath || "(root)"} ${validate.errors[0].message}` };
  };
  if (d.hand) run("ajv-hand", viaAjv(hand, join(repo, d.hand)));
  run("ajv-gen", viaAjv(gen, join(genDir, d.gen)));
  run("zod", () => {
    const r = schemas[d.gen.replace(/\.json$/, "")].safeParse(doc);
    return { ok: r.success, detail: r.success ? "" : `${r.error.issues[0].path.join(".") || "(root)"} ${r.error.issues[0].message}` };
  });
}
for (const r of out) console.log(JSON.stringify(r));
