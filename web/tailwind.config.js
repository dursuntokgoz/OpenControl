/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        panel: {
          bg: '#0b1220',
          surface: '#111a2e',
          border: '#1f2b47',
          accent: '#3b82f6',
        },
      },
    },
  },
  plugins: [],
}
