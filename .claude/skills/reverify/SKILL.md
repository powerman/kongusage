---
name: reverify
description: >-
  Recheck the kongusage docs (docs/mapping.md, docs/usage.md, docs/mise.md, docs/design.md,
  docs/usage-features.md) against the latest releases of Kong, usage and mise,
  and report what changed and what has to be updated.
  Run manually by the owner, e.g. after new releases of Kong, usage or mise.
argument-hint: '[YYYY-MM-DD]'
disable-model-invocation: true
---

# Reverify the docs

The docs record the behaviour of the versions they name, bugs included.
The regression tests with real consumers are the main evidence:
every workaround has a test naming the versions and the condition to remove it.
This procedure finds what changed since those versions; it reports and proposes edits,
and edits nothing before the owner's OK.

It starts after the owner has updated mise (`mise self-update`) and the tools (`mise up`),
so the versions in `PATH` and in `mise.lock` are the newest ones to check,
whatever `minimum_release_age` says: the owner may have chosen a younger release on purpose.
Newer releases are held back: report them, but do not test them.

A date given as the argument replaces today (e.g. to test how the floor is raised);
the releases are still the real ones.

## Rules

- Get other versions of `usage` and `mise` by mise only, without changing `mise.toml`
  before the owner's OK: `mise exec usage@<version> -- <command>`,
  `mise install github:jdx/mise@<version>` (its binary is in
  `"$(mise where github:jdx/mise@<version>)/bin"`).
  Never `mise use`, and never `mise self-update` or `mise up`: that is the owner's choice.
- Run a `mise` other than the one in `PATH` only in the isolated environment of step 3.
- Clones of upstream repositories go into `$T`, never into this repository.
- Work and experiments go into `T=$(mktemp -d)`; remove it at the end by `rm -rf "${T:?}"`.
- At the end, `mise uninstall` the versions this run installed.

Scripts are in the base directory of this skill (`$S` below):

- `mise-usage-lib.sh <YYYY-MM-DD>`: mise releases since the date with their usage-lib versions.
- `usage-keys.sh <clone> <tag>`: nodes and properties usage-lib of the tag accepts.

## 1. Versions

The versions the docs were checked with are named at the top of each doc
and in `docs/design.md` "Supported versions"; `MinUsageVersion` (`convert.go`) is the floor.
The newest ones now are `usage --version`, `mise --version` and Kong in `go.mod`.
Newer releases:

- Kong: `go list -m -versions github.com/alecthomas/kong` (it has tags, no GitHub releases).
- usage: `gh release list -R jdx/usage`.
- mise: `"$S/mise-usage-lib.sh" <today minus three months>` (newest first, takes a minute).

Compute the new supported range by `docs/design.md` "Supported versions":
the floor is the oldest usage version among the `usage` CLI releases of the last three months
and the usage-lib of the mise releases of the last three months, but not older than 6.12.0;
the mise range starts at the oldest mise release whose usage-lib is not older than the floor.
Report whether the floor can be raised.

## 2. Changelogs

Read the changes since the versions named in the docs:

- usage: `gh api -H 'Accept: application/vnd.github.raw' repos/jdx/usage/contents/CHANGELOG.md`;
- mise: the same for `repos/jdx/mise`, only entries about usage, completion and tasks;
- Kong: `gh api repos/alecthomas/kong/compare/<old>...<new>` (commit messages and files).

Note every change of a construct used by `docs/mapping.md`, `docs/usage.md`, `docs/mise.md`
and of a node or property listed in `docs/usage-features.md`.

## 3. Tests

Run the tests against the newest consumers and the oldest supported ones.
A failing test marks a claim to recheck in step 4; a passing test proves only what it checks.

- `go test -count=1 ./...` checks the newest usage and mise (in `PATH`)
  and `MinUsageVersion` (by `mise exec`, as `mise.toml` lists it).
- A new floor, if it differs from `MinUsageVersion`: the usage in `PATH` is the current one
  of the tests, so run them with it in `PATH`:

  ```sh
  mise exec usage@<floor> -- go test -count=1 ./...
  ```

- The oldest supported mise: `TestMiseMount` runs `mise __complete_word__`
  in a temporary directory.
  Build the test binary with the usual environment, then run it in the isolated environment,
  which isolates the config too:
  another mise version may fail to parse the owner's config
  (and then completes nothing but warnings) or try to migrate it.

  ```sh
  mise install github:jdx/mise@<version>
  go test -c -o "$T/kongusage.test" .
  env MISE_CONFIG_DIR="$T/mise/config" MISE_DATA_DIR="$T/mise/data" \
      MISE_CACHE_DIR="$T/mise/cache" MISE_STATE_DIR="$T/mise/state" MISE_LOG_FILE=/dev/null \
      PATH="$(mise where github:jdx/mise@<version>)/bin:$PATH" \
      "$T/kongusage.test" -test.count=1 -test.run TestMiseMount
  ```

- Kong, if a newer one than in `go.mod` exists, without touching `go.mod`:

  ```sh
  cp go.mod go.sum "$T/"
  go get -modfile="$T/go.mod" github.com/alecthomas/kong@<version>
  go test -modfile="$T/go.mod" -count=1 ./...
  ```

## 4. Experiments

Recheck by small experiments in `$T` (`usage complete-word`, `usage lint`, `usage generate md`,
`usage diff`; for mise, a task in `$T` with `MISE_TRUSTED_CONFIG_PATHS="$T/..."`
and the isolated environment of step 3).
`mise __complete_word__ --shell bash --line 'mise run <task> '` prints the candidates,
or `files` for the fallback to files; errors go to stderr.
Read the upstream source in blobless clones in `$T` when the behaviour is unclear.
The pre-rework docs/mapping.md, with source line references and experiments per claim,
may still be in `git log --follow -p -- docs/mapping.md`.

- every item of "Kong only" of `docs/mapping.md`:
  whether the newest usage (or Kong) makes it expressible;
- every item marked "undocumented" or "a bug" and every "contrary to" in the docs;
- every claim naming a version (`grep -nE '[0-9]+\.[0-9]+\.[0-9]+' docs/*.md`), at the floor too
  and the version ranges in comments of code and tests
  (`grep -rn 'usage [0-9]' --include='*.go'`);
- every node and property usage accepts which it did not accept in the newest usage version
  named in the docs (`usage-keys.sh` on a clone of `jdx/usage`, then `diff`);
  classify each one: derived from Kong (`docs/mapping.md`),
  "Accepted" or "Not accepted" with the reason (`docs/usage-features.md`),
  or a lost Kong feature it makes expressible;
- if the floor can be raised: which nodes, properties and values that makes acceptable
  (`usage-keys.sh` for the old and the new floor, changelogs for new values),
  and which of them `docs/usage-features.md` would accept;
  which claims about versions below the new floor become moot,
  and which choices of `docs/mapping.md` were made for a version below it
  (e.g. a "Rejected" alternative whose drawback was a bug fixed since).

## 5. Per-version output

`docs/design.md` "Usage version" emits one spec, known to the floor.
Report whether any trigger to emit the spec per usage version has fired:

- a node is needed for correct completion (not a nicety),
  and a consumer inside the supported range refuses it;
- usage removed or renamed a node, or changed its meaning so that no single form
  is right on both sides of the change;
- the supported range is to be widened to consumers far behind the newest
  (e.g. mise of a distribution).

If one has, the minimal mechanism is: read `__USAGE` (the usage-lib version, set for
every mount and `run=` command) in the `--usage-spec` handler, gate only the nodes of
a small "node → minimal usage version" table, and emit the floor spec without `__USAGE`.

## 6. Upstream reports

Check the state of the reports (open or closed, fixed in which release):
the links in docs and code
(`grep -rnE --exclude-dir=.claude 'issue #|github.com/.*/(issues|pull|discussions)/'`),
and issues, pull requests and discussions of the owner in `alecthomas/kong`, `jdx/usage`
and `jdx/mise` created since 2026-11-01:

```sh
gh search issues --author=@me --repo jdx/usage --repo jdx/mise --repo alecthomas/kong \
    --created '>=2026-11-01' --json url,state,title
gh search prs --author=@me --repo jdx/usage --repo jdx/mise --repo alecthomas/kong \
    --created '>=2026-11-01' --json url,state,title
gh api graphql -f query='{search(type: DISCUSSION, first: 50,
    query: "repo:jdx/mise repo:jdx/usage author:@me created:>=2026-11-01")
    {nodes {... on Discussion {url title closed}}}}'
```

## 7. Report

Report to the owner:

1. The versions: old and new range of Kong, the `usage` CLI, usage-lib and mise;
   whether the floor can be raised.
2. Each change found: the claim (`file:line`), the evidence (changelog entry, test, experiment),
   and the proposed edit of docs, code and tests.
   A fixed bug removes a workaround only when the floor reaches the fixed version.
3. New nodes and properties with their classification; what a raised floor makes acceptable.
4. The triggers of step 5.
5. The upstream reports.

Then wait for the owner's OK.

## 8. Update

After the owner's OK, apply the agreed edits:

- the versions at the top of each doc and in `docs/design.md` "Supported versions",
  the "Kong only" heading of `docs/mapping.md`, version ranges in comments;
- for a new floor: `MinUsageVersion`, `usage = ['latest', '<floor>']` in `mise.toml`,
  then `mise install` (check that `git diff mise.lock` changes only usage);
  the floor is also in `min_usage_version` of the golden files (`testdata/*.kdl`)
  and in messages expected by tests.

Then check the "Accepted" section of `docs/usage-features.md` against `usagekdl.Specs`:

1. If the format of the section changed
   (the `## Accepted` heading, `<Title>:` subsection lines, backticked names,
   properties in parentheses), update `TestSpecs` (`internal/usagekdl/spec_test.go`) to match;
   the subsection titles are hardcoded in `acceptedFeatures()`.
2. Run `KONGUSAGE_TEST_DOCS=1 go test -count=1 -run TestSpecs ./internal/usagekdl/`.
   A failure means the doc and `usagekdl.Specs` disagree: fix the side which is wrong.

Finally run `mise run` (format, lint and tests).
