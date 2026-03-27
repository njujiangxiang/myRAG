# Design System — myRAG

## Product Context
- **What this is:** Multi-user RAG (Retrieval-Augmented Generation) system for building intelligent knowledge bases
- **Who it's for:** Developers and small teams who need self-hosted document search and AI chat
- **Space/industry:** Developer tools / AI infrastructure
- **Project type:** Web application (Dashboard + Chat interface)
- **Competitors:** Microsoft GraphRAG, RAGFlow

## Aesthetic Direction
- **Direction:** Modern/Tech-inspired (Apple/Linear-inspired)
- **Decoration level:** Intentional — subtle tech aesthetics without clutter
- **Mood:** Professional, clean, modern. Tech-forward with subtle gradient accents.
- **Reference sites:** Linear, Vercel, Raycast, Arc Browser

## Typography

### Body: System Font Stack
**Rationale:** Best performance, native look and feel, no FOIT. Apple-style rendering with smooth anti-aliasing.

```css
font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'Helvetica Neue', sans-serif;
```

### Code: JetBrains Mono (optional)
**Rationale:** For code blocks and technical content when needed.

### Type Scale
| Level | Size | Weight | Usage |
|-------|------|--------|-------|
| H1 | 30px / 1.875rem | 700 | Page titles |
| H2 | 24px / 1.5rem | 600 | Section headers |
| H3 | 18px / 1.125rem | 600 | Card titles |
| Body | 14px / 0.875rem | 400 | Body text |
| Small | 12px / 0.75rem | 400/500 | Captions, labels |
| Mono | 13px / 0.8125rem | 400 | Code, file names |

## Color

### Approach: Apple-Inspired Tech Theme
OKLCH-based color system with blue-purple primary and blue-green accent. Light mode only with glass morphism effects.

### Primary (Blue-Purple)
| Token | OKLCH | Usage |
|-------|-------|-------|
| `--color-primary` | oklch(0.55 0.2 260) | Primary buttons, links, accents |
| `--color-primary-foreground` | oklch(0.98 0 0) | Text on primary background |
| `--color-ring` | oklch(0.55 0.2 260) | Focus rings |

### Accent (Blue-Green)
| Token | OKLCH | Usage |
|-------|-------|-------|
| `--color-accent` | oklch(0.6 0.18 200) | Accent buttons, highlights |
| `--color-accent-foreground` | oklch(0.98 0 0) | Text on accent background |

### Neutrals
| Token | OKLCH | Usage |
|-------|-------|-------|
| `--color-background` | oklch(0.98 0.002 286) | Page background |
| `--color-foreground` | oklch(0.2 0.01 286) | Primary text |
| `--color-card` | oklch(1 0 0) | Card backgrounds |
| `--color-card-foreground` | oklch(0.2 0.01 286) | Card text |
| `--color-secondary` | oklch(0.95 0.003 286) | Secondary backgrounds |
| `--color-secondary-foreground` | oklch(0.3 0.01 286) | Secondary text |
| `--color-muted` | oklch(0.95 0.003 286) | Muted backgrounds |
| `--color-muted-foreground` | oklch(0.5 0.01 286) | Muted text |
| `--color-border` | oklch(0.9 0.005 286) | Borders, dividers |
| `--color-input` | oklch(0.9 0.005 286) | Input borders |

### Semantic Colors
| Token | Tailwind | Usage |
|-------|----------|-------|
| `--success` | green-500 | Success states, indexed status |
| `--warning` | yellow-500 | Warnings, processing status |
| `--error` | red-500 | Errors, failed status |
| `--info` | blue-500 | Info alerts, tips |

### Tech Glow Effects
| Token | OKLCH | Usage |
|-------|-------|-------|
| `--color-glow-blue` | oklch(0.7 0.15 250 / 0.15) | Primary glow shadows |
| `--color-glow-purple` | oklch(0.7 0.15 280 / 0.15) | Accent glow shadows |

### Dark Mode Strategy
Not implemented. System uses light mode only with glass morphism effects for depth.

## Spacing

### Base Unit: 8px
All spacing derives from an 8px base for visual rhythm.

### Scale
| Token | Size | Usage |
|-------|------|-------|
| `--space-2xs` | 4px | Tight inline spacing |
| `--space-xs` | 8px | Compact internal padding |
| `--space-sm` | 12px | Button padding, form elements |
| `--space-md` | 16px | Standard padding |
| `--space-lg` | 24px | Card padding, section gaps |
| `--space-xl` | 32px | Large section spacing |
| `--space-2xl` | 48px | Page sections |
| `--space-3xl` | 64px | Hero spacing |

### Density: Comfortable
Prioritize scannability over information density. White space is a design element.

## Layout

### Approach: Hybrid
- **Dashboard areas:** Strict 12-column grid, predictable alignment
- **Marketing/Empty states:** Creative, centered layouts with personality
- **Chat interface:** Conversation bubbles, max-width content

### Grid
- **Desktop (≥1024px):** 12 columns, 24px gutters, 32px margins
- **Tablet (768-1023px):** 8 columns, 16px gutters, 24px margins
- **Mobile (<768px):** 4 columns, 16px gutters, 16px margins

### Max Content Width: 1200px
Content wider than 1200px reduces readability.

### Border Radius Scale
| Token | Size | Usage |
|-------|------|-------|
| `--radius-sm` | 6px | Small elements, tags |
| `--radius-md` | 8px | Buttons, inputs |
| `--radius-lg` | 12px | Cards, modals (default) |
| `--radius-xl` | 16px | Large containers, sections |
| `--radius-full` | 9999px | Pills, avatars, status badges |

## Motion

### Approach: Intentional
Subtle entrance animations and meaningful state transitions. Motion serves comprehension, not decoration.

### Easing
| Token | Value | Usage |
|-------|-------|-------|
| `--ease-out` | cubic-bezier(0.33, 0, 0, 1) | Elements entering |
| `--ease-in` | cubic-bezier(1, 0, 0.67, 1) | Elements exiting |
| `--ease-in-out` | cubic-bezier(0.67, 0, 0.33, 1) | Movements, transforms |

### Duration
| Token | Value | Usage |
|-------|-------|-------|
| `--duration-micro` | 50-100ms | Micro-interactions (checkbox, toggle) |
| `--duration-short` | 150ms | Hover states, small transitions |
| `--duration-medium` | 250ms | Standard animations, modals |
| `--duration-long` | 400-700ms | Page transitions, complex sequences |

## Shadows

### Soft, Layered
| Token | Value | Usage |
|-------|-------|-------|
| `--shadow-sm` | 0 1px 2px rgba(0,0,0,0.04) | Cards, buttons |
| `--shadow-md` | 0 4px 8px rgba(0,0,0,0.06) | Hover states, dropdowns |
| `--shadow-lg` | 0 8px 24px rgba(0,0,0,0.08) | Modals, popovers |
| `--shadow-xl` | 0 16px 48px rgba(0,0,0,0.1) | Full-page overlays |

## Components

### Button Variants
- **Primary:** Coral background, white text, subtle shadow
- **Secondary:** Neutral-100 background, neutral-700 text
- **Ghost:** Transparent, neutral-600 text, hover:bg-neutral-100

### Status Indicators
- **Indexed (success):** Green background (Tailwind green-500), indicates document fully processed
- **Processing (warning):** Yellow background (Tailwind yellow-500), indicates document being processed
- **Pending:** Blue background (Tailwind blue-500), indicates document uploaded but not yet processing
- **Error:** Red background (Tailwind red-500), indicates document processing failed

### Empty States
Always include:
1. Large emoji or icon (64px)
2. Friendly title (H3, neutral-800)
3. Helpful description (neutral-500)
4. Primary action button

### Chat Citations
Citation markers: Inline superscript with subtle background. References displayed below assistant message showing:
- Document count badge
- Expandable document list with filenames
- Page numbers when available
- Source links to original documents

## Decisions Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-03-25 | Initial design system created | Modern/Friendly aesthetic chosen over Professional Minimal or Data Dense to make complex AI concepts more accessible |
| 2026-03-25 | System font stack (not Google Fonts) | Performance and native feel. No FOIT issues. |
| 2026-03-25 | Large border radius (14px default) | Reinforces modern aesthetic with Apple-inspired rounded corners. |
| 2026-03-25 | OKLCH blue-purple primary | Tech-forward aesthetic. Differentiates from corporate blue while maintaining professionalism. |
| 2026-03-25 | Glass morphism effects | Adds depth and premium feel to the interface. |
| 2026-03-26 | Light mode only | Simplifies implementation. Target users prefer light mode for productivity apps. |
| 2026-03-26 | Tailwind color utilities (green-500, etc) | Pragmatic approach using Tailwind's built-in semantic colors. |

---

## Quick Reference (CSS Custom Properties)

```css
:root {
  /* Colors - OKLCH */
  --color-primary: oklch(0.55 0.2 260);
  --color-accent: oklch(0.6 0.18 200);
  --color-background: oklch(0.98 0.002 286);
  --color-foreground: oklch(0.2 0.01 286);
  --color-border: oklch(0.9 0.005 286);
  --success: oklch(0.72 0.19 142);  /* green-500 */
  --warning: oklch(0.71 0.19 70);   /* yellow-500 */
  --error: oklch(0.63 0.24 25);     /* red-500 */
  --info: oklch(0.67 0.19 250);     /* blue-500 */

  /* Typography */
  --font-body: -apple-system, BlinkMacSystemFont, 'SF Pro Display', sans-serif;
  --font-mono: 'JetBrains Mono', monospace;

  /* Spacing */
  --space-md: 16px;
  --space-lg: 24px;

  /* Radius */
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 14px;
  --radius-xl: 20px;

  /* Shadows */
  --shadow-glow-blue: 0 0 30px oklch(0.7 0.15 250 / 0.15);
  --shadow-glow-purple: 0 0 30px oklch(0.7 0.15 280 / 0.15);

  /* Motion */
  --duration-short: 200ms;
  --ease-default: cubic-bezier(0.4, 0, 0.2, 1);
}
```

---

*Generated by /design-consultation on 2026-03-25*
