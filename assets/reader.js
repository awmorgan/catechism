/**
 * Catechism Reader - Modern Client Controller
 * Handles theme, font sizing, drawer, navigation, and paragraph jumping.
 */
(() => {
    'use strict';

    // --- Theme Management ---
    const THEME_KEY = 'ccc-theme';
    function getPreferredTheme() {
        const saved = localStorage.getItem(THEME_KEY);
        if (saved === 'dark' || saved === 'light') return saved;
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    function applyTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        const btn = document.getElementById('btn-theme-toggle');
        if (btn) {
            btn.textContent = theme === 'dark' ? '☀️' : '🌙';
            btn.setAttribute('aria-label', theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode');
            btn.title = theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode';
        }
        localStorage.setItem(THEME_KEY, theme);
    }

    // Apply immediately before full DOM rendering to avoid flash
    applyTheme(getPreferredTheme());

    // --- Font Sizing Management ---
    const FONT_KEY = 'ccc-font-size';
    let currentFontSize = parseFloat(localStorage.getItem(FONT_KEY)) || 1.15;

    function applyFontSize(size) {
        currentFontSize = Math.min(2.0, Math.max(0.9, Math.round(size * 100) / 100));
        document.documentElement.style.setProperty('--reading-size', currentFontSize + 'rem');
        localStorage.setItem(FONT_KEY, String(currentFontSize));
    }
    applyFontSize(currentFontSize);

    document.addEventListener('DOMContentLoaded', () => {
        // Re-bind theme button in case DOM was loading
        const themeBtn = document.getElementById('btn-theme-toggle');
        if (themeBtn) {
            themeBtn.onclick = () => {
                const next = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
                applyTheme(next);
            };
        }

        // Font size buttons
        const smallerBtn = document.getElementById('btn-font-smaller');
        const largerBtn = document.getElementById('btn-font-larger');
        if (smallerBtn) smallerBtn.onclick = () => applyFontSize(currentFontSize - 0.1);
        if (largerBtn) largerBtn.onclick = () => applyFontSize(currentFontSize + 0.1);

        // --- TOC Drawer ---
        const drawer = document.getElementById('toc-drawer');
        const backdrop = document.getElementById('drawer-backdrop');
        const openBtn = document.getElementById('btn-open-drawer');
        const closeBtn = document.getElementById('btn-close-drawer');

        function openDrawer() {
            if (!drawer || !backdrop) return;
            drawer.classList.add('active');
            backdrop.classList.add('active');
            drawer.setAttribute('aria-hidden', 'false');
            // Scroll to current page in drawer
            const currentLink = drawer.querySelector('.current-page');
            if (currentLink) {
                requestAnimationFrame(() => {
                    currentLink.scrollIntoView({ block: 'center' });
                });
            }
        }

        function closeDrawer() {
            if (!drawer || !backdrop) return;
            drawer.classList.remove('active');
            backdrop.classList.remove('active');
            drawer.setAttribute('aria-hidden', 'true');
        }

        if (openBtn) openBtn.onclick = openDrawer;
        if (closeBtn) closeBtn.onclick = closeDrawer;
        if (backdrop) backdrop.onclick = closeDrawer;

        // --- Jump to Paragraph ---
        const jumpForm = document.getElementById('jump-form');
        const jumpInput = document.getElementById('jump-input');
        if (jumpForm && jumpInput) {
            jumpForm.onsubmit = (e) => {
                e.preventDefault();
                const pNum = parseInt(jumpInput.value.trim(), 10);
                if (isNaN(pNum) || pNum < 1 || pNum > 2865) {
                    alert('Please enter a valid paragraph number between 1 and 2865.');
                    return;
                }

                // Check if paragraph is already on current page
                const localTarget = document.getElementById('p-' + pNum);
                if (localTarget) {
                    localTarget.scrollIntoView({ block: 'center' });
                    localTarget.classList.add('highlight-hit');
                    setTimeout(() => localTarget.classList.remove('highlight-hit'), 2500);
                    history.pushState(null, '', '#p-' + pNum);
                    closeDrawer();
                    return;
                }

                // Use global window.CCC_PARA_MAP or fetch mapping
                if (window.CCC_PARA_MAP && window.CCC_PARA_MAP[pNum]) {
                    const rootPrefix = document.body.dataset.root || '';
                    window.location.href = rootPrefix + window.CCC_PARA_MAP[pNum] + '#p-' + pNum;
                } else {
                    // Fallback: fetch para-map.json
                    const rootPrefix = document.body.dataset.root || '';
                    fetch(rootPrefix + 'assets/para-map.json')
                        .then(res => res.json())
                        .then(map => {
                            window.CCC_PARA_MAP = map;
                            if (map[pNum]) {
                                window.location.href = rootPrefix + map[pNum] + '#p-' + pNum;
                            } else {
                                alert('Paragraph ' + pNum + ' not found.');
                            }
                        })
                        .catch(() => {
                            alert('Could not load paragraph directory.');
                        });
                }
            };
        }

        // --- Keyboard Navigation ---
        document.addEventListener('keydown', (e) => {
            // Ignore if inside input/textarea
            if (['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName)) return;

            if (e.key === 'Escape') {
                closeDrawer();
                const searchModal = document.getElementById('search-modal');
                if (searchModal && searchModal.open) searchModal.close();
            }

            if (e.key === 'ArrowLeft') {
                const prevLink = document.querySelector('a.nav-card.prev');
                if (prevLink && prevLink.href) window.location.href = prevLink.href;
            }

            if (e.key === 'ArrowRight') {
                const nextLink = document.querySelector('a.nav-card.next');
                if (nextLink && nextLink.href) window.location.href = nextLink.href;
            }
        });

        // --- Copy Paragraph Anchor on Click ---
        document.querySelectorAll('.paragraph-number a').forEach(a => {
            a.onclick = (e) => {
                e.preventDefault();
                const hash = a.getAttribute('href');
                const target = document.querySelector(hash);
                if (target) {
                    target.scrollIntoView({ block: 'center' });
                    target.classList.add('highlight-hit');
                    setTimeout(() => target.classList.remove('highlight-hit'), 2000);
                    history.pushState(null, '', hash);
                    if (navigator.clipboard) {
                        navigator.clipboard.writeText(window.location.href);
                    }
                }
            };
        });

        // If URL has a hash on load, highlight paragraph
        if (window.location.hash) {
            try {
                const target = document.querySelector(window.location.hash);
                if (target) {
                    setTimeout(() => {
                        target.scrollIntoView({ block: 'center' });
                        target.classList.add('highlight-hit');
                        setTimeout(() => target.classList.remove('highlight-hit'), 2500);
                    }, 150);
                }
            } catch { }
        }
    });
})();

