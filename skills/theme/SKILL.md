---
name: theme
description: Design a Balpyo theme — a project-local design system (tokens, component appearance, layout CSS, deck chrome) that re-skins a deck without touching a single slide. Use when the user wants custom colours, fonts, or layout for a Balpyo deck, asks for brand/corporate styling of slides, or mentions a balpyo theme, tokens.json, or components.json.
---

# Designing a Balpyo theme

A theme is **one design system**: the table that turns the author's meanings (`role="title"`,
`color="accent1"`, `gap="md"`) into values. Swap it and the same slides look like a different
deck, with no edit to the markup.

Two facts set the shape of the work:

- **There is no fallback theme.** Every colour, font, size, and placement comes from the theme
  (the engine owns only the stage size and margin, list indents, and link colour). Anything a
  theme fails to supply stops the build *before any slide renders* — a missing piece is never a
  half-styled slide.
- **The theme owns values and placement; the author owns meaning.** So a theme never contains
  slide content, and layout CSS never contains colour.

This skill and its reference files are complete on their own. The binary must be on PATH
(`balpyo version`); see the `deck` skill for installing it and for the SOML the decks are
written in. `--check-layout` and the `pptx` target also need Chromium or Chrome.

## 1. Inherit — don't start from scratch

A theme being designed lives **inside the project**, and the directory name *is* the theme name:

```text
my-deck/
├── balpyo.json          # { "theme": "corp" }
├── slides/
└── themes/
    └── corp/
        ├── theme.json         # { "name": "corp", "extends": "minimal", … }
        ├── tokens.json        # partial, when inheriting
        ├── components.json    # partial
        ├── layouts/           # only the files being replaced
        ├── assets/            # optional — background/logo (PNG/JPEG), pictures for theme: (PNG/JPEG/SVG)
        └── README.md          # for deck authors — what the theme is for, how to use its variants and assets
```

Pick the closest built-in as the parent and override only what differs:

| Parent | Voice | Pick it when |
|---|---|---|
| `minimal` | neutral business document, white bg, sans | default — anything, when unsure |
| `editorial` | print page, paper-coloured bg, **serif** titles that sit low | text-heavy decks meant to be read afterwards |
| `ink` | dark room, near-black bg | stage, dark rooms, code-heavy decks |

**Read the parent before overriding it.** `references/builtin-themes.md` holds the complete
files of all three built-ins. Overriding is by whole entry — re-declaring the role `title`, the
variant `card`, or the chrome `footer` replaces every field of it — so copy the parent's entry
and change the fields you mean. `balpyo theme info <parent>` prints the parent's README.

A standalone theme (`"extends": ""`) has to supply everything itself: all three JSON files
complete (10 colours, 3 fonts, 6 spaces, 3 radii, 1 shadow, 5 text roles, and `code.tabSize`) and
a CSS file for every layout it declares — which should be all eleven, with slot CSS for the ten
slotted ones. Only choose that when the design genuinely shares nothing with a built-in;
inheriting is otherwise a few dozen lines.

Two naming rules, both hard errors: `theme.json`'s `name` must equal the directory name, and a
local theme **may not shadow a built-in name** (`themes/minimal/` is an error even when
building `--theme ink`). There is no path syntax — the value is always a name.

A name resolves in **built-in ∪ project `themes/<name>/` ∪ installed themes** (step 5). Run
`balpyo theme list --project my-deck` before picking a name: a project theme and an installed
theme with the same name stop every build that requests that name.

## 2. Write the design spec before the files

State, in your reply: the voice in one line; background and foreground; the heading/body/mono
font families; the six accents as hex with their intended meanings; which layouts get new
placement; and where the footer and page number sit. Then build the files from that.

The reason is the palette. **Six accents are a measured set, not a ranking** — they must
satisfy three numbers, and *nothing in the toolchain checks a project theme's palette*. So
**compute them yourself** before writing `tokens.json`:

| Rule | Number | What it prevents |
|---|---|---|
| Contrast against `bg` | ≥ 4.5:1 | unreadable colour. Contrast is symmetric, so this covers both accent-on-bg text and bg-on-accent text |
| Pairwise hue gap | ≥ 25° | two colours nobody can tell apart (projectors and colour vision deficiency break first) |
| Relative-luminance band | max/min ≤ 1.25 | one accent flaring out, so *brightness* rather than *number* creates hierarchy |

Use WCAG 2.1 relative luminance, and show the numbers in your reply so the user can see the
palette holds. For reference the built-ins measure: `minimal` 4.76–5.38 / band 1.18;
`editorial` 4.92–5.00 / 1.02; `ink` 5.93–6.00 / 1.02. A yellow that reaches 4.5:1 on white is
already brown — on a dark background the ceiling moves the other way.

**Accent numbers are not a hierarchy.** `accent2` is the neighbour of `accent1` on the colour
wheel, not the second-most-important colour. Hierarchy comes from the *deck* using two or three
of them with one meaning each.

## 3. Write the files

Every key, value grammar, and error: `references/package.md`. Shortest useful version:

- **`theme.json`** — `name`, `extends`, optional `layouts` (inherited, so don't re-list the
  eleven), `chrome` (footer and page-number placement — `align` and `role` are mandatory — and
  an optional `logo`), and optional `backgrounds` (per-layout slide backgrounds: a colour,
  token gradient, or picture from `assets/`, with an optional `overlay` layer on top, such as a
  photo darkened by `gradient(fg fg/0, 90deg)` or a corner glow `radial(accent1/35 bg/0, at 0% 100%)`).
  `pageNumber` can take a `format` such as `"{n} / {total}"`. Backgrounds and the logo are the theme's own: the
  deck never mentions them, so a theme swap swaps them too. Slides can also use any picture in
  `assets/` with `<image src="theme:<path under assets/>">` — the way to ship an icon library
  (SVG icons are fine there). `theme.json` keys are **not** checked for typos — a misspelt key
  is silently ignored, so compare against `package.md`.
- **`README.md`** — not read by the loader, but printed verbatim by `balpyo theme info <name>`,
  and deck authors (and the `deck` skill) read it before writing slides. It holds the **intent**
  that the other files can't: the voice of the theme, which decks it suits (and which it doesn't,
  with the theme to use instead), layout traits an author must know (e.g. "covers sit bottom
  left"), what each variant is *for*, how to split the accents, and, for every picture under
  `assets/`, its `theme:` path and what to use it for. Write uses, not values. When extending,
  cover only what you changed and point at the parent. The file name is exactly `README.md`.
  The built-in themes' READMEs are the model (`balpyo theme info minimal`); `package.md` has an
  example.
- **`tokens.json` `motion`** — the deck's animation rhythm (`duration`, `stagger`, `delay` in ms)
  and an optional default slide `transition` (`fade` · `push`). Optional for loading, but
  without it every deck that writes `enter=` fails. When inheriting, writing it replaces the
  parent's whole object.
- **`tokens.json`** — the **names** authors may write. Each colour needs all four fields
  (`name`, `rgb`, `slot`, `ref`). Colour work in a child theme is **recolouring by name**: the
  `clrScheme` has 12 slots, built-ins use 10, so adding a new colour *name* requires a free
  slot. Space, radius, shadow, and font have no such ceiling — add freely. Fonts need exactly
  one `major` (headings) and one `minor` (body) token.
- **`components.json`** — all five text roles complete (`size`, `lineHeight`, `align`, plus
  `font`/`color` naming tokens **that exist in this theme**), defaults per tag *or* per layout
  slot (`quote`/`quote`, `stat`/`value`, `stat`/`label`), variants, table cell appearance, and
  code-token colours. Every colour in the file must reference this theme's tokens. Defaults and
  variants may use the authoring grammar — `gradient(accent1 accent6, 135deg)`,
  `tint(accent1, 40%)` — so a deck says `variant="hero"` and the theme owns the gradient; roles,
  table cells, and code tokens take a single token name. A `#RRGGBB` is a load error even if no
  slide uses that variant. Unknown keys are errors in both `tokens.json` and
  `components.json`; any key starting with `$comment` (`$comment-accent`, …) is a comment,
  anywhere. Text variants and defaults set size, weight, colour, `tracking`, and `case` — not
  font or line height (those belong to roles). `tracking`/`case` are how kickers get tracked
  upper case (authors have no attribute for them); `text.mark.highlight` is the `<mark>` colour;
  `connector` holds the default line and end markers plus variants; shape fills may be
  `pattern(…)` hatches.
- **`layouts/<name>.css`** — a **selector-less declaration list** for flow mode, and
  `<name>.slots.css` for slot mode. **Placement only**: colour and font declarations here are
  warnings and never reach the PPTX master. Each slot file must carry its own `display: grid`
  and all of that layout's slot names as `grid-template-areas`. Use literal px in layout CSS
  rather than `var(--balpyo-space-…)`.

Six rules nothing in the toolchain will catch for you:

- **Centre costs slack; left is free.** In slot CSS use `justify-items: center` only where
  centring *is* the meaning (cover, a single `stat` number, the closing slide). Never write
  `justify-items: start` — left is already the default, and it shrinks the box to content
  width, so the PPTX box has zero slack and a line the viewer sets slightly wider gets
  re-broken and shrunk.
- **The `mono` token's `ea` must be a monospace CJK face** (`Noto Sans Mono CJK KR`). Pair a
  monospace Latin with a proportional CJK face and one Korean comment line is drawn 7.9%
  narrower in the browser than in PowerPoint.
- **Every font you name must be installed where the deck is built.** The loader does not check
  this, but every measuring build warns per missing family (`theme "corp" font "Malgun
  Gothic" (heading.ea, body.ea) is missing from the measuring browser — …`) and `--strict` fails on it — the boxes
  were measured with a substitute that PowerPoint will not use. Themes cannot bundle fonts.
  Pick fonts the user actually has (and their CI), or tell them what to install.
- **`clrScheme` slots are a convention, not a brightness.** A dark theme still puts `fg` in
  `dk1` and `bg` in `lt1`, or PowerPoint's theme-colour editor comes out inverted.
- **On a dark background use `surface`, not `muted`, for fills and lines** (that is what
  `ink`'s table borders do) — a grey line on black outshines the text.
- **Design for the inverted slide too.** `<slide invert>` swaps `bg` ↔ `fg` and `surface` ↔
  `muted` and leaves the six accents alone. So on a deck's dark slide the accents sit on your
  `fg` colour. Check their contrast against `fg` as well; the 4.5:1 rule above only covers `bg`.

Keep variant names identical to the built-in vocabulary (text `kicker` `lead`; shape `card`
`card-header` `card-body` `card-backdrop` `stat-card` `badge`; image `framed`; connector
`dashed` `accent` `plain`; table `striped` is built in and only needs `table.stripe`), and keep
the three sections decks lean on: `motion`, `text.mark`, and `connector.default`. A standalone
theme that drops one breaks every deck using `enter`, `<mark>`, or `<connector>`.
Sharing the vocabulary is the portability contract: if `variant="stat-card"` turns into a
warning after a theme swap, "looks different" becomes "the deck broke". Add new variants on top
of those, don't rename them.

## 4. Verify

The loader is the first gate. Run it against a real deck — the theme cannot be judged
alone:

```bash
balpyo check --project my-deck --theme corp --check-layout --strict
```

Every malformed theme fails here, before any slide renders, so a theme error never shows up as
"the text is missing". Work through `references/checklist.md` for what the loader requires,
what it does not check (so your theme must self-police), and the hand-checked rules above.

If the project has no deck yet, make one that exercises every section: a cover, a section
divider, a content slide with a card grid, a `two-column`, a `stat`, a `quote`, a `code` slide
with `<pre lang>`, a table with a header row and `variant="striped"`, a chart, connectors between
boxes, a `<mark>`, `enter=` on a slide, an `<slide invert>`, and an `end` slide.

Then look at it, because values change geometry — a bigger title adds a line and the line
overflows. `balpyo snap --project my-deck --theme corp` draws the web layout to PNGs with no
browser; read them. For the PPTX side, the gallery needs OnlyOffice Document Builder or
LibreOffice + `pdftoppm` (see the `deck` skill); without them it is web-only and says so:

```bash
balpyo preview --project my-deck --theme corp --gallery preview-corp
```

Build the same deck under the parent theme into a second gallery (`--gallery preview-minimal`)
and compare: what stayed identical is what you did *not* claim, and that sameness is the
evidence inheritance worked. `--gallery` may not overlap `--out` — `preview` refuses before
building.

When the user is happy, write the name into `balpyo.json` (`{ "theme": "corp" }`) — until then
keep it on the flag so reverting is free. (The `balpyo.json` theme must still resolve while you
use `--theme`.)

## 5. Share it — install, don't copy

When several decks should use the theme, move `themes/<name>/` out of the project into its own
directory or GitHub repository and **install** it. `balpyo.json` keeps holding just the name.

```bash
balpyo theme install github.com/acme/slide-themes@v1.2.0      # public GitHub only; no git needed
balpyo theme install ./slide-themes                           # a local directory — copied, not linked
balpyo theme install github.com/acme/slide-themes acme-dark   # one theme + its same-repo extends parents
balpyo theme list --project my-deck                           # built-in / installed / project, overlaps flagged
balpyo theme info <name> --project my-deck                    # where the theme comes from + its README.md
balpyo theme update [name...]                                 # re-fetch / re-copy from the recorded source
balpyo theme remove <name...>
```

- Installed themes land in `~/.balpyo/themes/<name>/` (`$BALPYO_HOME/themes/` if set), sources in
  `~/.balpyo/themes.json`. The **name comes from `theme.json`**, not the repo or directory name,
  and must be letters, digits, `_`, `.`, `-` (starting with a letter or digit).
- Addresses: `github.com/owner/repo`, `https://github.com/owner/repo.git`,
  `git@github.com:owner/repo.git`, `@branch|tag|commit`, `/sub/path` to search one subtree
  (`github.com/acme/themes/brand/dark@v1`), or a `…/tree/<ref>/<path>` URL. `./x`, `../x`, `/x`,
  `~/x` are always paths.
- Install finds every directory holding a `theme.json` under the source, skipping `.`-prefixed
  directories. It loads each with the build's own loader and is **all-or-nothing**: a built-in
  name, two themes of one name in the source, a symlink inside a theme, or a theme that does not
  load installs nothing. A name already installed from a different source needs `--force`.
- **An installed theme can extend only built-in or installed themes** — never a project theme.
  Publish parents together with their children.
- **Delete the project's `themes/<name>/` after installing** — the same name in both places is
  a build error (`theme "acme" exists both in the project's themes/acme/ and among installed themes — rename the local theme or remove the installation with balpyo theme remove acme`).
- A path install is a **snapshot**: edits to the source show up only after `balpyo theme update`
  (and `watch` does not see them — save a project file or restart it). While still iterating,
  keep the theme in the project's `themes/`.
- **Builds never touch the network.** On another machine or in CI, run the same
  `balpyo theme install` first; otherwise the build stops with `unknown theme "…"`. Private repos
  and non-GitHub hosts: check them out and install the directory by path.
- **Ship a `README.md` in every theme directory.** It is what `balpyo theme info` shows to the
  people installing it, and the same file GitHub shows on the repository page. In a repository
  holding several themes, each theme directory gets its own.
- Layout-CSS appearance-property warnings for an installed theme are reported **at install
  time**, not by builds — so fix them before publishing; deck authors can't.
- `theme remove` refuses while another installed theme extends the one being removed.

## Reference files

- `references/package.md` — the theme package in full: every key of every file, value
  grammars, the slot names per layout, the `clrScheme` slot table, chrome, backgrounds and
  overlays, `motion`, typography, `mark`, `connector`, tables, code, the `extends` merge rule,
  and complete worked inheriting themes.
- `references/checklist.md` — what the loader stops on, what it lets through, and the
  hand-checked rules, as a pass-before-delivery list.
- `references/builtin-themes.md` — the exact files of `minimal`, `editorial`, and `ink`.
