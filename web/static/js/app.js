// Ethereum Validator Monitor - Client App
console.log('Ethereum Validator Monitor initialized');

// HTMX configuration
document.body.addEventListener('htmx:configRequest', (event) => {
  // Add CSRF token if needed
  const csrfToken = document.querySelector('meta[name="csrf-token"]');
  if (csrfToken) {
    event.detail.headers['X-CSRF-Token'] = csrfToken.content;
  }
});

// Dark Mode Theme Persistence
(function() {
  const THEME_KEY = 'eth-validator-theme';
  const html = document.documentElement;
  const themeToggle = document.getElementById('theme-toggle');
  const themeToggleMobile = document.querySelector('.theme-controller-mobile');

  // Load saved theme or default to light
  function loadTheme() {
    const savedTheme = localStorage.getItem(THEME_KEY) || 'light';
    html.setAttribute('data-theme', savedTheme);

    // Update toggle states
    const isDark = savedTheme === 'dark';
    if (themeToggle) {
      themeToggle.checked = isDark;
    }
    if (themeToggleMobile) {
      themeToggleMobile.checked = isDark;
    }
  }

  // Save theme preference
  function saveTheme(theme) {
    localStorage.setItem(THEME_KEY, theme);
    html.setAttribute('data-theme', theme);
  }

  // Handle desktop theme toggle
  if (themeToggle) {
    themeToggle.addEventListener('change', (e) => {
      const newTheme = e.target.checked ? 'dark' : 'light';
      saveTheme(newTheme);
      // Sync with mobile toggle
      if (themeToggleMobile) {
        themeToggleMobile.checked = e.target.checked;
      }
    });
  }

  // Handle mobile theme toggle
  if (themeToggleMobile) {
    themeToggleMobile.addEventListener('change', (e) => {
      const newTheme = e.target.checked ? 'dark' : 'light';
      saveTheme(newTheme);
      // Sync with desktop toggle
      if (themeToggle) {
        themeToggle.checked = e.target.checked;
      }
    });
  }

  // Initialize on page load
  loadTheme();
})();
