/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  // Avoid conflicting with Element Plus / existing global resets
  corePlugins: {
    preflight: false,
  },
  theme: {
    extend: {
      fontFamily: {
        sans: ['"SF Pro Text"', '-apple-system', 'BlinkMacSystemFont', 'system-ui', 'sans-serif'],
        display: ['"SF Pro Display"', '-apple-system', 'BlinkMacSystemFont', 'system-ui', 'sans-serif'],
      },
      colors: {
        apple: {
          blue: '#0071e3',
          blueDeep: '#0066cc',
          ink: '#1d1d1f',
          muted: '#6e6e73',
          surface: '#f5f5f7',
        },
      },
      boxShadow: {
        'apple-soft': '0 4px 6px rgba(0, 0, 0, 0.05)',
        'apple-nav': '0 1px 0 rgba(0, 0, 0, 0.06)',
      },
      transitionTimingFunction: {
        apple: 'cubic-bezier(0.4, 0, 0.2, 1)',
      },
    },
  },
  plugins: [],
}
