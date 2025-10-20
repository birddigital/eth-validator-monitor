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
      },
    },
  },
  plugins: [
    require('daisyui')
  ],
  daisyui: {
    themes: [
      {
        light: {
          "primary": "#627eea",
          "secondary": "#454a75",
          "accent": "#67e8f9",
          "neutral": "#1f2937",
          "base-100": "#ffffff",
          "base-200": "#f3f4f6",
          "base-300": "#e5e7eb",
          "info": "#3abff8",
          "success": "#36d399",
          "warning": "#fbbd23",
          "error": "#f87272",
        },
      },
      "dark",
    ],
    darkTheme: "dark",
    base: true,
    styled: true,
    utils: true,
    logs: false,
  },
}
