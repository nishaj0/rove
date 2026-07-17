# Nostalgia Design Tokens (Reference)

Machine-readable token values referenced by SKILL.md. Light mode is the default; dark-mode tokens exist in parallel and should only be applied when explicitly requested.

> Note: light-mode hex values come from the stated design brief; dark-mode values were derived to satisfy "deep brown-black surfaces, warm cream text, brighter orange accents" since exact dark hexes weren't specified. Confirm against a source file if one exists and correct in place if needed.

## Color Tokens

```json
{
  "color": {
    "light": {
      "surface": "#FFEEDD",
      "surface-secondary": "#F6E6D6",
      "brand": "#FF631A",
      "brand-strong": "#D94A00",
      "heading": "#381C00",
      "body": "#4A3826",
      "border": "#C7AE8E"
    },
    "dark": {
      "surface": "#1E140C",
      "surface-secondary": "#2A1C12",
      "brand": "#FF7A3D",
      "brand-strong": "#FF9556",
      "heading": "#FBEBD9",
      "body": "#D9C7B0",
      "border": "#4A3826"
    },
    "chart-support": {
      "tan": "#C7AE8E",
      "rust": "#B5451A",
      "olive-green": "#4E6B3A"
    },
    "window-control-dots": {
      "red": "#E5484D",
      "amber": "#F5A623",
      "green": "#2FA84F",
      "note": "Decorative chrome only, fixed left-to-right order red/amber/green. Never repurpose as real status indicators."
    }
  },
  "semantic": {
    "bg-desktop": "{color.<mode>.surface}",
    "bg-window": "{color.<mode>.surface-secondary}",
    "text-heading": "{color.<mode>.heading}",
    "text-body": "{color.<mode>.body}",
    "border-default": "{color.<mode>.border}",
    "accent": "{color.<mode>.brand}",
    "accent-strong": "{color.<mode>.brand-strong}",
    "focus-ring": "{color.<mode>.brand-strong}",
    "focus-ring-offset": "2px"
  }
}
```

### Opacity variants (Body, per mode)
- `body/100` — primary body text
- `body/60` — captions, metadata, section labels
- `body/35` — disabled state text/icons
- `border/50` — dashed chart gridlines

## Typography Tokens

```json
{
  "font": {
    "display": "a literary/editorial serif (e.g. 'Source Serif 4', 'Lora', or similar) — major headings only",
    "body": "a clean geometric sans (e.g. 'Inter', 'Sora', or similar) — UI copy, labels, non-machine table content",
    "mono": "a monospace face (e.g. 'JetBrains Mono', 'IBM Plex Mono') — code, metadata, filenames, timestamps, system captions"
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
    "display": 1.15,
    "body": 1.4,
    "mono": 1.3
  }
}
```

## Spacing Scale

```json
{ "spacing": [4, 8, 12, 16, 24, 32, 48] }
```

## Shape / Border / Shadow Tokens

```json
{
  "radius": {
    "default": "3px",
    "max-allowed": "3px"
  },
  "border": {
    "width-default": "1px",
    "style": "solid",
    "color": "{border-default}"
  },
  "shadow": {
    "window": "4px 4px 0px rgba(56, 28, 0, 0.25)",
    "note": "Solid, directional, warm-brown-tinted offset shadow — no blur radius, no gray. This is the 'physical window' cue. In dark mode, tint the shadow with the dark border color instead of the light one."
  }
}
```

## Component Specs

### Window / panel
- Title bar: padding 8px 12px, `bg-window`, bottom border 1px `border-default`, optional 3 control dots (8px circles, red/amber/green, 4px gaps) + label (body font, small, `text-heading`)
- Body: padding 24px, `bg-window`, border 1px `border-default` (wraps whole window including title bar), radius 3px
- Shadow: per shadow.window token above

### Stat/KPI window
- Title bar as above
- Value: 2xl–3xl, bold, `text-heading`, display or heavy body weight
- Caption: sm, mono, `text-body` at 60% opacity

### Buttons
| Variant | Background | Border | Text | Radius |
|---|---|---|---|---|
| Primary | brand-strong | 1px border-default | cream/inverse | 3px |
| Secondary | surface-secondary | 1px border-default | body/heading | 3px |

### Table
- Header row: body font, `text-body` at 60%, bottom border 1px `border-default`
- Filename/size/date-machine cells: mono font
- Status tag: inline-flex, padding 2px 8px, border 1px (color = status meaning), radius 3px, background transparent (shows `bg-window` through), text same color as border

### Chart window
- Line/area: stroke = brand (2px), fill = linear gradient brand→transparent (only permitted gradient use case)
- Bar: flat fill, brand (primary series) + tan (comparison series), 0px bar-top radius
- Donut: flat segments, brand + rust + tan + olive-green, plain square swatch legend (8x8px)
- Gridlines: dashed, 1px, border-default at 50% opacity

## Do/Don't Quick Reference

| Do | Don't |
|---|---|
| Every container = bordered window with title bar | Borderless flat cards |
| Light mode default; dark mode opt-in only | Silently defaulting to or mixing dark-mode tokens |
| Radius ≤3px everywhere | Pill/fully-rounded shapes |
| Warm gradient fill under chart lines only | Gradients/glossy blur anywhere else |
| Mono for data/metadata, serif for major headings only | Serif for body copy; mono for headings |
| Solid directional warm shadow | Soft blurry gray drop shadow |
