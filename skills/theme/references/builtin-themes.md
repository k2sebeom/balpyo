# Built-in theme files

The exact files of the three built-in themes (`minimal`, `editorial`, `ink`) as this
version of the binary embeds them, with comments removed. Use them to see what a child theme
inherits and to copy a complete entry before changing it: re-declaring a role, variant, token or
chrome kind replaces **that whole entry**, so start from the parent's entry and edit the fields
you mean to change. `balpyo theme info <name>` prints each theme's README (what it is for).

Files that hold only placement are listed per layout: `<layout>.css` is flow mode and
`<layout>.slots.css` is slot mode. `canvas` has no slot file.

## `editorial`

### `theme.json`

```json
{
  "name": "editorial",
  "extends": "",
  "layouts": [
    "title", "section-divider", "content", "two-column",
    "image-full", "image-text", "quote", "code", "stat", "canvas", "end"
  ],
  "chrome": [
    {
      "kind": "footer",
      "align": "start",
      "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"]
    },
    {
      "kind": "pageNumber",
      "align": "end",
      "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"]
    }
  ]
}
```

### `tokens.json`

```json
{
  "color": [
    { "name": "fg",      "rgb": "1C1A17", "slot": "dk1",     "ref": "tx1" },
    { "name": "bg",      "rgb": "FAF8F4", "slot": "lt1",     "ref": "bg1" },
    { "name": "muted",   "rgb": "6A6459", "slot": "dk2",     "ref": "tx2" },
    { "name": "surface", "rgb": "F0EBE3", "slot": "lt2",     "ref": "bg2" },
    { "name": "accent1", "rgb": "585CDC", "slot": "accent1", "ref": "accent1" },
    { "name": "accent2", "rgb": "30776C", "slot": "accent2", "ref": "accent2" },
    { "name": "accent3", "rgb": "487730", "slot": "accent3", "ref": "accent3" },
    { "name": "accent4", "rgb": "7D6B32", "slot": "accent4", "ref": "accent4" },
    { "name": "accent5", "rgb": "A25741", "slot": "accent5", "ref": "accent5" },
    { "name": "accent6", "rgb": "B04774", "slot": "accent6", "ref": "accent6" }
  ],

  "font": [
    { "name": "heading", "latin": "Noto Serif",     "ea": "Noto Serif CJK KR",     "generic": "serif",     "scheme": "major" },
    { "name": "body",    "latin": "Noto Sans",      "ea": "Noto Sans CJK KR",      "generic": "sans-serif", "scheme": "minor" },
    { "name": "mono",    "latin": "Noto Sans Mono", "ea": "Noto Sans Mono CJK KR", "generic": "monospace" }
  ],

  "space": [
    { "name": "xs",  "px": 8 },
    { "name": "sm",  "px": 16 },
    { "name": "md",  "px": 24 },
    { "name": "lg",  "px": 40 },
    { "name": "xl",  "px": 56 },
    { "name": "2xl", "px": 80 }
  ],

  "radius": [
    { "name": "sm", "px": 0 },
    { "name": "md", "px": 2 },
    { "name": "lg", "px": 4 }
  ],

  "shadow": [
    { "name": "soft", "dist": 2, "dir": 90, "blur": 8, "color": "fg", "alpha": 0.12 }
  ],

  "motion": { "duration": 600, "stagger": 90, "delay": 150 }
}
```

### `components.json`

```json
{
  "text": {
    "roles": [
      { "name": "title",   "font": "heading", "size": 60, "weight": 400, "lineHeight": 1.2, "color": "fg",    "align": "start" },
      { "name": "heading", "font": "heading", "size": 32, "weight": 700, "lineHeight": 1.3, "color": "fg",    "align": "start" },
      { "name": "body",    "font": "body",    "size": 24, "weight": 400, "lineHeight": 1.5, "color": "fg",    "align": "start" },
      { "name": "caption", "font": "body",    "size": 18, "weight": 400, "lineHeight": 1.4, "color": "muted", "align": "start" },
      { "name": "code",    "font": "mono",    "size": 20, "weight": 400, "lineHeight": 1.5, "color": "fg",    "align": "start" }
    ],
    "componentDefaults": [
      { "tag": "h3",         "size": 26 },
      { "tag": "blockquote", "color": "muted" },
      { "layout": "quote", "slot": "quote", "size": 40, "color": "fg" },
      { "layout": "stat",  "slot": "value", "size": 96, "weight": 700, "color": "accent5" },
      { "layout": "stat",  "slot": "label", "size": 28, "weight": 700 }
    ],
    "mark": { "highlight": "tint(accent4, 70%)" },
    "variants": [
      { "name": "kicker", "size": 17, "weight": 700, "color": "accent5", "tracking": 0.1, "case": "upper" },
      { "name": "lead",   "size": 30 }
    ]
  },

  "shape": {
    "default": { "fill": "surface", "border": "1 solid muted", "radius": "sm" },
    "variants": [
      { "name": "card",          "fill": "bg",     "radius": "sm" },
      { "name": "card-header",   "border": "none", "radius": "sm" },
      { "name": "card-body",     "fill": "bg" },
      { "name": "card-backdrop", "fill": "none",   "border": "2 solid accent5", "radius": "sm" },
      { "name": "stat-card",     "border": "none", "radius": "md", "shadow": "soft" },
      { "name": "badge",         "border": "1 solid accent5", "radius": "lg" }
    ]
  },

  "connector": {
    "default": { "line": "2 solid muted", "start": "none", "end": "arrow" },
    "variants": [
      { "name": "dashed", "line": "2 dashed muted" },
      { "name": "accent", "line": "2 solid accent1" },
      { "name": "plain",  "end": "none" }
    ]
  },
  "image": {
    "default": {},
    "variants": [
      { "name": "framed", "border": "1 solid fg", "radius": "sm", "shadow": "soft" }
    ]
  },

  "table": {
    "cell":   { "border": "1 solid muted" },
    "header": { "fill": "accent1", "color": "bg", "weight": 700 },
    "stripe": "surface"
  },

  "code": {
    "tabSize": 2,
    "tokens": [
      { "class": "keyword", "color": "accent6" },
      { "class": "name",    "color": "accent1" },
      { "class": "attr",    "color": "accent2" },
      { "class": "string",  "color": "accent3" },
      { "class": "number",  "color": "accent4" },
      { "class": "comment", "color": "muted" }
    ]
  }
}
```

### `layouts/`

`canvas.css`

```css

```

`code.css`

```css

```

`code.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto auto;
grid-template-areas: "title" "code" "caption";
align-content: start;
```

`content.css`

```css

```

`content.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto 1fr;
grid-template-areas: "title" "body";
align-content: start;
```

`end.css`

```css
justify-content: end;
```

`end.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: 1fr auto auto;
grid-template-areas: "." "title" "contact";
```

`image-full.css`

```css
padding: 0;
```

`image-full.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: 1fr auto;
grid-template-areas: "media" "caption";
padding: 0;
```

`image-text.css`

```css
display: grid;
grid-template-columns: 1.2fr 1fr;
align-content: start;
```

`image-text.slots.css`

```css
display: grid;
grid-template-columns: 1.2fr 1fr;
grid-template-rows: auto 1fr;
grid-template-areas: "title title" "media body";
align-content: start;
```

`quote.css`

```css
justify-content: center;
```

`quote.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto;
grid-template-areas: "quote" "attribution";
align-content: center;
```

`section-divider.css`

```css
justify-content: end;
```

`section-divider.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: 1fr auto auto;
grid-template-areas: "." "title" "subtitle";
```

`stat.css`

```css
justify-content: center;
```

`stat.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto auto auto;
grid-template-areas: "title" "value" "label" "body";
align-content: center;
justify-items: center;
```

`title.css`

```css
justify-content: end;
```

`title.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: 1fr auto auto auto auto auto;
grid-template-areas: "." "kicker" "title" "subtitle" "author" "date";
```

`two-column.css`

```css
display: grid;
grid-template-columns: 1.4fr 1fr;
align-content: start;
```

`two-column.slots.css`

```css
display: grid;
grid-template-columns: 1.4fr 1fr;
grid-template-rows: auto 1fr;
grid-template-areas: "title title" "left right";
align-content: start;
```

## `ink`

### `theme.json`

```json
{
  "name": "ink",
  "extends": "",
  "layouts": [
    "title", "section-divider", "content", "two-column",
    "image-full", "image-text", "quote", "code", "stat", "canvas", "end"
  ],
  "chrome": [
    {
      "kind": "footer",
      "align": "start",
      "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"]
    },
    {
      "kind": "pageNumber",
      "align": "end",
      "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"]
    }
  ]
}
```

### `tokens.json`

```json
{
  "color": [
    { "name": "fg",      "rgb": "F2F4F7", "slot": "dk1",     "ref": "tx1" },
    { "name": "bg",      "rgb": "15171B", "slot": "lt1",     "ref": "bg1" },
    { "name": "muted",   "rgb": "9CA3AF", "slot": "dk2",     "ref": "tx2" },
    { "name": "surface", "rgb": "21252B", "slot": "lt2",     "ref": "bg2" },
    { "name": "accent1", "rgb": "3B96FF", "slot": "accent1", "ref": "accent1" },
    { "name": "accent2", "rgb": "00A987", "slot": "accent2", "ref": "accent2" },
    { "name": "accent3", "rgb": "00AD2B", "slot": "accent3", "ref": "accent3" },
    { "name": "accent4", "rgb": "CB8700", "slot": "accent4", "ref": "accent4" },
    { "name": "accent5", "rgb": "FF5E4F", "slot": "accent5", "ref": "accent5" },
    { "name": "accent6", "rgb": "D75EFF", "slot": "accent6", "ref": "accent6" }
  ],

  "font": [
    { "name": "heading", "latin": "Noto Sans",      "ea": "Noto Sans CJK KR",      "generic": "sans-serif", "scheme": "major" },
    { "name": "body",    "latin": "Noto Sans",      "ea": "Noto Sans CJK KR",      "generic": "sans-serif", "scheme": "minor" },
    { "name": "mono",    "latin": "Noto Sans Mono", "ea": "Noto Sans Mono CJK KR", "generic": "monospace" }
  ],

  "space": [
    { "name": "xs",  "px": 4 },
    { "name": "sm",  "px": 10 },
    { "name": "md",  "px": 20 },
    { "name": "lg",  "px": 32 },
    { "name": "xl",  "px": 48 },
    { "name": "2xl", "px": 72 }
  ],

  "radius": [
    { "name": "sm", "px": 8 },
    { "name": "md", "px": 16 },
    { "name": "lg", "px": 28 }
  ],

  "shadow": [
    { "name": "soft", "dist": 6, "dir": 90, "blur": 24, "color": "bg", "alpha": 0.55 }
  ],

  "motion": { "duration": 600, "stagger": 90, "delay": 150 }
}
```

### `components.json`

```json
{
  "text": {
    "roles": [
      { "name": "title",   "font": "heading", "size": 52, "weight": 600, "lineHeight": 1.2, "color": "fg",    "align": "start" },
      { "name": "heading", "font": "heading", "size": 34, "weight": 600, "lineHeight": 1.3, "color": "fg",    "align": "start" },
      { "name": "body",    "font": "body",    "size": 24, "weight": 400, "lineHeight": 1.5, "color": "fg",    "align": "start" },
      { "name": "caption", "font": "body",    "size": 18, "weight": 400, "lineHeight": 1.4, "color": "muted", "align": "start" },
      { "name": "code",    "font": "mono",    "size": 20, "weight": 400, "lineHeight": 1.5, "color": "fg",    "align": "start" }
    ],
    "componentDefaults": [
      { "tag": "h3",         "size": 28 },
      { "tag": "blockquote", "color": "muted" },
      { "layout": "quote", "slot": "quote", "size": 40, "color": "fg" },
      { "layout": "stat",  "slot": "value", "size": 88, "weight": 700, "color": "accent1" },
      { "layout": "stat",  "slot": "label", "size": 28, "weight": 700 }
    ],
    "mark": { "highlight": "shade(accent4, 55%)" },
    "variants": [
      { "name": "kicker", "size": 18, "weight": 700, "color": "accent1", "tracking": 0.1, "case": "upper" },
      { "name": "lead",   "size": 32 }
    ]
  },

  "shape": {
    "default": { "fill": "surface", "border": "none", "radius": "md" },
    "variants": [
      { "name": "card",          "fill": "surface", "radius": "lg", "shadow": "soft" },
      { "name": "card-header",   "fill": "accent1", "radius": "sm" },
      { "name": "card-body",     "fill": "surface" },
      { "name": "card-backdrop", "fill": "none",    "border": "3 solid accent1", "radius": "lg" },
      { "name": "stat-card",     "fill": "surface", "radius": "lg", "shadow": "soft" },
      { "name": "badge",         "fill": "none",    "border": "1 solid accent1", "radius": "lg" }
    ]
  },

  "connector": {
    "default": { "line": "2 solid muted", "start": "none", "end": "arrow" },
    "variants": [
      { "name": "dashed", "line": "2 dashed muted" },
      { "name": "accent", "line": "2 solid accent1" },
      { "name": "plain",  "end": "none" }
    ]
  },
  "image": {
    "default": {},
    "variants": [
      { "name": "framed", "radius": "md", "shadow": "soft" }
    ]
  },

  "table": {
    "cell":   { "border": "1 solid surface" },
    "header": { "fill": "accent1", "color": "bg", "weight": 700 },
    "stripe": "surface"
  },

  "code": {
    "tabSize": 2,
    "tokens": [
      { "class": "keyword", "color": "accent6" },
      { "class": "name",    "color": "accent1" },
      { "class": "attr",    "color": "accent2" },
      { "class": "string",  "color": "accent3" },
      { "class": "number",  "color": "accent4" },
      { "class": "comment", "color": "muted" }
    ]
  }
}
```

### `layouts/`

`canvas.css`

```css

```

`code.css`

```css

```

`code.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto auto;
grid-template-areas: "title" "code" "caption";
align-content: start;
```

`content.css`

```css

```

`content.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto 1fr;
grid-template-areas: "title" "body";
align-content: start;
```

`end.css`

```css
justify-content: center;
```

`end.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto;
grid-template-areas: "title" "contact";
align-content: center;
```

`image-full.css`

```css
padding: 0;
```

`image-full.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: 1fr auto;
grid-template-areas: "media" "caption";
padding: 0;
```

`image-text.css`

```css
display: grid;
grid-template-columns: 1fr 1.2fr;
align-content: center;
```

`image-text.slots.css`

```css
display: grid;
grid-template-columns: 1fr 1.2fr;
grid-template-rows: auto 1fr;
grid-template-areas: "title title" "media body";
align-content: start;
align-items: center;
```

`quote.css`

```css
justify-content: center;
```

`quote.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto;
grid-template-areas: "quote" "attribution";
align-content: center;
```

`section-divider.css`

```css
justify-content: center;
```

`section-divider.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto;
grid-template-areas: "title" "subtitle";
align-content: center;
justify-items: center;
```

`stat.css`

```css
justify-content: center;
```

`stat.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto auto auto;
grid-template-areas: "title" "value" "label" "body";
align-content: center;
justify-items: center;
```

`title.css`

```css
justify-content: center;
```

`title.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: 1fr auto auto auto var(--balpyo-space-2xl) auto auto 1fr;
grid-template-areas: "." "kicker" "title" "subtitle" "." "author" "date" ".";
```

`two-column.css`

```css
display: grid;
grid-template-columns: 1fr 1fr;
align-content: start;
```

`two-column.slots.css`

```css
display: grid;
grid-template-columns: 1fr 1fr;
grid-template-rows: auto 1fr;
grid-template-areas: "title title" "left right";
align-content: start;
```

## `minimal`

### `theme.json`

```json
{
  "name": "minimal",
  "extends": "",
  "layouts": [
    "title", "section-divider", "content", "two-column",
    "image-full", "image-text", "quote", "code", "stat", "canvas", "end"
  ],
  "chrome": [
    {
      "kind": "footer",
      "align": "start",
      "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"]
    },
    {
      "kind": "pageNumber",
      "align": "end",
      "role": "caption",
      "skipLayouts": ["title", "section-divider", "image-full", "end"]
    }
  ]
}
```

### `tokens.json`

```json
{
  "color": [
    { "name": "fg",      "rgb": "1A1A1A", "slot": "dk1",     "ref": "tx1" },
    { "name": "bg",      "rgb": "FFFFFF", "slot": "lt1",     "ref": "bg1" },
    { "name": "muted",   "rgb": "6B7280", "slot": "dk2",     "ref": "tx2" },
    { "name": "surface", "rgb": "F3F4F6", "slot": "lt2",     "ref": "bg2" },
    { "name": "accent1", "rgb": "0F65F9", "slot": "accent1", "ref": "accent1" },
    { "name": "accent2", "rgb": "0E8174", "slot": "accent2", "ref": "accent2" },
    { "name": "accent3", "rgb": "15803D", "slot": "accent3", "ref": "accent3" },
    { "name": "accent4", "rgb": "B45309", "slot": "accent4", "ref": "accent4" },
    { "name": "accent5", "rgb": "DC2626", "slot": "accent5", "ref": "accent5" },
    { "name": "accent6", "rgb": "9333EA", "slot": "accent6", "ref": "accent6" }
  ],

  "font": [
    { "name": "heading", "latin": "Noto Sans",      "ea": "Noto Sans CJK KR", "generic": "sans-serif", "scheme": "major" },
    { "name": "body",    "latin": "Noto Sans",      "ea": "Noto Sans CJK KR", "generic": "sans-serif", "scheme": "minor" },
    { "name": "mono",    "latin": "Noto Sans Mono", "ea": "Noto Sans Mono CJK KR", "generic": "monospace" }
  ],

  "space": [
    { "name": "xs",  "px": 6 },
    { "name": "sm",  "px": 12 },
    { "name": "md",  "px": 24 },
    { "name": "lg",  "px": 36 },
    { "name": "xl",  "px": 48 },
    { "name": "2xl", "px": 72 }
  ],

  "radius": [
    { "name": "sm", "px": 6 },
    { "name": "md", "px": 12 },
    { "name": "lg", "px": 24 }
  ],

  "shadow": [
    { "name": "soft", "dist": 4, "dir": 90, "blur": 16, "color": "fg", "alpha": 0.15 }
  ],
  "motion": { "duration": 600, "stagger": 90, "delay": 150 }
}
```

### `components.json`

```json
{
  "text": {
    "roles": [
      { "name": "title",   "font": "heading", "size": 56, "weight": 700, "lineHeight": 1.2, "color": "fg",    "align": "start" },
      { "name": "heading", "font": "heading", "size": 36, "weight": 700, "lineHeight": 1.3, "color": "fg",    "align": "start" },
      { "name": "body",    "font": "body",    "size": 24, "weight": 400, "lineHeight": 1.5, "color": "fg",    "align": "start" },
      { "name": "caption", "font": "body",    "size": 18, "weight": 400, "lineHeight": 1.4, "color": "muted", "align": "start" },
      { "name": "code",    "font": "mono",    "size": 20, "weight": 400, "lineHeight": 1.5, "color": "fg",    "align": "start" }
    ],
    "componentDefaults": [
      { "tag": "h3",         "size": 28 },
      { "tag": "blockquote", "color": "muted" },
      { "layout": "quote", "slot": "quote", "size": 40, "color": "fg" },
      { "layout": "stat",  "slot": "value", "size": 96, "weight": 700, "color": "accent1" },
      { "layout": "stat",  "slot": "label", "size": 28, "weight": 700 }
    ],
    "mark": { "highlight": "tint(accent4, 75%)" },
    "variants": [
      { "name": "kicker", "size": 18, "weight": 700, "color": "accent1", "tracking": 0.1, "case": "upper" },
      { "name": "lead",   "size": 32 }
    ]
  },

  "shape": {
    "default": { "fill": "surface", "border": "1 solid muted", "radius": "md" },
    "variants": [
      { "name": "card",          "fill": "bg",   "radius": "lg" },
      { "name": "card-header",   "border": "none", "radius": "sm" },
      { "name": "card-body",     "fill": "bg" },
      { "name": "card-backdrop", "fill": "none", "border": "3 solid accent1", "radius": "lg" },
      { "name": "stat-card",     "border": "none", "radius": "lg", "shadow": "soft" },
      { "name": "badge",         "border": "1 solid accent1", "radius": "lg" }
    ]
  },

  "connector": {
    "default": { "line": "2 solid muted", "start": "none", "end": "arrow" },
    "variants": [
      { "name": "dashed", "line": "2 dashed muted" },
      { "name": "accent", "line": "2 solid accent1" },
      { "name": "plain",  "end": "none" }
    ]
  },
  "image": {
    "default": {},
    "variants": [
      { "name": "framed", "border": "1 solid muted", "radius": "md", "shadow": "soft" }
    ]
  },

  "table": {
    "cell":   { "border": "1 solid muted" },
    "header": { "fill": "accent1", "color": "bg", "weight": 700 },
    "stripe": "surface"
  },

  "code": {
    "tabSize": 2,
    "tokens": [
      { "class": "keyword", "color": "accent6" },
      { "class": "name",    "color": "accent1" },
      { "class": "attr",    "color": "accent2" },
      { "class": "string",  "color": "accent3" },
      { "class": "number",  "color": "accent4" },
      { "class": "comment", "color": "muted" }
    ]
  }
}
```

### `layouts/`

`canvas.css`

```css

```

`code.css`

```css

```

`code.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto auto;
grid-template-areas: "title" "code" "caption";
align-content: start;
```

`content.css`

```css

```

`content.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto 1fr;
grid-template-areas: "title" "body";
align-content: start;
```

`end.css`

```css
justify-content: center;
```

`end.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto;
grid-template-areas: "title" "contact";
align-content: center;
justify-items: center;
```

`image-full.css`

```css
padding: 0;
```

`image-full.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: 1fr auto;
grid-template-areas: "media" "caption";
padding: 0;
```

`image-text.css`

```css
display: grid;
grid-template-columns: 1fr 1fr;
align-content: center;
```

`image-text.slots.css`

```css
display: grid;
grid-template-columns: 1fr 1fr;
grid-template-rows: auto 1fr;
grid-template-areas: "title title" "media body";
align-content: start;
align-items: center;
```

`quote.css`

```css
justify-content: center;
```

`quote.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto;
grid-template-areas: "quote" "attribution";
align-content: center;
```

`section-divider.css`

```css
justify-content: center;
```

`section-divider.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto;
grid-template-areas: "title" "subtitle";
align-content: center;
```

`stat.css`

```css
justify-content: center;
```

`stat.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto auto auto;
grid-template-areas: "title" "value" "label" "body";
align-content: center;
justify-items: center;
```

`title.css`

```css
justify-content: center;
```

`title.slots.css`

```css
display: grid;
grid-template-columns: 1fr;
grid-template-rows: auto auto auto var(--balpyo-space-2xl) auto auto;
grid-template-areas: "kicker" "title" "subtitle" "." "author" "date";
justify-items: center;
align-content: center;
```

`two-column.css`

```css
display: grid;
grid-template-columns: 1fr 1fr;
align-content: start;
```

`two-column.slots.css`

```css
display: grid;
grid-template-columns: 1fr 1fr;
grid-template-rows: auto 1fr;
grid-template-areas: "title title" "left right";
align-content: start;
```
