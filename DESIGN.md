---
name: EVE-KILL
description: Shared visual conventions for the Go and templ frontend.
colors:
  brand-primary: "#3b82f6"
  brand-primary-hover: "#2563eb"
  brand-secondary: "#10b981"
  brand-accent: "#00aaff"
  text-primary: "#ffffff"
  text-secondary: "#9ca3af"
  text-tertiary: "#6b7280"
  bg-primary: "#111827"
  bg-secondary: "rgba(26, 26, 26, 0.8)"
  bg-tertiary: "#1f2937"
typography:
  page-title:
    fontFamily: "Exo 2"
    fontSize: "20px"
    fontWeight: 700
  navigation:
    fontFamily: "Exo 2"
    fontSize: "16px"
    fontWeight: 500
  fine:
    fontFamily: "Exo 2"
    fontSize: "12px"
    lineHeight: "16px"
rounded:
  md: "6px"
  lg: "8px"
  xl: "12px"
spacing:
  1: "4px"
  2: "8px"
  3: "12px"
  4: "16px"
  5: "20px"
  6: "24px"
  8: "32px"
---

# EVE-KILL design system

## Overview

Preserve the established EVE-KILL interface when extending the Go frontend.
The existing Nuxt implementation remains the reference for ported appearance and behavior.
Use existing components and source before building replacements.

The interface combines space imagery, translucent dark surfaces, Exo 2 typography, and compact data lists.
Ship renders, entity portraits, and EVE data provide its visual identity.
Refinements must improve readability and operation while retaining this identity.

## Colors

The frontmatter records default values, not fixed colors for every user.
Use semantic CSS properties from `internal/frontend/static/site.css` in components.
Theme preferences can override brand, text, background, border, and status colors.

`frontend/src/utils/themeData.ts` defines supported preferences and presets.
`frontend/src/browser/preferences.ts` applies preferences before paint.
Custom killboards can also supply a domain theme.

Use primary text for names and important values.
Use secondary text for supporting labels.
Check small text against its actual surface, including translucent surfaces over backgrounds.
Use a visible selected treatment as well as color for active controls.

Retain EVE security colors, corporation accents, loss rows, and ISK colors.
These colors convey data; they are not interchangeable decoration.

## Typography

Use the self-hosted Exo 2 font and the existing metric-adjusted fallbacks.
Reserve monospace for code, API paths, and measurements that benefit from aligned characters.
Use tabular numerals for changing counts and comparable values.

The generated utility scale differs from Tailwind defaults:

| Utility | Size | Line height |
| --- | --- | --- |
| `text-fine` | 12px | 16px |
| `text-xs` | 14px | 20px |
| `text-sm` | 16px | 24px |
| `text-base` | 18px | 28px |

Read `frontend/styles/components.css` before changing this scale.
Keep dense data compact, but keep names, controls, and explanatory text readable.
Truncate long names within their allocated column without hiding their link or focus indicator.

## Layout

Reuse the shared shell, page header, top lists, valuable cards, kill list, and entity components.
The outer content limit is 85rem; the inner limit is 84rem.
The navbar has a separate 80rem limit and a 64px height.

Keep search centered between navigation and account actions when space permits.
Use the existing mobile navigation when the desktop controls no longer fit.
Allow dense toolbars to wrap or scroll locally without making the document scroll horizontally.

Preserve image dimensions or aspect ratios before images load.
Load expensive sections as fragments with a visible loading state and a retry action after failure.
Keep filters, keyboard focus, and relevant scroll state through fragment replacement.

## Elevation & Depth

The space background belongs behind the content.
Use the existing glass panel treatment for related content.
Dropdowns and dialogs use more opaque surfaces so text remains readable.
Keep blur in the shared shell and overlays, where it separates layers.

## Shapes

Use the existing 6px, 8px, and 12px radii for controls, panels, and overlays.
Use pills for compact filters and status labels.
Use the shared SVG icon sprite rather than text characters as new icons.

## Components

- Keep navigation destinations as links, including links inside clickable rows.
- Give icon buttons accessible action names and visible keyboard focus.
- Expose selected, expanded, disabled, and loading states through native elements or ARIA.
- Use arrow-key navigation and one tab stop when a control implements the tab pattern.
- Preserve native modifier-click and new-tab behavior for links.
- Keep search dismissal, focus return, query cancellation, and empty states consistent across pages.
- Support reduced motion with static feedback that still communicates state.

## Do's and Don'ts

- Verify desktop and narrow layouts with real names and values.
- Verify default colors, a preset, and custom theme colors before changing shared components.
- Keep frontend behavior in the existing TypeScript and Stimulus architecture.
- Rebuild checked-in assets after changing TypeScript or utility CSS.
- Regenerate Go files after changing templ sources.
- Do not replace working components with generic dashboard or marketing layouts.
- Do not hard-code a blue accent where a theme token exists.
- Do not treat a successful HTTP response as proof that an interaction works.
