# Cypherpunk Design Tokens (Reference)

Machine-readable token values referenced by SKILL.md. Treat hex values as the source of truth for implementation; SKILL.md is the narrative/usage layer on top of these.

> Note: hex values below were derived visually from the reference screenshot. Confirm against the original Figma/source file if one exists, and update this file in place if any value is corrected.

## Color Tokens

```json
{
  "color": {
    "lime": "#D7FF4E",
    "ink": "#131310",
    "ink-solid": "#0E0E0C",
    "beige": "#ECE6D6",
    "signal-green": "#2F6B3D",
    "signal-red": "#D64545",
    "signal-gold": "#C9A227"
  },
  "semantic": {
    "bg-page": "{color.lime}",
    "bg-surface": "{color.beige}",
    "bg-solid": "{color.ink-solid}",
    "text-primary": "{color.ink}",
    "text-inverse": "{color.beige}",
    "border-default": "{color.ink}",
    "border-width-default": "2px",
    "data-positive": "{color.signal-green}",
    "data-negative": "{color.signal-red}",
    "data-warning": "{color.signal-gold}",
    "focus-ring": "{color.ink}",
    "focus-ring-offset": "2px"
  }
}
```

### Opacity variants (Ink)
- `ink/100` — primary text, borders
- `ink/60` — secondary/caption text
- `ink/35` — disabled state text/icons
- `ink/20` — dashed gridlines, hairline dividers
- `ink/10` — subtle row-hover overlay on Beige

## Typography Tokens

```json
{
  "font": {
    "display": "JetBrains Mono, monospace (or a true pixel font, e.g. 'Pixel Operator', 'Press Start 2P' for heavier retro emphasis)",
    "body": "Inter, sans-serif (or Helvetica Now)",
    "mono": "JetBrains Mono, monospace"
  },
  "type-scale": {
    "xs": "12px",
    "sm": "14px",
    "base": "16px",
    "lg": "20px",
    "xl": "24px",
    "2xl": "32px",
    "3xl": "40px"
  },
  "weights": {
    "display": [500, 700],
    "body": [400, 500, 600],
    "mono": [400, 500]
  },
  "line-height": {
    "display": 1.1,
    "body": 1.4,
    "mono": 1.3
  }
}
```

## Spacing Scale

```json
{ "spacing": [4, 8, 12, 16, 24, 32, 48] }
```

## Shape / Border Tokens

```json
{
  "radius": {
    "default": "2px",
    "badge-square": "2px",
    "max-allowed": "2px"
  },
  "border": {
    "width-default": "2px",
    "width-hairline": "1px",
    "style": "solid",
    "color": "{color.ink}"
  },
  "elevation": {
    "shadow": "none",
    "note": "No box-shadow tokens exist in this system by design. Depth = border + background contrast only."
  }
}
```

## Component Specs

### Stat/KPI card
- Padding: 24px
- Border: 2px solid Ink
- Background: Beige
- Radius: 2px
- Layout: label (xs, ink/60) top-left · trend chip (xs, bordered, semantic color) top-right · value (2xl/3xl, body semibold or display) · caption (sm, ink/60) · sparkline (bottom, ~40px height, semantic color stroke, no fill or subtle gradient fill under the line only)

### Trend chip
- Padding: 4px 8px
- Border: 1.5–2px solid semantic color (green/red)
- Background: Beige
- Text: xs, mono or body medium, same semantic color as border
- Icon: small arrow (up/down), same semantic color

### Buttons
| Variant | Background | Border | Text | Radius |
|---|---|---|---|---|
| Primary | ink-solid | none | beige | 2px |
| Secondary | beige | 2px ink | ink | 2px |
| Tertiary/ghost | transparent | none | ink (underline on hover) | n/a |

### Nav badge (count chip)
- Size: ~20x20px minimum, square, radius 2px
- Background: ink-solid
- Text: white or beige, xs, mono or bold body

### Chart card
- Padding: 24px
- Border: 2px solid ink
- Background: beige
- Gridlines: dashed, ink/20, 1px
- Primary series stroke: ink, 2px
- Comparison series stroke: ink/40, 1.5px, dashed optional
- Legend swatches: 8x8px squares (not circles), matching series color, radius 0–1px

## Do/Don't Quick Reference

| Do | Don't |
|---|---|
| Lime page bg everywhere | A second/alternate page bg color |
| 2px Ink border on all raised surfaces | 1px hairline borders on cards/buttons |
| Sharp 0–2px corners | Pill/fully-rounded shapes |
| Signal colors only on real data | Signal colors as decorative accents |
| Ink as the default chart data color | Brand accent as default chart data color |
| Box-shadow: none | Any drop shadow or glow |
