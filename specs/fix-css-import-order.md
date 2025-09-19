# Specification: Fix CSS Import Order in MCP Portal Frontend

**Title**: Fix CSS Import Order Violation in globals.css
**Status**: Draft
**Authors**: Claude Code
**Date**: 2025-09-18
**Type**: Bugfix

## Overview

The MCP Portal frontend fails to start with a CSS parsing error due to incorrect placement of `@import` statements in the `globals.css` file. The CSS specification requires that `@import` rules must precede all other rules except `@charset` and `@layer` statements. Currently, font imports appear after the Tailwind CSS import, violating this requirement.

## Background/Problem Statement

When running `npm run dev` in the frontend directory, the application fails with the following error:

```
@import rules must precede all rules aside from @charset and @layer statements
```

The error indicates that `@import` statements for Google Fonts appear at line 2082, but the file only has 872 lines, suggesting CSS duplication during build. The root cause is that Tailwind CSS v4 uses `@import 'tailwindcss'` which expands into actual CSS rules, and any subsequent `@import` statements violate CSS specification requirements.

## Goals

- ✅ Resolve the CSS import order violation preventing the application from starting
- ✅ Preserve all existing functionality including font loading and styling
- ✅ Maintain compatibility with Tailwind CSS v4 import structure
- ✅ Ensure optimal font loading performance
- ✅ Keep the solution simple and maintainable

## Non-Goals

- ❌ Refactoring the entire CSS architecture
- ❌ Changing the Tailwind CSS version or configuration
- ❌ Modifying the Next.js font loading strategy
- ❌ Removing any existing styles or components
- ❌ Changing the build system or bundler configuration

## Technical Dependencies

- **Tailwind CSS v4**: Using new `@import` syntax instead of directives
- **Next.js 15.5.3**: With Turbopack for development
- **Google Fonts**: Inter and JetBrains Mono variable fonts
- **CSS Specification**: W3C CSS Cascade and Import Rules

## Detailed Design

### Root Cause Analysis

The issue stems from CSS import order requirements:

1. CSS specification mandates `@import` rules must come first (except after `@charset` and `@layer`)
2. Tailwind v4 uses `@import 'tailwindcss'` which expands into actual CSS rules
3. Font `@import` statements placed after Tailwind violate this rule
4. The build process concatenates CSS, showing errors at non-existent line numbers

### Implementation Approach

#### Option 1: Move Font Imports Before Tailwind (Recommended)

```css
/* Font imports MUST come first */
@import url("https://fonts.googleapis.com/css2?family=Inter:ital,opsz,wght@0,14..32,100..900;1,14..32,100..900&display=swap");
@import url("https://fonts.googleapis.com/css2?family=JetBrains+Mono:ital,wght@0,100..800;1,100..800&display=swap");

/* Then Tailwind CSS v4 */
@import "tailwindcss";

/* Then all other styles... */
@layer base {
  /* ... */
}
```

#### Option 2: Use Next.js Font Loading (Alternative)

Since Next.js already loads fonts via `next/font/google` in `layout.tsx`, the CSS imports are redundant:

```css
/* Remove redundant font imports */
/* Tailwind CSS v4 */
@import "tailwindcss";

/* All other styles... */
@layer base {
  /* ... */
}
```

### Code Structure Changes

**File: `/cmd/docker-mcp/portal/frontend/src/app/globals.css`**

The file will be modified to:

1. Move font `@import` statements to the very top (before Tailwind)
2. Remove redundant `@font-face` declarations (fonts are already loaded via imports)
3. Maintain all existing CSS variables, layers, and component styles

### Integration with Existing Systems

- **Next.js Font Loading**: Already configured in `layout.tsx` with font variables
- **Tailwind Configuration**: No changes needed, continues using v4 import syntax
- **CSS Custom Properties**: All existing variables remain functional
- **Component Styles**: No impact on existing component styling

## User Experience

No visible changes to users. The application will:

- Start successfully without CSS parsing errors
- Load fonts with the same performance characteristics
- Display all UI elements with identical styling
- Maintain dark/light theme functionality

## Testing Strategy

### Unit Tests

- Verify CSS file syntax is valid
- Ensure all imports are in correct order
- Validate CSS custom properties are defined

### Integration Tests

- Test application starts without errors
- Verify fonts load correctly in browser
- Check all components render with proper styling
- Validate theme switching works

### E2E Tests

- Full application flow with font rendering
- Cross-browser font display verification
- Performance metrics for font loading

### Manual Testing Checklist

```bash
# 1. Clean build
rm -rf .next node_modules/.cache

# 2. Start development server
npm run dev

# 3. Verify no CSS parsing errors
# 4. Check browser console for errors
# 5. Verify fonts load (Network tab)
# 6. Test theme switching
# 7. Check all UI components display correctly
```

## Performance Considerations

### Font Loading Optimization

- Google Fonts CDN provides optimal caching
- `font-display: swap` ensures text remains visible during load
- Variable fonts reduce total download size
- Next.js font optimization still applies via `next/font/google`

### CSS Bundle Size

- No increase in bundle size (reordering only)
- Removal of redundant `@font-face` rules reduces size slightly
- Tailwind tree-shaking remains effective

## Security Considerations

- Google Fonts CDN is already whitelisted in CSP headers
- No new external resources introduced
- Font loading remains from trusted sources only

## Documentation

### Code Comments

Add clarifying comments in `globals.css`:

```css
/*
 * CSS Import Order (Critical):
 * 1. @charset (if needed)
 * 2. @import statements (fonts, external CSS)
 * 3. @import 'tailwindcss'
 * 4. @layer rules
 * 5. All other CSS rules
 *
 * DO NOT place @import after tailwindcss or @layer rules
 */
```

### Update Developer Notes

Document the CSS import order requirement in:

- Frontend README
- Developer setup guide
- CSS architecture documentation

## Implementation Phases

### Phase 1: Immediate Fix (MVP)

1. Reorder imports in `globals.css`
2. Test application starts successfully
3. Verify fonts load correctly

### Phase 2: Optimization (Optional)

1. Remove redundant `@font-face` declarations
2. Consolidate font loading strategy
3. Add CSS validation to build process

### Phase 3: Prevention (Future)

1. Add ESLint rule for CSS import order
2. Create pre-commit hook for CSS validation
3. Document CSS architecture decisions

## Open Questions

1. **Should we remove CSS font imports entirely?**

   - Since Next.js handles font loading via `next/font/google`, the CSS imports may be redundant
   - Decision: Keep for now as fallback, can remove if performance testing shows no benefit

2. **Should we add CSS linting?**
   - Stylelint could prevent future import order issues
   - Decision: Consider in Phase 3 after immediate fix

## References

- [CSS @import Rule - MDN](https://developer.mozilla.org/en-US/docs/Web/CSS/@import)
- [Tailwind CSS v4 Documentation](https://tailwindcss.com/docs/installation)
- [Next.js Font Optimization](https://nextjs.org/docs/app/building-your-application/optimizing/fonts)
- [CSS Cascade Specification](https://www.w3.org/TR/css-cascade-4/#at-import)

## Implementation Checklist

- [ ] Move font imports before Tailwind import
- [ ] Remove redundant @font-face declarations
- [ ] Add explanatory comments
- [ ] Test development server starts
- [ ] Verify fonts load in browser
- [ ] Check all UI components render correctly
- [ ] Test theme switching functionality
- [ ] Update documentation if needed
