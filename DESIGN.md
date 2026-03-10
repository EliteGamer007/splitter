# Theming & Branding Guide

Splitter's frontend is designed to be modern, clean, and highly customizable. It uses **Next.js** and **Tailwind CSS** with a core set of design tokens.

## Design Principles

1. **Clarity**: High contrast and clear typography to ensure readability of long-form and short-form content.
2. **Hierarchy**: A 3-column layout that prioritizes content while keeping navigation and context (trending/search) accessible.
3. **Responsiveness**: Fluid transitions between desktop, tablet, and mobile views.
4. **Consistency**: Reusable components based on a strict set of design tokens.

## Design Tokens (Tailwind Config)

Splitter uses a custom color palette defined in `tailwind.config.js` and `styles/globals.css`.

### Core Colors
- **Primary**: `#6366f1` (Indigo 500) - Used for primary actions, active tabs, and branding.
- **Secondary**: `#10b981` (Emerald 500) - Used for success states and secondary interactions.
- **Background**: `#ffffff` (Light) / `#0f172a` (Dark)
- **Surface**: Subtle grays (`#f8fafc` / `#1e293b`) for card backgrounds and sidebars.

### Typography
- **Headings**: `Inter` or `System Sans-Serif`. Weight: 700 (Bold).
- **Body**: `Inter`. Weight: 400 (Regular).
- **Monospace**: `JetBrains Mono` or `Consolas` for DIDs and code snippets.

## Layout System

Splitter uses a standard 3-column architecture:

- **Left Column**: Navigation (Home, Explore, Notifications, Profile).
- **Middle Column**: Main feed or content view.
- **Right Column**: Widgets (Search, Trending Hashtags, Recommended Instances).

## Components

The UI is built using the following core components:
- `PostCard`: Displays content, author, and interaction counts.
- `StoryCircle`: Represents an ephemeral story at the top of the feed.
- `ActionButton`: Unified styles for Like, Repost, and Reply.
- `DIDBadge`: A specialized monospace badge for displaying shortened DIDs.

## Customization

To add a new theme or modify existing styles, please refer to the [RECIPES.md](RECIPES.md#2-customize-the-frontend-theme) guide.
