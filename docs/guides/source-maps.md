# Source maps for a page

A guide, not a contract: it walks the url-shortener's front end and may change
without a version bump.

A minified error (`index-abc.js:1:48211`) is unreadable. A source map turns it
back into `src/ui/UrlsView.tsx:42`. The map is also the whole source, so where
it lives decides who can read it.

## The rule

**A map is never in the image.** The image copies all of `dist/`, and the
server serves any file under it, so a map left there is published to every
visitor.

## The build

GoReleaser builds the page **once**, in a before hook, and the web image copies
the result. The hook does three things:

1. `vite build` writes maps with `build.sourcemap: "hidden"`: the files exist,
   but the bundles carry no `sourceMappingURL` comment and a browser never asks
   for them. (Alloy asks the map server by the script's own path, so it needs
   no comment.)
2. `scripts/sourcemaps.ts` moves every `*.map` to `dist-sourcemaps/` at the
   repository root, keeping the path the script is served under:
   `dist-sourcemaps/assets/index-<hash>.js.map`. That is the layout `smctl
   push --maps` expects.
3. The same script then **fails the build** if any `*.map` is left anywhere
   under `dist/`. It is a test (`scripts/sourcemaps.spec.ts`) and a build step;
   without the failure a stray map would be silent.

## The version

The release version is the one identifier shared by the image tag, Faro's
`app.release` and the maps' tag. The before hook sets
`VITE_APP_VERSION={{ .Version }}` (the tag without its `v`), the Vite build
bakes it into the bundle, and the page reports it as Faro's `app.version` and
`app.release`. A build without it (`yarn build` on a laptop) reports `dev`.
`smctl push` reads the same version from GoReleaser's `dist/metadata.json`
(replacing `+` with `_` in the tag), so the two cannot differ.

## Pushing

The release workflow passes `sourcemaps-image: web` and `sourcemaps-app:
url-shortener` to the shared `release-public` workflow. Right after GoReleaser,
in the same job, it runs `smctl push --goreleaser-dist dist --image web --app
url-shortener --maps dist-sourcemaps`. The artifact (an OCI artifact of type
`application/vnd.ocictl.sourcemaps.v1`, one layer holding the `.map` files)
lands in

    ghcr.io/<owner>/sourcemaps/url-shortener:<release version>

pushed with the workflow's own token, so the existing `packages: write` is all
it needs. The image is called `web` but the application, and the Faro app
name, is `url-shortener`; the app override is what keeps the repository named
after the application. Because the push is in the job that published the image,
maps exist only for a version whose image was published, from the same build; a
failed push fails the release.

The step is **off** until the repository variable `SOURCEMAPS_ENABLED` is
`true`; with it unset, the release is what it was. The package is created
private by ghcr: make it public (or grant the reader) for the map server to
pull it.

## Reading them

`smctl serve` (truvity/ocictl) answers Alloy's `faro.receiver` from that
repository: `GET /<app>/<release>/assets/index-<hash>.js.map`. That service,
and its configuration, belong to whoever runs the collector; this repository
only produces the artifact. Retention is the registry's.
