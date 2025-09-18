/** @type {import('postcss-load-config').Config} */
const config = {
  plugins: {
    // Tailwind CSS v4 with PostCSS integration
    '@tailwindcss/postcss': {
      // v4: Enhanced processing options
      content: ['./src/**/*.{js,ts,jsx,tsx,mdx}'],
      // Enable CSS nesting for better organization
      nesting: true,
      // Enable CSS custom properties optimization
      customProperties: true,
    },

    // Autoprefixer for cross-browser compatibility
    // Should come after Tailwind but before cssnano
    ...(process.env.NODE_ENV === 'production' && {
      autoprefixer: {
        // Target modern browsers but include fallbacks
        overrideBrowserslist: [
          '> 1%',
          'last 2 versions',
          'Firefox ESR',
          'not dead',
          'not IE 11',
          'not op_mini all',
        ],
        // Enable CSS Grid support
        grid: 'autoplace',
        // Flexbox support
        flexbox: 'no-2009',
      },
    }),

    // CSS Nano for production optimization
    // Should be the last plugin in the chain
    ...(process.env.NODE_ENV === 'production' && {
      cssnano: {
        preset: [
          'default',
          {
            // Optimize comments
            discardComments: {
              removeAll: true,
            },
            // Optimize whitespace
            normalizeWhitespace: true,
            // Optimize selectors
            minifySelectors: true,
            // Optimize parameters
            minifyParams: true,
            // Optimize font values
            minifyFontValues: true,
            // Preserve CSS custom properties for theme switching
            // This is important for Tailwind CSS custom properties
            cssDeclarationSorter: false,
            reduceIdents: false,
            // Optimize duplicates
            discardDuplicates: true,
            // Merge rules when safe
            mergeRules: true,
            // Merge longhand properties
            mergeLonghand: true,
            // Optimize colors
            colormin: true,
            // Don't remove unused CSS (Tailwind handles this)
            discardUnused: false,
            // Don't normalize z-index (can break stacking contexts)
            zindex: false,
            // Font family optimization
            minifyFontFamily: true,
          },
        ],
      },
    }),
  },
};

export default config;
