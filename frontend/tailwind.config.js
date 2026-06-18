/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        // Palette lifted from the reference TimeKeeper pages.
        bg: '#1a1a2e',
        card: '#16213e',
        panel: '#0f3460',
        accent: '#e94560',
        mint: '#4ecca3',
        muted: '#888888',
      },
    },
  },
  plugins: [],
};