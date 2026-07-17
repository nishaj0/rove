---
name: nostalgia-design
description: Nostalgia's visual design system — a retro desktop-OS design language built on toasted-cream surfaces, espresso-brown ink, glowing orange accents, crisp taupe borders, and window-style chrome (title bars, control dots, folder metaphors). Light mode is the default; a CRT-terminal dark mode is available and should only be used when explicitly requested or specified. Use this skill whenever building, styling, or reviewing any UI in this codebase — dashboards, archives, media libraries, personal tools, cards, tables, charts, nav bars, or any component. Always apply this skill instead of generic defaults (no soft gradients, no pill rounding, no blur-heavy glossy depth, no borderless flat panels) when the user mentions Nostalgia, retro desktop UI, archive/library dashboards, personal/developer utility tools, or any screen in this project.
license: MIT
metadata:
  author: nishaj
---

# Nostalgia Design System Skill

## Mission

You are the design-system guideline author for **Nostalgia**, a retro desktop-OS design language for archives, personal dashboards, developer utilities, note apps, portfolios, and playful SaaS tools. Create practical, implementation-ready guidance that can be directly used by engineers building this UI — and apply it directly whenever generating Nostalgia components or screens.

## Brand

Nostalgia turns a generated interface into a vintage operating-system workspace: toasted-cream desktops, espresso-brown ink, glowing orange highlights, and window-like panels with visible chrome. It's playful and familiar, but precise — every surface has a border, every panel has a title bar, and every interaction should feel like pressing a tangible desktop control rather than tapping a flat modern button.

**Core design principle — "every panel is a window":** Content never floats as a borderless flat card. It sits inside a bordered "window" with a title-bar strip (optionally carrying decorative control dots), the same way a retro desktop app would present a document or utility. Depth comes from warm, directional drop shadows and crisp 1px borders — never soft blur or glassy gradients.

## Modes

Nostalgia has two modes and both must be supported in any implementation:

- **Light mode (default).** Toasted-cream desktop and warm brown ink. Use this unless the person building on this skill explicitly asks for dark mode or the product/user has specified dark as their preference.
- **Dark mode (opt-in).** Shifts to a deep brown-black, CRT-terminal atmosphere with warm cream text and brighter, hotter orange accents. Only apply this when specifically requested — never default to it, and never silently mix tokens from both modes on the same screen.

When in doubt about which mode to build, build light mode and mention that a dark variant exists and can be requested.

## Colors

### Light mode (default)

- **Surface (`#FFEEDD`):** The desktop/page background — the "wallpaper" layer that sits behind windows. Never put content copy directly on this layer without a window/panel border around it.
- **Secondary Surface (`#F6E6D6`):** Window bodies, panel backgrounds, table zebra rows, and any secondary/raised surface living inside a window.
- **Brand (`#FF631A`):** Primary orange accent — active nav state, links, primary chart series, active toggle/control state.
- **Brand Strong (`#D94A00`):** Stronger CTA and "pressed/active chrome" state — primary button backgrounds, active title-bar accents.
- **Heading (`#381C00`):** Espresso-brown, used for all headings and title-bar text.
- **Body (`#4A3826`):** Slightly lighter warm brown for body copy, labels, and general UI text.
- **Border (Taupe, `#C7AE8E`):** The crisp 1px hairline used on every window, panel, input, and divider in light mode.

### Dark mode (opt-in)

- **Surface (dark, `#1E140C`):** Deep brown-black desktop background.
- **Secondary Surface (dark, `#2A1C12`):** Window bodies and panels — one step lighter than the desktop so window chrome still reads as "raised."
- **Brand (dark, `#FF7A3D`):** Brighter, hotter orange accent — needs the lift in luminance to read clearly against near-black surfaces.
- **Brand Strong (dark, `#FF9556`):** Even hotter orange for pressed/active states and primary CTAs — this is the "glowing" moment the system is named for.
- **Heading (dark, `#FBEBD9`):** Warm cream, used for headings and title-bar text.
- **Body (dark, `#D9C7B0`):** Slightly muted warm cream for body copy and labels.
- **Border (dark, `#4A3826`):** Warm brown hairline — still crisp 1px, just shifted darker to sit correctly against the dark surfaces.

**Derived semantic tokens** (map to the mode-appropriate values above, don't introduce new hexes):
- `bg-desktop` = Surface
- `bg-window` = Secondary Surface
- `text-heading` = Heading
- `text-body` = Body
- `border-default` = Border (Taupe in light, warm brown in dark)
- `accent` = Brand
- `accent-strong` = Brand Strong

## Component-Level Rules

### Windows / panels (the core container)
- Every content container is a "window": a bordered rectangle (1px Border) with a distinct title-bar strip across the top, using `bg-window`/Secondary Surface for the body.
- Title bar carries the panel's label (e.g. "Activity.log," "Weekly playback hours") in Heading color, optionally preceded by three small decorative control dots (red, amber, green, in that fixed order) — these are chrome, not status indicators, and should not be reused to mean anything about the panel's real state.
- Radius tops out at 3px everywhere, including windows, buttons, and tags — never fully rounded, never sharp 0px either; a small consistent radius is part of the "tactile, not sterile" feel.
- Shadow: warm, directional, and physical — a soft-but-solid offset shadow (e.g. offset down-right, brown-tinted, no blur haze) that suggests a window sitting above the desktop. Never a diffuse modern blur shadow, never a colorless gray shadow.

### Navigation
- A fixed top menu bar and/or a left sidebar sit directly on the desktop Surface color, styled like a system nav rather than a floating app-shell panel — no card border around the whole sidebar itself, only around individual grouped sections if needed.
- Active nav item: Brand-colored text/icon (not a solid fill background) — reads like a "currently open" system menu item rather than a modern pill-highlight tab.
- Section labels ("NAVIGATION," "COLLECTIONS"): small caps, Body color at reduced opacity, mono or body font, used as system-style section dividers.
- Sidebar list items with trailing counts (e.g. a collection name with an item count on the right): plain text row, no badge chrome — the count itself is enough, styled in Body color at reduced opacity.
- App identity mark: a folder or window icon treated as a tiny "desktop icon," paired with a product name (Heading, serif) and a small version/mode caption underneath in mono (e.g. "v2.4.1 — Library mode").

### Stat / KPI cards
- Structured as a window: title-bar strip with control dots + label, window body below containing a large numeric value (Heading, bold, large size — serif or heavy sans) and a supporting caption line in mono or body font at reduced opacity (e.g. "+248 this week").
- No sparkline required inside the card itself — Nostalgia favors dedicated chart windows below/beside stat cards rather than cramming a chart into every stat tile, matching the reference layout's separation of "numbers" windows from "chart" windows.

### Charts
- Chart canvas lives inside its own window (title bar + bordered body), same chrome as every other panel.
- Primary series color is Brand (orange); a comparison/secondary series uses a muted taupe/tan tone — never a competing hue.
- Line/area charts: Brand-colored line with a soft warm gradient fill fading toward the window body color beneath it — this is the one place a gradient is allowed, since it reads as a warm "glow," not a glossy modern effect.
- Bar charts: flat-fill bars, Brand for the primary series and a muted taupe for the comparison series, sharp/crisp bar edges (no rounded bar tops).
- Donut/pie charts: flat-fill segments using Brand plus 2–3 supporting warm neutrals (rust, tan, olive/green used sparingly) with a plain swatch-and-label legend beneath — never a glossy 3D donut effect.
- Gridlines: thin dashed Border-color hairlines at reduced opacity.

### Tables
- Header row and body rows live inside a single window (one title bar, one bordered body) — not each row as its own card.
- Filenames, sizes, and other machine-legible values render in the mono font; everything else (Type, Collection, Date) renders in body font.
- Status values render as small bordered tags (3px radius, 1px border, text + border sharing a color) — never a solid filled pill. Vary the tag's color per status meaning (e.g. Brand orange for an in-progress state, a muted neutral for a settled/complete state) but keep every tag built from the same bordered-text shape.
- Row dividers: 1px Border hairlines; avoid heavy zebra-striping beyond the Secondary Surface tone already used for the window body.

### Buttons
- Primary: flat Brand Strong fill, Heading-inverse (cream) text, 1px Border, 3px radius. Active/pressed state visibly "presses" — shift to a slightly darker fill and/or nudge the warm shadow inward, mimicking a physical key press.
- Secondary: Secondary Surface fill, 1px Border, Body/Heading text.
- States required: default, hover (slight fill shift, no color-hue change), active/pressed (shadow tucks in, fill darkens — the "pressed key" moment), focus-visible (visible offset outline, since the existing border is thin and easy to miss as a focus cue), disabled (reduced opacity, no pointer events).

### Inputs
- Secondary Surface fill, 1px Border, 3px radius, Body-color text and placeholder (placeholder at reduced opacity, not a foreign gray).
- Icons inside inputs (search glyph, etc.): line-art style, Body color.

## Typography

- **Display (literary serif):** Page titles and major headings only (e.g. "Media library"). This is the most "editorial" moment on a screen — reserve it for the single most important heading per view.
- **Body (geometric sans):** All UI labels, nav items, descriptions, table content (except machine values), button text, captions.
- **Mono (monospace):** Code, metadata, system labels, timestamps, filenames, chart axis labels, and version/status captions (e.g. "v2.4.1 — Library mode," "Last sync 4 minutes ago"). This is what makes the system read as "machine output" layered under the editorial serif headings — use it generously for anything that resembles data.

## Accessibility

- WCAG 2.2 AA contrast minimum for all text/background pairs in both modes — check Body-on-Secondary-Surface and Brand-on-Surface pairings explicitly, since warm-on-warm palettes can quietly fail contrast even when they look legible.
- Every interactive element needs a visible focus-visible state distinct from its resting border — the default 1px border is too subtle to double as a focus indicator.
- Decorative window control dots must never be the sole way status is communicated — pair real status (processing, error, complete) with a text label or bordered tag, not just a colored dot.
- Respect `prefers-reduced-motion` for any chart draw-in, window-open, or pressed-button transition animation.
- Provide a dark-mode toggle affordance rather than relying on `prefers-color-scheme` alone if the product wants users to be able to override the OS setting — but default the initial state to light regardless.

## Writing Tone

Warm but plainspoken, like a well-worn instruction manual. Titles can be a little editorial ("Media library," "Activity.log" as a panel name), but body copy and captions stay factual and compact ("Last sync 4 minutes ago," "68% of 1.2 TB," "3 need review"). Numbers and units are always explicit and machine-legible in mono where they represent real data.

## Rules: Do
- Treat every content container as a bordered window with a title-bar strip — never a borderless flat card
- Default to light mode; only build dark mode when explicitly requested, and never mix tokens from both modes on one screen
- Keep radius small and consistent (≤3px) everywhere — buttons, windows, tags, inputs
- Use mono type for anything that represents data, metadata, or system output; reserve the serif for major headings only

## Rules: Don't
- Don't use soft blur-heavy shadows or glassy gradients anywhere except the one allowed case: a warm gradient fill under a line/area chart series
- Don't make buttons, tags, or windows fully rounded/pill-shaped — 3px is the ceiling
- Don't let decorative title-bar control dots double as real status indicators
- Don't default new screens to dark mode without an explicit request or a stated user preference

## Expected Behavior

- Follow the "every panel is a window" principle first, then mode consistency (don't mix light/dark tokens), then novelty.
- When uncertain whether dark mode applies, default to light and say a dark variant exists on request.
- Provide concrete token references (not "an orange accent," but "Brand, `#FF631A` in light mode / `#FF7A3D` in dark mode") in every recommendation.
- Keep guidance opinionated and implementation-focused — state hex values, font roles, border widths, and radii explicitly rather than describing them abstractly.

## QA Checklist

- [ ] Is every content container built as a bordered window with a title-bar strip, never a borderless flat card?
- [ ] Is the screen using light-mode tokens by default, with dark mode only present if explicitly requested?
- [ ] Is radius consistently ≤3px across windows, buttons, tags, and inputs?
- [ ] Is a warm gradient fill used only under chart line/area series, and nowhere else?
- [ ] Do decorative control dots ever double as real status communication? (They shouldn't — check for a paired text/tag instead.)
- [ ] Is the mono font used for data/metadata/system labels, with the serif reserved for major headings only?
- [ ] Does every interactive element have a visible focus-visible state distinct from its resting border?

For full token values (hex codes, exact type scale, spacing scale, and component measurements) in machine-readable form, see `DESIGN.md` in this skill's directory.
