# ink

A dark theme: ink-black background, light text, bright accents, Noto Sans. It is a dark theme in
its own right, not an automatic dark mode of `minimal`.

## Works best for

- Conference stages, dark rooms, live demos, product launches.
- Code-heavy talks — the `code` layout and syntax colors are tuned for the dark background.
- Avoid it for decks that will be printed or handed out; use `minimal` for those.

## Look

- Covers center the title group vertically, left-aligned.
- Cards are `surface` panels a step lighter than the background, with a soft shadow;
  `card-header` is filled with `accent1`.
- Rounder corners than the other built-ins (`radius` sm 8 · md 16 · lg 28).
- Accents are bright for contrast on dark — one per slide is plenty.

## Variants — what each one is for

The variant names are the vocabulary every built-in theme shares, so a deck that uses them
re-skins cleanly with `--theme`.

| Element | Variant | Use it for |
|---|---|---|
| `<text>` · `p` | `kicker` | The small uppercase line above a title (`slot="kicker"`). One per slide, never a sentence. |
| `<text>` · `p` | `lead` | The opening paragraph under a heading — one or two lines, larger than body. |
| `<shape>` | `card` | A grouped block of content in a `row`/`grid` of peers. A lighter `surface` panel with a soft shadow — the default container for grouped content. |
| `<shape>` | `card-header` | The label strip at the top of a `card`. Put a short `<text>` inside. |
| `<shape>` | `card-body` | The content area under a `card-header`. |
| `<shape>` | `card-backdrop` | An outlined frame around a whole group of cards or a diagram — the "this is one thing" box. |
| `<shape>` | `stat-card` | A tile holding one big number and its caption (with the `stat` layout or a row of KPIs). |
| `<shape>` | `badge` | A pill-shaped tag or label ("NEW", "v2", a status). Short text only. |
| `<connector>` | `accent` | The main path through a diagram. |
| `<connector>` | `dashed` | Optional, async or secondary relationships. |
| `<connector>` | `plain` | A line without an arrowhead — grouping, not flow. |
| `<image>` | `framed` | Screenshots and photos that sit on the slide rather than bleed off it. |

## Colors — how to use them

Write color names, never hex values — that is what lets a deck switch themes.

| Token | Use it for |
|---|---|
| `fg` · `bg` | Text and slide background. |
| `muted` | Captions, secondary text, de-emphasized lines. |
| `surface` | Panels and cards that sit on the background. |
| `accent1` … `accent6` | blue · teal · green · amber · coral · violet. All six have the same contrast and brightness, so the number is not a rank. |

`accent1` is this theme's signature accent — the kicker, badge and backdrop already use it. Give
the other accents a meaning per deck (say, `accent3` for "shipped", `accent4` for "at risk"), use two
or three of them, and keep each meaning the same on every slide.

## Assets

None — ink ships no pictures. Screenshots read best inside `framed`; dark-UI screenshots blend
in better than light ones.
