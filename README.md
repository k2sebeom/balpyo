<p align="center">
  <img src="docs/assets/logo.png" alt="balpyo logo" width="200">
</p>

# balpyo

**English** · [한국어](README.ko.md)

Write slides in **SOML** (Slide Oriented Markup Language), and **balpyo** applies a theme and renders
both a **web presentation** and a **native, editable PPTX**.

![A cover rendered by balpyo — example theme aurora](examples/showcase/images/themes/balpyo-intro-01-aurora.png)

- **Markup made for slides.** `<slide>` · `<row>` · `<grid>` · `<shape>` · `<table>` · `<chart>` · `<note>` —
  the vocabulary of slides is first-class tags. The vocabulary is closed, so typos and structural
  mistakes come back as diagnostics with a `line:col`.
- **No coordinates.** You declare layout with flex and grid; the boxes a headless browser measures
  become the PPTX coordinates as-is.
- **Real PPTX.** Text boxes are text boxes, shapes are preset shapes, tables are editable tables, and
  slots are placeholders — slides are never baked into pictures. Colors stay theme-color references,
  so changing the theme colors in PowerPoint carries through.
- **Themes are design systems.** One package of tokens · component appearance · variants · layout
  placement compiles to both web CSS and a PPTX theme. Switch themes without touching a single line
  of your slides.
- **One binary.** The web output is a single `index.html` with no external references — presenter
  view · overview · notes · deep links · print (PDF) included.

## Showcase

Below are the repository's [example decks](examples/decks/) exactly as balpyo renders them. The
outputs (`index.html` · `.pptx`) are in [`examples/showcase/decks/`](examples/showcase/decks/) too —
download a PPTX and open it in PowerPoint or Keynote, and every word and shape is editable.

| | |
|---|---|
| ![The build path drawn with connectors](examples/showcase/images/balpyo-intro-04.png) | ![An incident timeline on a proportional time axis](examples/showcase/images/incident-postmortem-03.png) |
| A `<connector>` diagram — in PPTX these become connectors glued to the boxes · [`balpyo-intro`](examples/decks/balpyo-intro) | A proportional time axis — flag cards on a `<stack>` · [`incident-postmortem`](examples/decks/incident-postmortem) (`ink`) |
| ![KPI cards drawn with paths, bars and stars](examples/showcase/images/quarterly-review-03.png) | ![A photo and a funnel](examples/showcase/images/quarterly-review-06.png) |
| `type="path"` curves · bars · `star` — all editable shapes · [`quarterly-review`](examples/decks/quarterly-review) | The `image-text` layout — the picture becomes a PPTX placeholder |
| ![Contributing factors converging on one outcome](examples/showcase/images/incident-postmortem-06.png) | ![A Go code window](examples/showcase/images/incident-postmortem-05.png) |
| Curved connectors converge on one outcome | `<pre lang="go">` — highlighting becomes PPTX text runs |
| ![A table and a chart](examples/showcase/images/quarterly-review-04.png) | ![A bar chart with two series](examples/showcase/images/incident-postmortem-08.png) |
| `<table>` and `<chart>` — both editable in PPTX | A two-series `<chart>` with a legend |
| ![A cover with a radial glow](examples/showcase/images/ec2-intro-01.png) | ![Nitro as stacked isometric layers](examples/showcase/images/ec2-intro-06.png) |
| A backdrop (`radial()`) and gradient text · [`ec2-intro`](examples/decks/ec2-intro) | `cube` · `right-brace` presets — the same shapes in PPTX |
| ![An instance state diagram](examples/showcase/images/ec2-intro-10.png) | ![A highly available web tier](examples/showcase/images/ec2-intro-11.png) |
| A state machine — straight · elbow · U-turn connectors | Nested regions and flowing connectors (`flow`) |

### Same slide, different themes

The slide files stay the same; only the theme changes (`balpyo build --theme <name>`). Slides use
**token names** like `accent1` · `surface` and **variants** like `card` instead of colors, so when
the theme that defines those values changes, the same source becomes a different deck.

```xml
<slide id="geometry" enter="rise">
  <text variant="kicker">Geometry pass</text>
  <h1>The browser computes the coordinates</h1>
  <grid cols="3" gap="lg" grow="1">
    <shape variant="card" type="rounded-rect" w="fill" pad="md">
      <col gap="sm" h="fill" justify="between">
        <shape variant="card-header" type="rounded-rect" fill="accent1" w="fill" h="44" pad="xs">
          <text color="bg" align="center" anchor="center"><strong>① Declare</strong></text>
        </shape>
        <pre lang="soml" size="14">&lt;row gap="md"&gt; …</pre>
        <p size="20">Authors declare layout with flex and grid. Coordinates have no place in it.</p>
      </col>
    </shape>
    <!-- … two more cards … -->
  </grid>
</slide>
```

| `minimal` (default) | `editorial` |
|---|---|
| ![minimal](examples/showcase/images/themes/balpyo-intro-06-minimal.png) | ![editorial](examples/showcase/images/themes/balpyo-intro-06-editorial.png) |
| **`ink`** | **`harbor`** ([example theme](examples/themes/harbor)) |
| ![ink](examples/showcase/images/themes/balpyo-intro-06-ink.png) | ![harbor](examples/showcase/images/themes/balpyo-intro-06-harbor.png) |
| **`aurora`** ([example theme](examples/themes/aurora)) | **`aurora`** — a photo slide from another deck |
| ![aurora](examples/showcase/images/themes/balpyo-intro-06-aurora.png) | ![aurora's image-text](examples/showcase/images/themes/quarterly-review-06-aurora.png) |

A theme owns not just colors but **placement** and **pictures**, too. The same cover source sits in
the center in minimal, at the bottom in editorial, middle-left in ink, and top-left in harbor — the
cover has no alignment in it at all. aurora even brings its own background art for the cover and
closing slides: the deck never references a picture.

| `minimal` | `editorial` |
|---|---|
| ![minimal cover](examples/showcase/images/themes/balpyo-intro-01-minimal.png) | ![editorial cover](examples/showcase/images/themes/balpyo-intro-01-editorial.png) |
| **`ink`** | **`harbor`** |
| ![ink cover](examples/showcase/images/themes/balpyo-intro-01-ink.png) | ![harbor cover](examples/showcase/images/themes/balpyo-intro-01-harbor.png) |
| **`aurora`** — cover | **`aurora`** — closing slide |
| ![aurora cover](examples/showcase/images/themes/balpyo-intro-01-aurora.png) | ![aurora closing](examples/showcase/images/themes/balpyo-intro-12-aurora.png) |

## Quick start

You need Go 1.26 and Chrome/Chromium (to measure the PPTX layout).

```bash
git clone https://github.com/k2sebeom/balpyo && cd balpyo
go build -o balpyo ./cmd/balpyo

./balpyo init my-talk --author "Your Name"
./balpyo watch --project my-talk                     # http://127.0.0.1:4600 — redraws on save
./balpyo build --project my-talk --target html,pptx  # → my-talk/dist/index.html + <title>.pptx
./balpyo check --project my-talk --check-layout --strict   # also checks overflow · off-stage · overlap (for CI)
```

A project is one config file and a slide directory. File order is slide order.

```text
my-talk/
├── balpyo.json          # title · author · aspect ratio · theme · footer · page numbers
├── slides/
│   ├── 001_title.slide
│   └── 002_content.slide
├── assets/              # images referenced by <image src>
└── themes/              # your own themes (optional)
```

```xml
<slide id="cover" layout="title">
  <text slot="kicker" variant="kicker">SLIDE ORIENTED MARKUP LANGUAGE</text>
  <h1 slot="title">balpyo</h1>
  <text slot="subtitle" role="heading" color="muted">Write once, get web + editable PPTX</text>
  <note>Speaker notes go here.</note>
</slide>
```

## Further reading

The documentation is currently written in Korean.

- [Overview](docs/index.md) — what balpyo is and how it works
- [Getting started](docs/guides/getting-started.md) — from install to presenting
- [Reference](docs/reference/index.md) — tags · appearance attributes · layouts · theme packages · CLI
- [Choosing and making themes](docs/guides/theming.md) — the three built-in themes and your own
- [Examples](examples/) — four finished decks and two standalone themes

## Contributing

Documentation for people changing balpyo lives in [`contributing/`](contributing/) — architecture,
the geometry pass, testing, design decisions (ADRs), and the backlog. The gate is a single
`go test ./...`. When rendered output changes, regenerate the showcase above with `make showcase`
and commit it alongside.
