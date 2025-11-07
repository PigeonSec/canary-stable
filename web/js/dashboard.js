// Dashboard - Live Data Polling Only
// All write operations (logout, clear matches) are handled by HTML forms
class Dashboard {
    constructor() {
        this.matches = [];
        this.filteredMatches = [];
        this.currentPage = 0;
        this.pageSize = 20;
        this.refreshInterval = null;

        this.init();
    }

    async init() {
        this.setupEventListeners();
        this.setupThemeToggle();

        // Initial load
        await this.loadMetrics();
        await this.loadMatches();
        await this.loadPerformanceMetrics();

        // Start auto-refresh for live data
        this.startAutoRefresh();
    }

    setupEventListeners() {
        // Search and filter (client-side)
        const refreshBtn = document.getElementById('refreshBtn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.loadMatches());
        }

        const searchInput = document.getElementById('searchInput');
        if (searchInput) {
            searchInput.addEventListener('input', () => this.filterMatches());
        }

        const timeRange = document.getElementById('timeRange');
        if (timeRange) {
            timeRange.addEventListener('change', () => this.loadMatches());
        }

        const priorityFilter = document.getElementById('priorityFilter');
        if (priorityFilter) {
            priorityFilter.addEventListener('change', () => this.filterMatches());
        }

        // Pagination
        const prevPage = document.getElementById('prevPage');
        if (prevPage) {
            prevPage.addEventListener('click', () => this.prevPage());
        }

        const nextPage = document.getElementById('nextPage');
        if (nextPage) {
            nextPage.addEventListener('click', () => this.nextPage());
        }

        // Clear matches button (shows confirmation)
        const clearBtn = document.getElementById('clearBtn');
        if (clearBtn) {
            clearBtn.addEventListener('click', () => {
                if (confirm('Are you sure you want to clear all matches from memory?')) {
                    const clearForm = document.getElementById('clearForm');
                    if (clearForm) {
                        clearForm.submit();
                    }
                }
            });
        }
    }

    setupThemeToggle() {
        const themeToggle = document.getElementById('themeToggle');
        if (!themeToggle) return;

        const html = document.documentElement;

        const savedTheme = localStorage.getItem('theme') || 'light';
        html.setAttribute('data-bs-theme', savedTheme);
        this.updateThemeIcon(savedTheme);

        themeToggle.addEventListener('click', () => {
            const currentTheme = html.getAttribute('data-bs-theme');
            const newTheme = currentTheme === 'light' ? 'dark' : 'light';
            html.setAttribute('data-bs-theme', newTheme);
            localStorage.setItem('theme', newTheme);
            this.updateThemeIcon(newTheme);
        });
    }

    updateThemeIcon(theme) {
        const icon = document.querySelector('#themeToggle i');
        if (icon) {
            icon.className = theme === 'light' ? 'bi bi-moon-fill' : 'bi bi-sun-fill';
        }
    }

    // Helper to safely update element text content
    safeSetText(elementId, text) {
        const element = document.getElementById(elementId);
        if (element) {
            element.textContent = text;
        }
    }

    async loadMetrics() {
        try {
            const response = await fetch('/api/metrics');
            if (!response.ok) throw new Error('Failed to load metrics');

            const data = await response.json();
            this.safeSetText('totalMatches', data.total_matches.toLocaleString());
            this.safeSetText('totalCerts', data.total_certs.toLocaleString());
            this.safeSetText('activeRules', data.rules_count.toLocaleString());

            // Format uptime
            const uptime = data.uptime_seconds;
            let uptimeStr = '';
            if (uptime < 60) {
                uptimeStr = uptime + 's';
            } else if (uptime < 3600) {
                uptimeStr = Math.floor(uptime / 60) + 'm';
            } else if (uptime < 86400) {
                uptimeStr = Math.floor(uptime / 3600) + 'h';
            } else {
                uptimeStr = Math.floor(uptime / 86400) + 'd';
            }
            this.safeSetText('uptime', uptimeStr);

            // Show clear button if there are matches
            const clearBtn = document.getElementById('clearBtn');
            if (clearBtn && data.recent_matches > 0) {
                clearBtn.style.display = '';
            }

            this.updateStatusBadge(true);
        } catch (error) {
            console.error('Error loading metrics:', error);
            this.updateStatusBadge(false);
        }
    }

    async loadPerformanceMetrics() {
        try {
            const response = await fetch('/api/metrics/performance?minutes=60');
            if (!response.ok) throw new Error('Failed to load performance metrics');

            const data = await response.json();
            const current = data.current;

            if (current) {
                this.safeSetText('certsPerMin', current.certs_per_minute.toLocaleString());
                this.safeSetText('matchesPerMin', current.matches_per_minute.toLocaleString());
                this.safeSetText('avgMatchTime', current.avg_match_time_us + ' μs');
                this.safeSetText('cpuUsage', current.cpu_percent.toFixed(1) + '%');
                this.safeSetText('memoryUsage', current.memory_used_mb.toFixed(1) + ' MB');
                this.safeSetText('goroutines', current.goroutine_count.toLocaleString());
            }
        } catch (error) {
            console.error('Error loading performance metrics:', error);
        }
    }

    async loadMatches() {
        const timeRangeEl = document.getElementById('timeRange');
        const timeRange = timeRangeEl ? timeRangeEl.value : '30';

        try {
            const response = await fetch(`/api/matches/recent?minutes=${timeRange}`);
            if (!response.ok) throw new Error('Failed to load matches');

            const data = await response.json();
            this.matches = data.matches || [];
            this.matches = this.sortByNewestFirst(this.matches);

            this.filterMatches();
            this.updateStatusBadge(true);
        } catch (error) {
            console.error('Error loading matches:', error);
            this.updateStatusBadge(false);
            this.matches = [];
            this.renderMatches();
        }
    }

    sortByNewestFirst(matches) {
        return matches.sort((a, b) => {
            const dateA = new Date(a.detected_at);
            const dateB = new Date(b.detected_at);
            return dateB - dateA;
        });
    }

    filterMatches() {
        const searchInputEl = document.getElementById('searchInput');
        const priorityFilterEl = document.getElementById('priorityFilter');

        const searchTerm = searchInputEl ? searchInputEl.value.toLowerCase() : '';
        const priorityFilter = priorityFilterEl ? priorityFilterEl.value : '';

        this.filteredMatches = this.matches.filter(match => {
            const domainMatch = match.dns_names.some(domain =>
                domain.toLowerCase().includes(searchTerm)
            );
            const priorityMatch = !priorityFilter || match.priority === priorityFilter;
            return domainMatch && priorityMatch;
        });

        this.currentPage = 0;
        this.renderMatches();
    }

    renderMatches() {
        const tbody = document.getElementById('matchesTableBody');
        if (!tbody) return;

        const start = this.currentPage * this.pageSize;
        const end = start + this.pageSize;
        const pageMatches = this.filteredMatches.slice(start, end);

        // Update counts
        this.safeSetText('matchCount', `${this.filteredMatches.length} matches`);
        this.safeSetText('matchCountFooter', `${this.filteredMatches.length} matches`);

        // Update pagination buttons
        const prevPage = document.getElementById('prevPage');
        const nextPage = document.getElementById('nextPage');
        if (prevPage) prevPage.disabled = this.currentPage === 0;
        if (nextPage) nextPage.disabled = end >= this.filteredMatches.length;

        if (pageMatches.length === 0) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="6" class="text-center text-muted py-5">
                        <i class="bi bi-inbox fs-1 d-block mb-2"></i>
                        No matches found. Adjust filters or wait for new certificates...
                    </td>
                </tr>
            `;
            return;
        }

        tbody.innerHTML = pageMatches.map(match => this.renderMatchRow(match)).join('');
    }

    renderMatchRow(match) {
        const timestamp = new Date(match.detected_at).toLocaleString();
        const domains = match.dns_names.slice(0, 3).join(', ') +
                       (match.dns_names.length > 3 ? ` (+${match.dns_names.length - 3} more)` : '');

        const priorityBadge = {
            critical: 'danger',
            high: 'warning',
            medium: 'info',
            low: 'secondary'
        }[match.priority] || 'secondary';

        const keywords = Array.isArray(match.matched_domains)
            ? match.matched_domains.join(', ')
            : match.matched_domains;

        return `
            <tr>
                <td><small>${this.escapeHtml(timestamp)}</small></td>
                <td>
                    <div class="text-truncate" style="max-width: 300px;" title="${this.escapeHtml(match.dns_names.join(', '))}">
                        ${this.escapeHtml(domains)}
                    </div>
                </td>
                <td><span class="badge bg-secondary">${this.escapeHtml(match.matched_rule)}</span></td>
                <td><span class="badge bg-${priorityBadge}">${this.escapeHtml(match.priority)}</span></td>
                <td><small><code>${this.escapeHtml(keywords)}</code></small></td>
                <td>
                    <a href="https://crt.sh/?q=${encodeURIComponent(match.tbs_sha256)}"
                       target="_blank"
                       rel="noopener noreferrer"
                       class="btn btn-sm btn-outline-primary"
                       title="View on crt.sh">
                        <i class="bi bi-box-arrow-up-right"></i>
                    </a>
                </td>
            </tr>
        `;
    }

    prevPage() {
        if (this.currentPage > 0) {
            this.currentPage--;
            this.renderMatches();
        }
    }

    nextPage() {
        const maxPage = Math.ceil(this.filteredMatches.length / this.pageSize) - 1;
        if (this.currentPage < maxPage) {
            this.currentPage++;
            this.renderMatches();
        }
    }

    startAutoRefresh() {
        // Refresh metrics and matches every 5 seconds
        this.refreshInterval = setInterval(() => {
            this.loadMetrics();
            this.loadMatches();
            this.loadPerformanceMetrics();
        }, 5000);
    }

    updateStatusBadge(online) {
        const badge = document.getElementById('statusBadge');
        if (!badge) return;

        if (online) {
            badge.className = 'badge bg-success';
            badge.innerHTML = '<i class="bi bi-check-circle me-1"></i>Online';
        } else {
            badge.className = 'badge bg-danger';
            badge.innerHTML = '<i class="bi bi-x-circle me-1"></i>Offline';
        }
    }

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize dashboard when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        const dashboard = new Dashboard();
    });
} else {
    // DOM already loaded
    const dashboard = new Dashboard();
}
