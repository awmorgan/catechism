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
        // Default to light mode for comfortable standard reading; user can toggle anytime
        return 'light';
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
    const MIN_FONT_SIZE = 1.0;
    const MAX_FONT_SIZE = 2.8;
    const FONT_STEP = 0.15;
    let currentFontSize = parseFloat(localStorage.getItem(FONT_KEY)) || 1.15;

    function applyFontSize(size) {
        currentFontSize = Math.min(MAX_FONT_SIZE, Math.max(MIN_FONT_SIZE, Math.round(size * 100) / 100));
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
        if (smallerBtn) smallerBtn.onclick = () => applyFontSize(currentFontSize - FONT_STEP);
        if (largerBtn) largerBtn.onclick = () => applyFontSize(currentFontSize + FONT_STEP);

        // --- TOC Drawer ---
        const drawer = document.getElementById('toc-drawer');
        const backdrop = document.getElementById('drawer-backdrop');
        const openBtn = document.getElementById('btn-open-drawer');
        const closeBtn = document.getElementById('btn-close-drawer');
        const bottomTocBtn = document.getElementById('btn-bottom-toc');

        function openDrawer() {
            if (!drawer || !backdrop) return;
            drawer.classList.add('active');
            backdrop.classList.add('active');
            drawer.setAttribute('aria-hidden', 'false');
            // Scroll to current page in drawer after slide-in begins
            setTimeout(() => {
                let currentLink = drawer.querySelector('.current-page') || drawer.querySelector('[aria-current="page"]');
                if (!currentLink) {
                    const currentPath = window.location.pathname.split('/').pop() || 'index.html';
                    currentLink = drawer.querySelector(`a[data-page="${currentPath}"]`);
                    if (currentLink) {
                        currentLink.classList.add('current-page');
                        currentLink.setAttribute('aria-current', 'page');
                    }
                }
                if (currentLink) {
                    if (!currentLink.querySelector('.toc-current-badge')) {
                        const badge = document.createElement('span');
                        badge.className = 'toc-current-badge';
                        badge.innerHTML = '<span class="toc-current-dot" aria-hidden="true">●</span> You are here';
                        currentLink.prepend(badge);
                    }
                    currentLink.scrollIntoView({ block: 'center', behavior: 'smooth' });
                }
            }, 120);
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
        if (bottomTocBtn) bottomTocBtn.onclick = openDrawer;

        // --- Jump to Paragraph ---
        const jumpForm = document.getElementById('jump-form');
        const jumpInput = document.getElementById('jump-input');
        if (jumpForm && jumpInput) {
            jumpInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    if (typeof jumpForm.requestSubmit === 'function') {
                        jumpForm.requestSubmit();
                    } else {
                        jumpForm.dispatchEvent(new Event('submit', { cancelable: true }));
                    }
                }
            });

            jumpForm.onsubmit = (e) => {
                e.preventDefault();
                const pNum = parseInt(jumpInput.value.trim(), 10);
                if (isNaN(pNum) || pNum < 1 || pNum > 2865) {
                    alert('Please enter a valid paragraph number between 1 and 2865.');
                    return;
                }

                // Dismiss virtual keyboard on mobile
                jumpInput.blur();

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

        // --- Responsive Footnote Popovers ---
        let footnotePopover = document.getElementById('footnote-popover');
        let footnoteBackdrop = document.getElementById('footnote-backdrop');

        function ensureFootnoteElements() {
            if (!footnoteBackdrop) {
                footnoteBackdrop = document.createElement('div');
                footnoteBackdrop.id = 'footnote-backdrop';
                footnoteBackdrop.className = 'footnote-backdrop';
                document.body.appendChild(footnoteBackdrop);
                footnoteBackdrop.onclick = closeFootnotePopover;
            }
            if (!footnotePopover) {
                footnotePopover = document.createElement('div');
                footnotePopover.id = 'footnote-popover';
                footnotePopover.className = 'footnote-popover';
                footnotePopover.setAttribute('role', 'dialog');
                footnotePopover.setAttribute('aria-label', 'Footnote');
                document.body.appendChild(footnotePopover);
            }
        }

        function closeFootnotePopover() {
            if (footnotePopover) footnotePopover.classList.remove('active');
            if (footnoteBackdrop) footnoteBackdrop.classList.remove('active');
        }

        function showFootnotePopover(refLink) {
            ensureFootnoteElements();
            const noteId = refLink.getAttribute('href');
            const noteEl = document.querySelector(noteId);
            if (!noteEl) return;

            // Clone note content without the backlink
            const clone = noteEl.cloneNode(true);
            const backlink = clone.querySelector('a[href^="#s"]');
            const noteNum = backlink ? backlink.textContent.trim() : refLink.textContent.trim();
            if (backlink) backlink.remove();

            footnotePopover.innerHTML = `
                <div class="footnote-popover-header">
                    <span class="footnote-popover-title">Footnote ${noteNum}</span>
                    <button type="button" class="footnote-popover-close" aria-label="Close footnote">✕</button>
                </div>
                <div class="footnote-popover-body">
                    ${clone.innerHTML.trim()}
                </div>
            `;

            footnotePopover.querySelector('.footnote-popover-close').onclick = closeFootnotePopover;

            const isMobile = window.innerWidth < 640;
            if (isMobile) {
                // Mobile bottom sheet
                footnoteBackdrop.classList.add('active');
                footnotePopover.classList.add('active');
            } else {
                // Desktop / Tablet floating popover positioned near superscript
                footnoteBackdrop.classList.add('active');
                footnotePopover.classList.add('active');

                const rect = refLink.getBoundingClientRect();
                const popoverWidth = 380;
                let left = rect.left + window.scrollX - (popoverWidth / 2) + (rect.width / 2);
                left = Math.max(16, Math.min(window.innerWidth - popoverWidth - 16, left));

                // Position above or below depending on space
                let top;
                if (rect.bottom + 220 > window.innerHeight && rect.top > 220) {
                    top = rect.top + window.scrollY - 200;
                } else {
                    top = rect.bottom + window.scrollY + 8;
                }

                footnotePopover.style.left = `${left}px`;
                footnotePopover.style.top = `${top}px`;
            }
        }

        // Intercept all footnote reference clicks
        document.querySelectorAll('a.note-ref').forEach(link => {
            link.onclick = (e) => {
                e.preventDefault();
                showFootnotePopover(link);
            };
        });

        // --- Keyboard Navigation ---
        document.addEventListener('keydown', (e) => {
            // Ignore if inside input/textarea
            if (['INPUT', 'TEXTAREA', 'SELECT'].includes(e.target.tagName)) return;

            if (e.key === 'Escape') {
                closeDrawer();
                closeFootnotePopover();
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

        // --- Active Reading Paragraph Tracker in Breadcrumbs ---
        const activeParaEl = document.getElementById('active-reading-para');
        if (activeParaEl) {
            const paraElements = Array.from(document.querySelectorAll('p[id^="p-"]'));
            if (paraElements.length > 0) {
                let ticking = false;

                function updateActivePara() {
                    ticking = false;
                    const topThreshold = 140; // below sticky header and breadcrumbs bar
                    let currentPara = null;

                    for (let i = 0; i < paraElements.length; i++) {
                        const rect = paraElements[i].getBoundingClientRect();
                        if (rect.top <= topThreshold) {
                            currentPara = paraElements[i];
                        } else {
                            if (!currentPara && rect.top < window.innerHeight * 0.7) {
                                currentPara = paraElements[i];
                            }
                            break;
                        }
                    }

                    if (currentPara) {
                        const pNum = currentPara.id.replace('p-', '');
                        activeParaEl.textContent = `¶ ${pNum}`;
                        activeParaEl.dataset.para = pNum;
                        activeParaEl.title = `Current paragraph ¶ ${pNum} (click to center)`;
                        activeParaEl.style.display = 'inline-flex';
                    } else if (window.scrollY < 80) {
                        activeParaEl.style.display = 'none';
                    }
                }

                activeParaEl.onclick = () => {
                    const pNum = activeParaEl.dataset.para;
                    if (pNum) {
                        const el = document.getElementById('p-' + pNum);
                        if (el) {
                            el.scrollIntoView({ block: 'center', behavior: 'smooth' });
                            el.classList.add('highlight-hit');
                            setTimeout(() => el.classList.remove('highlight-hit'), 2000);
                        }
                    }
                };

                window.addEventListener('scroll', () => {
                    if (!ticking) {
                        window.requestAnimationFrame(updateActivePara);
                        ticking = true;
                    }
                }, { passive: true });

                // Initial check on load
                updateActivePara();
            }
        }

        // Auto-scroll breadcrumbs trail so the active section / leaf is immediately visible on mobile
        const breadcrumbsTrail = document.querySelector('.breadcrumbs-trail');
        if (breadcrumbsTrail && window.innerWidth < 768) {
            setTimeout(() => {
                breadcrumbsTrail.scrollTo({ left: breadcrumbsTrail.scrollWidth, behavior: 'smooth' });
            }, 300);
        }
    });
})();

