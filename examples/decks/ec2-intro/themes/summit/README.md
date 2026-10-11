# summit

A cloud-keynote look built on `minimal`: squid-ink text on white, one orange accent, tracked
uppercase kickers, a soft orange glow in the corner of title, section and end slides, a fade
between slides and an `n / total` page number.

summit only renames colors and sets rhythm. Layouts, roles and variant names are `minimal`'s — run
`balpyo theme info minimal` for what each variant is for.

## Works best for

- Product and service introductions, architecture walkthroughs, launch-style talks.
- Decks that alternate section dividers with dense content slides — the glow marks the
  dividers, the content slides stay plain white.

## Using it

- Let the orange (`accent1`) mark one thing per slide: the key number, the service being
  introduced, the main path in a diagram (`<connector variant="accent">`).
- Use the other accents for categories in diagrams and charts, not for emphasis.
- Do not add your own background to `title`, `section-divider` or `end` slides — the glow
  is the theme's signature there.

## Assets

None — summit ships no pictures. Put service icons and diagrams in the deck's `assets/`.
