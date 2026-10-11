# aurora

A standalone dark example theme: midnight-navy background, blue-white text, Noto Sans Display
headings, and aurora artwork behind the cover, section dividers and closing slide. It is a complete
design system on its own (`extends` is empty) and the example of a theme that **owns pictures** —
the deck never names a background; switching to aurora brings them, switching away removes them.

## Works best for

- Keynotes, launches, talks in a dark room — decks with a cover, a few section breaks and a
  closing slide, where the artwork carries the mood.
- Technical talks with diagrams and code: the accents are bright enough to read as lines and
  syntax colours on the dark background.
- Less suited to decks that will be printed, or read afterwards on paper — use `editorial` or
  `minimal` for those.

## Look

- Covers, section dividers and the closing slide stand on the **left, vertically centred**, in a
  760px column; the light in the artwork sits on the right and along the bottom. Keep those
  titles to one or two short lines so they stay on the dark side.
- Content slides have a faint violet glow in the top-right corner; quote slides a soft pink glow
  in the middle.
- Every slide fades into the next (`motion.transition`), and `enter=` runs at 700ms with a 100ms
  stagger.
- Page numbers read `3 / 12`. Footer and page number are hidden on the cover, section dividers,
  full-bleed images and the closing slide.
- Install it with `balpyo theme install <path to examples/themes/aurora>` or copy the directory to
  your project's `themes/aurora/`, then set `"theme": "aurora"` in `balpyo.json`.

## Variants — what each one is for

aurora uses the same variant names as the built-in themes, so a deck written for `minimal`
switches to aurora without warnings.

| Element | Variant | Use it for |
|---|---|---|
| `<text>` · `p` | `kicker` | The small tracked uppercase line above a title, in teal. One per slide, never a sentence. |
| `<text>` · `p` | `lead` | The opening paragraph under a heading — one or two lines, larger than body, muted. |
| `<shape>` | `card` | A grouped block of content in a `row`/`grid` of peers — a lighter navy panel with a hairline of light around it. |
| `<shape>` | `card-header` | The label strip at the top of a `card` — a violet→pink gradient. Put a short `<text color="bg">` inside. |
| `<shape>` | `card-body` | The content area under a `card-header`. |
| `<shape>` | `card-backdrop` | A translucent frame with a violet hairline around a whole group of cards or a diagram. |
| `<shape>` | `stat-card` | A tile holding one big number and its caption. |
| `<shape>` | `badge` | A teal pill for a short tag ("NEW", "v2", a status). |
| `<connector>` | `accent` | The main path through a diagram (teal). |
| `<connector>` | `dashed` | Optional, async or secondary relationships. |
| `<connector>` | `plain` | A line without an arrowhead — grouping, not flow. |
| `<image>` | `framed` | Screenshots and photos that sit on the slide rather than bleed off it. |

## Colors — how to use them

Write color names, never hex values — that is what lets a deck switch themes.

| Token | Use it for |
|---|---|
| `fg` · `bg` | Text and slide background. `color="bg"` is the dark text to put on an accent fill. |
| `muted` | Captions, secondary text, connector lines. |
| `surface` | Panels and cards, and hairlines between them. On this background, draw dividers with `surface`, not `muted` — a grey line on navy outshines the text. |
| `accent1` … `accent6` | violet · teal · green · amber · coral · pink. All six have the same contrast and brightness on `bg`, so the number is not a rank. |

`accent2` (teal — kickers, the main diagram path, big stat numbers) and `accent1` (violet — table
headers, highlights) are this theme's pair. Give the other accents one meaning per deck and keep it.

**Inverted slides** (`<slide invert>`) turn light: navy text on a blue-white background. The
accents are tuned for the dark side, so on an inverted slide use them as fills (with
`color="bg"` text), not as text colour. The artwork backgrounds stay; their overlays swap with
`bg` and become a light wash on the side where the text stands.

## Assets

The three backgrounds are the theme's own and are applied by layout — a slide never names them.

| File | Applied to | What it is |
|---|---|---|
| `assets/cover.jpg` | `layout="title"` | Aurora curtains over dunes; the light is low and on the right. |
| `assets/section.jpg` | `layout="section-divider"` | Violet and teal curtains across the frame, darkened from the left by the overlay. |
| `assets/end.jpg` | `layout="end"` | A glow low on the horizon, dark above. |

They are generated artwork, 1344×768 JPEG. To rebrand, replace a file at the same path — every
deck using aurora picks it up on the next build.
