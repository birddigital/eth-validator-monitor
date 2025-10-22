/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./internal/web/**/*.templ",
    "./internal/web/**/*.go",
    "./web/**/*.html",
  ],
  theme: {
    extend: {
      colors: {
        'eth-primary': '#627eea',
        'eth-secondary': '#454a75',
        // Ether.fi inspired colors
        'eth-purple': '#474276',
        'eth-purple-light': '#655ea8',
        'eth-tan': '#ccaa77',
        'eth-cream': '#faf8f0',
        'eth-dark-bg': '#17171a',
      },
      fontFamily: {
        'display': ['Crimson Pro', 'serif'],
        'sans': ['DM Sans', 'ui-sans-serif', 'system-ui', 'sans-serif'],
      },
      backdropBlur: {
        xs: '2px',
      },
      boxShadow: {
        'glass': '0 8px 32px 0 rgba(31, 38, 135, 0.15)',
        'glow': '0 0 20px rgba(98, 126, 234, 0.3)',
        'glow-lg': '0 0 40px rgba(98, 126, 234, 0.4)',
        'gradient': '0 10px 40px -10px rgba(71, 66, 118, 0.4)',
      },
      animation: {
        'slide-down': 'slideDown 250ms ease-out',
        'slide-up': 'slideUp 250ms ease-in',
        'fade-in': 'fadeIn 500ms ease-in',
      },
      keyframes: {
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-20px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideUp: {
          '0%': { opacity: '1', transform: 'translateY(0)' },
          '100%': { opacity: '0', transform: 'translateY(-20px)' },
        },
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
      },
    },
  },
  plugins: [
    require('daisyui')
  ],
  daisyui: {
    themes: [
      {
        // Custom light theme
        light: {
          "primary": "#627eea",          // Ethereum blue
          "secondary": "#454a75",        // Dark slate
          "accent": "#ccaa77",           // Tan/gold accent
          "neutral": "#1f2937",          // Dark gray
          "base-100": "#ffffff",         // White background
          "base-200": "#f3f4f6",         // Light gray
          "base-300": "#e5e7eb",         // Medium gray
          "info": "#3abff8",             // Info blue
          "success": "#10b981",          // Success green (WCAG compliant)
          "warning": "#f59e0b",          // Warning amber (WCAG compliant)
          "error": "#ef4444",            // Error red (WCAG compliant)
        },
        // Custom eth-dark theme inspired by ether.fi
        "eth-dark": {
          "primary": "#7c93f5",          // Lighter Ethereum blue for dark
          "primary-content": "#faf8f0",  // Cream text
          "secondary": "#655ea8",        // Purple accent
          "secondary-content": "#faf8f0",
          "accent": "#ccaa77",           // Tan/gold accent
          "accent-content": "#0e0b31",   // Dark text on tan
          "neutral": "#2d3748",          // Slate gray
          "neutral-content": "#faf8f0",
          "base-100": "#0f172a",         // Very dark blue-gray (slate-900)
          "base-200": "#1e293b",         // Dark slate (slate-800)
          "base-300": "#334155",         // Medium slate (slate-700)
          "base-content": "#faf8f0",     // Cream text
          "info": "#3b82f6",             // Blue
          "info-content": "#ffffff",
          "success": "#10b981",          // Emerald
          "success-content": "#ffffff",
          "warning": "#f59e0b",          // Amber
          "warning-content": "#000000",
          "error": "#ef4444",            // Red
          "error-content": "#ffffff",
        }
      },
    ],
    darkTheme: "eth-dark",
    base: true,
    styled: true,
    utils: true,
    logs: false,
  },
}
