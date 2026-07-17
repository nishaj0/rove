---
name: cypherpunk-design
description: Cypherpunk's visual design system — a stark retro-tech, two-tone interface built on an electric-lime surface, near-black ink text, and beige raised components with bold 2px borders. Use this skill whenever building, styling, or reviewing any UI in this codebase — dashboards, consoles, data tables, cards, forms, buttons, charts, nav/sidebars, or any component. Always apply this skill instead of generic defaults (no soft gradients, no pill rounding, no glossy card depth, no alternate section backgrounds) when the user mentions Cypherpunk, security consoles, audit/secrets dashboards, developer tooling UI, or any screen in this project.
license: MIT
metadata:
  author: nishaj
---

# Cypherpunk Design System Skill

## Mission

You are the design-system guideline author for **Cypherpunk**, a stark retro-tech design language for security dashboards, developer tools, crypto/decentralized-web products, and experimental SaaS UI. Create practical, implementation-ready guidance that can be directly used by engineers building this UI — and apply it directly whenever generating Cypherpunk components or screens.

## Brand

Cypherpunk is deliberately loud, technical, and subversive — a two-tone visual language that refuses the soft-SaaS look. Every section sits on an **electric-lime** surface. All reading text is **near-black ink**. Interactive/raised surfaces (cards, inputs, secondary buttons) are **beige**, always with a bold ink border. There is no third background color and no gradient — the starkness is the point.

**Core design principle — "two tones, hard edges":** The palette is intentionally restrictive (lime page, beige surface, ink text/borders, plus small hits of semantic color for data only). Depth comes from borders and contrast, never shadows or gradients. Corners stay crisp. Nothing about this system should read as soft, glossy, or app-store generic.

## Style Foundations

- **Visual style:** stark retro-tech, two-tone, high-contrast — not soft-SaaS, not glassy, not pastel
- **Typography scale:** 12/14/16/20/24/32/40 | Fonts: display=pixel/monospace display face (e.g. JetBrains Mono or a true pixel font), body=a clean grotesk sans (e.g. Inter or Helvetica Now), data/numeric=JetBrains Mono | weights: display 500/700, body 400/500/600, mono 400/500
- **Color palette:** Lime, Ink, Beige, plus restrained semantic accents (Signal Green, Signal Red, Signal Gold) — see Colors below
- **Spacing scale:** 4/8/12/16/24/32/48
- **Corners:** 0–2px radius maximum everywhere. No pill shapes, no fully-rounded buttons or badges except small square-ish count badges.
- **Borders:** 2px solid Ink is the default border weight for cards, inputs, and buttons. This is a defining trait — thin 1px hairlines read as "wrong" in this system.
- **Elevation:** none. No drop shadows, no soft glows. Depth is communicated entirely through the lime/beige contrast and the 2px ink border.
- **Mode:** single-mode by design (lime + ink + beige). This system does not have a conventional "light/dark" toggle — the lime surface *is* the brand; treat it as fixed rather than themeable.

## Colors

- **Lime (`#D7FF4E`):** Primary page/section background. Every screen's base surface. Loud, electric, unapologetic — never dial it back into a muted olive or pastel green.
- **Ink (`#131310`):** Primary text color and the color of every border. Near-black, not pure `#000000` — slightly warm so it doesn't look like a default browser black.
- **Beige (`#ECE6D6`):** Raised surface color for cards, inputs, dropdowns, and secondary buttons. This is the only other background color in the system — do not introduce a third neutral.
- **Ink-solid (`#0E0E0C`):** Full-black fill used for primary buttons, active nav pills, and avatar/badge chips. Text on this fill is Beige or white.
- **Signal Green (`#2F6B3D`):** Positive/upward data only — sparkline upticks, "clean" status. Not a UI accent; reserve for data meaning "good."
- **Signal Red (`#D64545`):** Negative/urgent data only — downward trends, due/overdue states, critical severity. Reserve for data meaning "attention needed."
- **Signal Gold (`#C9A227`):** Warning-tier data only — the middle severity between green and red (e.g. "warning" in a stacked chart legend).

**Derived semantic tokens** (map to the above, don't introduce new hexes):
- `bg-page` = Lime
- `bg-surface` = Beige
- `text-primary` / `border-default` = Ink
- `bg-solid` = Ink-solid (primary buttons, active states, badges)
- `data-positive` = Signal Green
- `data-negative` = Signal Red
- `data-warning` = Signal Gold

## Component-Level Rules

### Page shell / navigation
- Sidebar sits directly on the Lime background — no separate sidebar panel color, no shadow separating it from content.
- Active nav item: Beige pill-free rectangle (sharp corners) with Ink text, or an Ink-solid rectangle with Lime/Beige text for the single "current" item — pick one convention and hold it everywhere.
- Section count badges (e.g. a "3" next to a nav label): small square Ink-solid chip, white/Lime text, sharp corners — never a fully round pill.
- A single bold horizontal Ink rule separates primary nav from secondary/resource links at the bottom of a sidebar.

### Top bar
- Search input: Beige surface, 2px Ink border, sharp corners, placeholder text in muted Ink (not a lighter gray — this system doesn't do tinted grays, it does opacity-reduced Ink).
- Primary action button (e.g. "New secret"): Ink-solid fill, Beige/white text, sharp corners, no icon-only ambiguity — always paired with a label.
- Secondary buttons (e.g. "Export audit"): Beige fill, 2px Ink border, Ink text.
- Avatar / account chip: small square Ink-solid fill with initials, sharp corners, top-right corner of the shell.

### Stat / KPI cards
- Beige surface, 2px Ink border, sharp corners, no shadow.
- Anatomy: small-caps label (top-left) → inline trend chip (top-right, bordered, containing an arrow + delta) → large bold numeric value (display or heavy body weight) → supporting caption line → inline sparkline anchored to the bottom of the card.
- Sparkline color follows the semantic data tokens: Signal Green for upward/positive trend lines, Signal Red for downward/urgent trend lines. Do not use Ink for a trend line — Ink is reserved for structure and text, not chart data.
- Trend chip border and text use the same semantic color as the sparkline it summarizes.

### Charts (line + bar)
- Chart canvas sits inside a Beige card with a 2px Ink border, same as stat cards — never a borderless floating chart.
- Gridlines: thin dashed Ink at low opacity — never colored gridlines.
- Primary data series: solid Ink line (or Ink-solid bars) — this system's "default" data color is Ink itself, not a brand accent. Reserve Signal Green/Red/Gold strictly for comparison series, severity segments, or explicit good/bad meaning (e.g. a "7-day baseline" comparison line renders in a muted Ink tint, not a competing hue).
- Stacked bar segments use the three signal colors directly as a severity legend (e.g. Critical=Signal Red, Warning=Signal Gold, Info=Ink-solid) — always show a legend with color swatches, never rely on position alone.
- Axis labels and legend text: small, mono or body font, Ink at reduced opacity.
- Chart footer actions ("Export CSV", "Audit report →"): treat as tertiary — Ink text, underline or arrow affordance only, no button chrome.

### Buttons
- Primary: Ink-solid background, Beige or white text, sharp corners, bold body weight.
- Secondary: Beige background, 2px Ink border, Ink text.
- Tertiary/ghost: no fill, no border, Ink text with underline or arrow affordance on hover/focus.
- States required: default, hover (slightly darken/lighten fill — never introduce a new hue), focus-visible (a visible 2px offset outline — do not rely on the existing border alone to indicate focus), active, disabled (Ink at ~35% opacity, no pointer events).

### Inputs & dropdowns
- Beige surface, 2px Ink border, sharp corners, Ink text.
- Dropdown chevron and calendar/filter icons: Ink, line-art style, no filled icon glyphs.
- Focus state: border stays Ink but gains a visible offset focus ring — never swap border color to a foreign accent hue (no blue focus rings in this system).

### Tables & lists
- No zebra striping with a third color — use a thin Ink hairline divider between rows instead, same convention as borders elsewhere.
- Row hover: subtle Ink-tinted overlay on the Beige surface, not a color swap.

## Typography

- **Display (pixel/mono display face):** Section titles like screen headers ("Console," "Policy events by environment"). Used with intent — this is the loudest, most "branded" typographic moment on a screen. Keep body copy far away from this face; it should never be used for paragraphs or dense UI labels.
- **Body (grotesk sans):** All UI labels, nav items, table content, form labels, button text, captions. Default weight 400, medium (500) for emphasis, semibold (600) only for the single most important line on a card (e.g. the large stat number can also live here instead of the display face, if a screen wants a quieter tone).
- **Mono (JetBrains Mono or equivalent):** Chart axis labels, timestamps, numeric data, legend values. Reinforces the "this is real measured data" feeling — never use the display pixel face for data, and never use it for numbers that update frequently (it reads noisy at small sizes).

## Accessibility

- WCAG 2.2 AA contrast minimum for all text/background pairs — Ink-on-Lime and Ink-on-Beige should both be checked explicitly; Signal Gold text/borders on Lime are the pairing most likely to fail contrast and need verification.
- Every interactive element must have a visible focus-visible state — a 2px offset outline is required since this system's default border is already thick and easy to confuse with a focus indicator if not handled deliberately.
- Never convey severity (Critical/Warning/Info) through color alone — always pair Signal Red/Gold/Ink-solid with a text label or icon, since this is a design system built around data-critical dashboards.
- Respect `prefers-reduced-motion` for any sparkline draw-in or chart transition animation.

## Writing Tone

Blunt, technical, no filler. Short labels over descriptive sentences ("Rotations due," not "Number of credentials pending rotation"). Status and caption copy states facts plainly ("no drift detected," "within window") rather than reassuring or apologetic tone. Numbers and units are always explicit (e.g. "48h," "2.4k," "+18%") — never spelled out in prose where a compact numeric label would do.

## Rules: Do
- Keep every screen's base background Lime — never introduce a second page background color
- Give every card, input, and secondary button a full 2px Ink border with sharp (0–2px) corners
- Reserve Signal Green/Red/Gold strictly for data meaning (trend direction, severity) — never as decorative UI accents
- Use the display pixel/mono face only for section headers and major titles, never for body or data

## Rules: Don't
- Don't add soft gradients, glassy blur, or drop shadows anywhere — depth comes from borders only
- Don't use pill/fully-rounded shapes for buttons, badges, or nav items — sharp corners only, with the narrow exception of small square count-badges
- Don't introduce a third page background color or a tinted gray — neutrals in this system are Lime, Beige, and Ink (with opacity variants of Ink), nothing else
- Don't let a chart's primary data series use a semantic accent color — Ink is the default data color; accents are reserved for explicit good/bad/severity meaning

## Expected Behavior

- Follow the "two tones, hard edges" principle first, then component consistency, then novelty.
- When uncertain whether something should be Lime or Beige, ask: is this the page/section itself (Lime) or something raised on top of it that a user interacts with (Beige)? That test resolves nearly every case.
- Provide concrete token references (not "a warning color," but "Signal Gold, `#C9A227`") in every recommendation.
- Keep guidance opinionated and implementation-focused — state hex values, font names, border widths, and corner radii explicitly rather than describing them abstractly.

## QA Checklist

- [ ] Is the page background Lime with no second background hue introduced anywhere?
- [ ] Does every card, input, and secondary button have a full 2px Ink border and sharp corners?
- [ ] Is any shadow, gradient, or glassy blur present anywhere? (There shouldn't be.)
- [ ] Do Signal Green/Red/Gold only appear attached to real data meaning (trend, severity) rather than decoratively?
- [ ] Is the display pixel/mono face reserved for section headers/titles only, never body or data?
- [ ] Does every interactive element have a visible focus-visible state distinct from its resting border?
- [ ] Is severity/status ever conveyed by color alone without a text label or icon backup?

For full token values (hex codes, exact type scale, spacing scale, and component measurements) in machine-readable form, see `DESIGN.md` in this skill's directory.
