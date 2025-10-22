// Ethereum Validator Monitor - Client App
console.log('Ethereum Validator Monitor initialized');

// ===========================================================================
// HTMX ENHANCED CONFIGURATION
// ===========================================================================
document.body.addEventListener('htmx:configRequest', (event) => {
  // Add CSRF token if needed
  const csrfToken = document.querySelector('meta[name="csrf-token"]');
  if (csrfToken) {
    event.detail.headers['X-CSRF-Token'] = csrfToken.content;
  }

  // Add custom headers
  event.detail.headers['X-Requested-With'] = 'XMLHttpRequest';
});

// Global HTMX event listeners for smooth UX
document.body.addEventListener('htmx:beforeSwap', (event) => {
  // Handle 4xx/5xx errors gracefully
  if (event.detail.xhr.status >= 400) {
    console.error('HTMX Request failed:', event.detail.xhr.status);
  }
});

// Add loading indicator support
document.body.addEventListener('htmx:beforeRequest', (event) => {
  const loadingBar = document.getElementById('loading-bar');
  if (loadingBar) {
    loadingBar.classList.remove('hidden');
  }
});

document.body.addEventListener('htmx:afterRequest', (event) => {
  const loadingBar = document.getElementById('loading-bar');
  if (loadingBar) {
    loadingBar.classList.add('hidden');
  }
});

// Preserve scroll position on swaps
document.body.addEventListener('htmx:beforeSwap', (event) => {
  if (event.detail.target.hasAttribute('data-preserve-scroll')) {
    event.detail.shouldSwap = true;
    event.detail.isError = false;
  }
});

// ===========================================================================
// DARK MODE THEME PERSISTENCE
// ===========================================================================
document.addEventListener('DOMContentLoaded', function() {
  const THEME_KEY = 'eth-validator-theme';
  const html = document.documentElement;
  const themeToggle = document.getElementById('theme-toggle');
  const themeToggleMobile = document.querySelector('.theme-controller-mobile');

  // Load saved theme or default to light
  function loadTheme() {
    const savedTheme = localStorage.getItem(THEME_KEY) || 'light';
    html.setAttribute('data-theme', savedTheme);
    console.log('Theme loaded:', savedTheme);

    // Add smooth transition class
    html.style.transition = 'background-color 0.3s ease, color 0.3s ease';

    // Update toggle states
    const isDark = savedTheme === 'eth-dark';
    if (themeToggle) {
      themeToggle.checked = isDark;
    }
    if (themeToggleMobile) {
      themeToggleMobile.checked = isDark;
    }
  }

  // Save theme preference
  function saveTheme(theme) {
    console.log('Saving theme:', theme);
    localStorage.setItem(THEME_KEY, theme);
    html.setAttribute('data-theme', theme);
  }

  // Handle desktop theme toggle
  if (themeToggle) {
    themeToggle.addEventListener('change', (e) => {
      const newTheme = e.target.checked ? 'eth-dark' : 'light';
      console.log('Desktop toggle changed to:', newTheme);
      saveTheme(newTheme);
      // Sync with mobile toggle
      if (themeToggleMobile) {
        themeToggleMobile.checked = e.target.checked;
      }
    });
    console.log('Desktop theme toggle attached');
  } else {
    console.warn('Desktop theme toggle (#theme-toggle) not found');
  }

  // Handle mobile theme toggle
  if (themeToggleMobile) {
    themeToggleMobile.addEventListener('change', (e) => {
      const newTheme = e.target.checked ? 'eth-dark' : 'light';
      console.log('Mobile toggle changed to:', newTheme);
      saveTheme(newTheme);
      // Sync with desktop toggle
      if (themeToggle) {
        themeToggle.checked = e.target.checked;
      }
    });
    console.log('Mobile theme toggle attached');
  } else {
    console.warn('Mobile theme toggle (.theme-controller-mobile) not found');
  }

  // Initialize on page load
  loadTheme();

  // ===========================================================================
  // IMPROVED DROPDOWN BEHAVIOR
  // ===========================================================================

  // Enhanced dropdown control
  const dropdownToggles = document.querySelectorAll('[data-dropdown-toggle]');

  dropdownToggles.forEach(toggle => {
    const menu = toggle.nextElementSibling;

    toggle.addEventListener('click', (e) => {
      e.stopPropagation();
      const isExpanded = toggle.getAttribute('aria-expanded') === 'true';

      // Close other dropdowns
      document.querySelectorAll('[aria-expanded="true"]').forEach(other => {
        if (other !== toggle) {
          other.setAttribute('aria-expanded', 'false');
          const otherMenu = other.nextElementSibling;
          if (otherMenu) {
            otherMenu.classList.remove('dropdown-open');
          }
        }
      });

      // Toggle current dropdown
      toggle.setAttribute('aria-expanded', !isExpanded);

      if (!isExpanded && menu) {
        menu.classList.add('dropdown-open');
      } else if (menu) {
        menu.classList.remove('dropdown-open');
      }
    });
  });

  // Close dropdowns on outside click
  document.addEventListener('click', (event) => {
    const dropdowns = document.querySelectorAll('.dropdown');

    dropdowns.forEach(dropdown => {
      const dropdownContent = dropdown.querySelector('.dropdown-content');

      // If click is outside dropdown
      if (!dropdown.contains(event.target)) {
        // Remove open class
        if (dropdownContent) {
          dropdownContent.classList.remove('dropdown-open');
        }
        // Reset aria-expanded
        const toggle = dropdown.querySelector('[aria-expanded]');
        if (toggle) {
          toggle.setAttribute('aria-expanded', 'false');
        }
      }
    });
  });

  // Close dropdowns on Escape
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      document.querySelectorAll('[aria-expanded="true"]').forEach(toggle => {
        toggle.setAttribute('aria-expanded', 'false');
        const menu = toggle.nextElementSibling;
        if (menu) {
          menu.classList.remove('dropdown-open');
        }
      });
    }
  });

  // ===========================================================================
  // LAZY LOADING IMAGES
  // ===========================================================================

  const lazyImages = document.querySelectorAll('.lazy-load');
  if (lazyImages.length > 0) {
    const imageObserver = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          const img = entry.target;
          if (img.dataset.src) {
            img.src = img.dataset.src;
            img.classList.remove('lazy-load');
            imageObserver.unobserve(img);
          }
        }
      });
    });

    lazyImages.forEach(img => imageObserver.observe(img));
  }

  // ===========================================================================
  // KEYBOARD NAVIGATION ENHANCEMENTS
  // ===========================================================================

  // Add focus trap for modals (if implemented)
  const modals = document.querySelectorAll('[role="dialog"]');
  modals.forEach(modal => {
    const focusableElements = modal.querySelectorAll(
      'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled])'
    );

    if (focusableElements.length > 0) {
      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];

      modal.addEventListener('keydown', (e) => {
        if (e.key === 'Tab') {
          if (e.shiftKey && document.activeElement === firstElement) {
            e.preventDefault();
            lastElement.focus();
          } else if (!e.shiftKey && document.activeElement === lastElement) {
            e.preventDefault();
            firstElement.focus();
          }
        }
      });
    }
  });

  // ===========================================================================
  // SERVER-SENT EVENTS (SSE) FOR REAL-TIME UPDATES
  // ===========================================================================

  let eventSource = null;
  let reconnectAttempts = 0;
  const MAX_RECONNECT_ATTEMPTS = 5;
  const RECONNECT_DELAY = 3000; // 3 seconds

  function connectSSE() {
    // Only connect if on dashboard page
    const dashboardPage = document.getElementById('health-indicators');
    if (!dashboardPage) {
      console.log('Not on dashboard page, skipping SSE connection');
      return;
    }

    if (eventSource) {
      console.log('SSE already connected');
      return;
    }

    console.log('Connecting to SSE endpoint...');
    eventSource = new EventSource('/api/sse');

    eventSource.onopen = function() {
      console.log('SSE connection established');
      reconnectAttempts = 0;
    };

    // Listen for health status updates
    eventSource.addEventListener('health-status', function(event) {
      console.log('Received health-status event:', event.data);
      try {
        const data = JSON.parse(event.data);
        updateHealthIndicators(data);
      } catch (error) {
        console.error('Failed to parse health-status data:', error);
      }
    });

    // Listen for metrics updates
    eventSource.addEventListener('metrics-update', function(event) {
      console.log('Received metrics-update event:', event.data);
      try {
        const data = JSON.parse(event.data);
        updateMetrics(data);
      } catch (error) {
        console.error('Failed to parse metrics-update data:', error);
      }
    });

    // Listen for new alerts
    eventSource.addEventListener('new-alert', function(event) {
      console.log('Received new-alert event:', event.data);
      try {
        const data = JSON.parse(event.data);
        displayAlert(data);
      } catch (error) {
        console.error('Failed to parse new-alert data:', error);
      }
    });

    // Handle heartbeat events
    eventSource.addEventListener('heartbeat', function(event) {
      // Silent heartbeat, just log in verbose mode
      // console.log('Heartbeat received');
    });

    eventSource.onerror = function(error) {
      console.error('SSE connection error:', error);
      eventSource.close();
      eventSource = null;

      // Attempt to reconnect
      if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
        reconnectAttempts++;
        console.log(`Reconnecting in ${RECONNECT_DELAY}ms (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`);
        setTimeout(connectSSE, RECONNECT_DELAY);
      } else {
        console.error('Max reconnection attempts reached, giving up');
      }
    };
  }

  // Update health indicators based on SSE data
  function updateHealthIndicators(data) {
    const healthIndicators = document.getElementById('health-indicators');
    if (!healthIndicators) return;

    // Update database status
    const dbCard = healthIndicators.querySelector('[data-component="database"]');
    if (dbCard) {
      updateHealthCard(dbCard, data.database_status, 'Database');
    }

    // Update Redis status
    const redisCard = healthIndicators.querySelector('[data-component="redis"]');
    if (redisCard) {
      // Redis status not in current HealthStatusData, use database as placeholder
      // TODO: Add redis_status to HealthStatusData in events.go
      updateHealthCard(redisCard, 'healthy', 'Redis Cache');
    }

    // Update last sync timestamp
    const timestamp = new Date(data.last_sync * 1000);
    const timeElements = healthIndicators.querySelectorAll('.text-xs.text-gray-400');
    timeElements.forEach(el => {
      if (el.textContent.includes('Last checked:')) {
        el.textContent = `Last checked: ${formatTimeAgo(timestamp)}`;
      }
    });
  }

  // Update individual health card
  function updateHealthCard(card, status, name) {
    // Update status badge
    const badge = card.querySelector('.inline-flex.items-center');
    if (badge) {
      // Remove old status classes
      badge.className = 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium';

      // Add new status classes
      if (status === 'healthy') {
        badge.classList.add('bg-green-100', 'text-green-800', 'dark:bg-green-900/30', 'dark:text-green-400');
        badge.innerHTML = '<span class="w-2 h-2 mr-1.5 rounded-full bg-green-400 animate-pulse"></span>Healthy';
      } else if (status === 'degraded') {
        badge.classList.add('bg-yellow-100', 'text-yellow-800', 'dark:bg-yellow-900/30', 'dark:text-yellow-400');
        badge.innerHTML = '<span class="w-2 h-2 mr-1.5 rounded-full bg-yellow-400 animate-pulse"></span>Degraded';
      } else if (status === 'unhealthy') {
        badge.classList.add('bg-red-100', 'text-red-800', 'dark:bg-red-900/30', 'dark:text-red-400');
        badge.innerHTML = '<span class="w-2 h-2 mr-1.5 rounded-full bg-red-400 animate-pulse"></span>Unhealthy';
      } else {
        badge.classList.add('bg-gray-100', 'text-gray-800', 'dark:bg-gray-700', 'dark:text-gray-400');
        badge.innerHTML = '<span class="w-2 h-2 mr-1.5 rounded-full bg-gray-400"></span>Unknown';
      }
    }

    // Update icon
    const iconContainer = card.querySelector('.flex-shrink-0 > div');
    if (iconContainer) {
      // Remove old classes
      iconContainer.className = 'w-12 h-12 rounded-full flex items-center justify-center';

      // Add new classes based on status
      if (status === 'healthy') {
        iconContainer.classList.add('bg-green-100', 'dark:bg-green-900/30');
        iconContainer.innerHTML = `<svg class="w-6 h-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
        </svg>`;
      } else if (status === 'degraded') {
        iconContainer.classList.add('bg-yellow-100', 'dark:bg-yellow-900/30');
        iconContainer.innerHTML = `<svg class="w-6 h-6 text-yellow-600 dark:text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
        </svg>`;
      } else if (status === 'unhealthy') {
        iconContainer.classList.add('bg-red-100', 'dark:bg-red-900/30');
        iconContainer.innerHTML = `<svg class="w-6 h-6 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
        </svg>`;
      } else {
        iconContainer.classList.add('bg-gray-100', 'dark:bg-gray-700');
        iconContainer.innerHTML = `<svg class="w-6 h-6 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>`;
      }
    }
  }

  // Format timestamp as relative time
  function formatTimeAgo(date) {
    const seconds = Math.floor((new Date() - date) / 1000);

    if (seconds < 60) {
      return `${seconds} seconds ago`;
    } else if (seconds < 3600) {
      const minutes = Math.floor(seconds / 60);
      return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
    } else if (seconds < 86400) {
      const hours = Math.floor(seconds / 3600);
      return `${hours} hour${hours > 1 ? 's' : ''} ago`;
    } else {
      const days = Math.floor(seconds / 86400);
      return `${days} day${days > 1 ? 's' : ''} ago`;
    }
  }

  // Update metrics (placeholder for future implementation)
  function updateMetrics(data) {
    console.log('Updating metrics:', data);
    // TODO: Implement metrics update logic
  }

  // Display alert notification (placeholder for future implementation)
  function displayAlert(data) {
    console.log('Displaying alert:', data);
    // TODO: Implement alert notification logic
  }

  // Connect to SSE on page load
  connectSSE();

  // Close SSE connection when page unloads
  window.addEventListener('beforeunload', function() {
    if (eventSource) {
      eventSource.close();
      console.log('SSE connection closed');
    }
  });

  console.log('All event listeners attached successfully');
});
