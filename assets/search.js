/**
 * Catechism Reader - Client Search Modal
 * Catechism Reader - Modern Client Search Controller
 * Features: Context-centered snippets, search persistence, in-text word highlighting, and floating runner bar.
 */
(() => {
    'use strict';

    let searchIndex = null;
    let isFetching = false;
    const WHOLE_WORDS_KEY = 'ccc-search-whole-words';
    const ACTIVE_SEARCH_KEY = 'ccc-active-search';

    // --- State Persistence Helpers ---
    function getActiveSearch() {
        try {
            const raw = sessionStorage.getItem(ACTIVE_SEARCH_KEY);
            return raw ? JSON.parse(raw) : null;
        } catch {
            return null;
        }
    }

    function setActiveSearch(data) {
        try {
            if (data) {
                sessionStorage.setItem(ACTIVE_SEARCH_KEY, JSON.stringify(data));
            } else {
                sessionStorage.removeItem(ACTIVE_SEARCH_KEY);
            }
        } catch { }
    }

    function escapeRegex(str) {
        return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    }

    function buildTermRegex(term, isWholeWord) {
        const escaped = escapeRegex(term);
        if (!isWholeWord) {
            return new RegExp(escaped, 'iu');
        }
        return new RegExp('(?<![\\p{L}\\p{N}_])' + escaped + '(?![\\p{L}\\p{N}_])', 'iu');
    }

    // --- Context-Centered Snippet Generator ---
    function getMatchSnippet(text, terms, isWhole) {
        if (!text) return { text: '', prefixEllipsis: false, suffixEllipsis: false };
        if (text.length <= 220) return { text, prefixEllipsis: false, suffixEllipsis: false };

        let firstMatchIdx = -1;
        let matchLen = 0;
        for (const term of terms) {
            const rx = buildTermRegex(term, isWhole);
            const m = rx.exec(text);
            if (m && (firstMatchIdx === -1 || m.index < firstMatchIdx)) {
                firstMatchIdx = m.index;
                matchLen = m[0].length;
            }
        }

        if (firstMatchIdx === -1) {
            return { text: text.slice(0, 220), prefixEllipsis: false, suffixEllipsis: true };
        }

        const windowSize = 220;
        const beforePadding = 60;
        let start = Math.max(0, firstMatchIdx - beforePadding);
        let end = Math.min(text.length, start + windowSize);

        if (end - start < windowSize && start > 0) {
            start = Math.max(0, end - windowSize);
        }

        // Adjust to word boundaries if possible
        if (start > 0) {
            const nextSpace = text.indexOf(' ', start);
            if (nextSpace !== -1 && nextSpace < firstMatchIdx) {
                start = nextSpace + 1;
            }
        }
        if (end < text.length) {
            const lastSpace = text.lastIndexOf(' ', end);
            if (lastSpace !== -1 && lastSpace > firstMatchIdx + matchLen) {
                end = lastSpace;
            }
        }

        return {
            text: text.slice(start, end),
            prefixEllipsis: start > 0,
            suffixEllipsis: end < text.length
        };
    }

    // --- In-Text Word Highlighting ---
    function clearInTextWordHighlights() {
        document.querySelectorAll('mark.search-word-highlight').forEach(mark => {
            const parent = mark.parentNode;
            if (parent) {
                parent.replaceChild(document.createTextNode(mark.textContent), mark);
                parent.normalize();
            }
        });
    }

    function highlightWordsInElement(container, terms, isWhole) {
        clearInTextWordHighlights();
        if (!container || !terms.length) return;

        const highlightPattern = terms.map(t => {
            const esc = escapeRegex(t);
            return isWhole ? '(?<![\\p{L}\\p{N}_])' + esc + '(?![\\p{L}\\p{N}_])' : esc;
        }).join('|');
        const rx = new RegExp('(' + highlightPattern + ')', 'giu');

        // Walk text nodes, skipping footnote references and paragraph anchors
        const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT, {
            acceptNode: (node) => {
                const parent = node.parentElement;
                if (!parent) return NodeFilter.FILTER_REJECT;
                if (parent.tagName === 'SUP' || parent.classList.contains('paragraph-number') || parent.tagName === 'MARK' || parent.closest('details.footnotes')) {
                    return NodeFilter.FILTER_REJECT;
                }
                return NodeFilter.FILTER_ACCEPT;
            }
        });

        const textNodes = [];
        let curr;
        while ((curr = walker.nextNode())) {
            textNodes.push(curr);
        }

        for (const node of textNodes) {
            const val = node.nodeValue;
            if (!rx.test(val)) continue;
            rx.lastIndex = 0;

            const frag = document.createDocumentFragment();
            let lastIdx = 0;
            let m;
            while ((m = rx.exec(val)) !== null) {
                if (m.index > lastIdx) {
                    frag.appendChild(document.createTextNode(val.slice(lastIdx, m.index)));
                }
                const mark = document.createElement('mark');
                mark.className = 'search-word-highlight';
                mark.textContent = m[0];
                frag.appendChild(mark);
                lastIdx = rx.lastIndex;
            }
            if (lastIdx < val.length) {
                frag.appendChild(document.createTextNode(val.slice(lastIdx)));
            }

            if (node.parentNode) {
                node.parentNode.replaceChild(frag, node);
            }
        }
    }

    // --- Search Navigation Runner Bar ---
    function updateRunnerBar() {
        const activeSearch = getActiveSearch();
        let bar = document.getElementById('search-runner-bar');

        if (!activeSearch || !activeSearch.matches || activeSearch.currentIndex === undefined || activeSearch.currentIndex < 0) {
            if (bar) bar.remove();
            clearInTextWordHighlights();
            return;
        }

        const { query, matches, currentIndex } = activeSearch;
        if (!bar) {
            bar = document.createElement('div');
            bar.id = 'search-runner-bar';
            bar.className = 'search-runner-bar';
            bar.setAttribute('role', 'region');
            bar.setAttribute('aria-label', 'Search Navigation');
            document.body.appendChild(bar);
        }

        const isFirst = currentIndex <= 0;
        const isLast = currentIndex >= matches.length - 1;

        const countFormatted = matches.length.toLocaleString();
        const currentHitFormatted = (currentIndex + 1).toLocaleString();

        bar.innerHTML = `
            <div class="search-runner-info" id="btn-runner-info" role="button" tabindex="0" title="Click to view all results" aria-label="Search results for ${escapeHtml(query)}">
                <span class="search-runner-icon" aria-hidden="true">🔍</span>
                <span class="search-runner-query" title="${escapeHtml(query)}">${escapeHtml(query)}</span>
                <span class="search-runner-count">${currentHitFormatted} / ${countFormatted}</span>
            </div>
            <div class="search-runner-actions">
                <button type="button" id="btn-runner-prev" class="btn-runner" title="Previous hit" aria-label="Previous match" ${isFirst ? 'disabled' : ''}>
                    <span class="btn-runner-icon">‹</span><span class="btn-runner-text"> Prev</span>
                </button>
                <button type="button" id="btn-runner-next" class="btn-runner" title="Next hit" aria-label="Next match" ${isLast ? 'disabled' : ''}>
                    <span class="btn-runner-text">Next </span><span class="btn-runner-icon">›</span>
                </button>
                <button type="button" id="btn-runner-list" class="btn-runner" title="View all results" aria-label="View all results">
                    <span class="btn-runner-list-icon" aria-hidden="true">≡</span><span class="btn-runner-text"> Results</span>
                </button>
                <button type="button" id="btn-runner-close" class="btn-runner btn-runner-close" title="Exit search" aria-label="Close search">✕</button>
            </div>
        `;

        const openModal = () => {
            const modal = document.getElementById('search-modal');
            if (modal) modal.showModal();
        };

        const runnerInfo = document.getElementById('btn-runner-info');
        if (runnerInfo) {
            runnerInfo.onclick = openModal;
            runnerInfo.onkeydown = (e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    openModal();
                }
            };
        }

        document.getElementById('btn-runner-prev').onclick = () => stepHit(-1);
        document.getElementById('btn-runner-next').onclick = () => stepHit(1);
        document.getElementById('btn-runner-list').onclick = openModal;
        document.getElementById('btn-runner-close').onclick = () => {
            setActiveSearch(null);
            updateRunnerBar();
        };
    }

    function stepHit(delta) {
        const activeSearch = getActiveSearch();
        if (!activeSearch || !activeSearch.matches) return;

        const nextIdx = activeSearch.currentIndex + delta;
        if (nextIdx < 0 || nextIdx >= activeSearch.matches.length) return;

        navigateToHit(nextIdx);
    }

    function scrollToSearchMatch(containerEl) {
        if (!containerEl) return;

        // Clear active hit state on any previously active marks
        document.querySelectorAll('mark.search-active-hit').forEach(m => m.classList.remove('search-active-hit'));

        const marks = containerEl.querySelectorAll('mark.search-word-highlight');
        if (marks.length > 0) {
            const targetMark = marks[0];
            targetMark.classList.add('search-active-hit');

            // Center the specific matching word in the viewport
            targetMark.scrollIntoView({ block: 'center', behavior: 'smooth' });

            // Re-verify after layout stabilization to ensure it is not hidden by sticky header or runner bar
            setTimeout(() => {
                const rect = targetMark.getBoundingClientRect();
                const vh = window.innerHeight;
                if (rect.top < 110 || rect.bottom > vh - 90) {
                    targetMark.scrollIntoView({ block: 'center' });
                }
            }, 120);
        } else {
            containerEl.scrollIntoView({ block: 'center', behavior: 'smooth' });
        }
    }

    function navigateToHit(targetIndex) {
        const activeSearch = getActiveSearch();
        if (!activeSearch || !activeSearch.matches[targetIndex]) return;

        activeSearch.currentIndex = targetIndex;
        setActiveSearch(activeSearch);

        const hit = activeSearch.matches[targetIndex];
        const rootPrefix = document.body.dataset.root || '';
        const currentPath = window.location.pathname;

        // Check if hit is on the current page
        if (currentPath.endsWith(hit.path) || (currentPath.endsWith('/') && hit.path === 'index.html')) {
            updateRunnerBar();
            const pEl = document.getElementById('p-' + hit.p);
            if (pEl) {
                highlightWordsInElement(pEl, activeSearch.terms || activeSearch.query.split(/\s+/), activeSearch.isWhole);
                scrollToSearchMatch(pEl);
            }
            history.pushState(null, '', '#p-' + hit.p);
        } else {
            // Navigate to target page
            window.location.href = rootPrefix + hit.path + '#p-' + hit.p;
        }
    }

    function escapeHtml(str) {
        return (str || '').replace(/[&<>"']/g, m => ({
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            '"': '&quot;',
            "'": '&#39;'
        }[m]));
    }

    // --- Main Initializer ---
    document.addEventListener('DOMContentLoaded', () => {
        const modal = document.getElementById('search-modal');
        const input = document.getElementById('search-query');
        const resultsContainer = document.getElementById('search-results');
        const openBtn = document.getElementById('btn-open-search');
        const closeBtn = document.getElementById('btn-close-search');
        const wholeWordsCheckbox = document.getElementById('search-whole-words');
        const searchCountEl = document.getElementById('search-count');

        if (!modal || !input || !resultsContainer) return;

        let wholeWords = localStorage.getItem(WHOLE_WORDS_KEY) !== 'false';
        if (wholeWordsCheckbox) {
            wholeWordsCheckbox.checked = wholeWords;
            wholeWordsCheckbox.onchange = () => {
                wholeWords = wholeWordsCheckbox.checked;
                localStorage.setItem(WHOLE_WORDS_KEY, String(wholeWords));
                triggerSearch();
            };
        }

        function ensureIndex(callback) {
            if (searchIndex) {
                callback(searchIndex);
                return;
            }
            if (isFetching) return;
            isFetching = true;
            const rootPrefix = document.body.dataset.root || '';
            fetch(rootPrefix + 'assets/search-index.json')
                .then(res => res.json())
                .then(data => {
                    searchIndex = data;
                    isFetching = false;
                    callback(searchIndex);
                })
                .catch(() => {
                    isFetching = false;
                    resultsContainer.innerHTML = '<p style="padding:1rem;color:red;">Error loading search index.</p>';
                });
        }

        function openSearch() {
            modal.showModal();
            input.value = '';
            if (searchCountEl) searchCountEl.textContent = '';
            resultsContainer.innerHTML = '<p style="padding:1rem;color:var(--text-muted);">Type words to search paragraphs across the Catechism...</p>';
            const activeSearch = getActiveSearch();

            // Restore previous search results if available
            if (activeSearch && activeSearch.query) {
                input.value = activeSearch.query;
                if (activeSearch.matches && activeSearch.matches.length > 0) {
                    renderResults(activeSearch.matches, activeSearch.terms || activeSearch.query.split(/\s+/), activeSearch.isWhole);
                } else {
                    triggerSearch();
                }
            } else {
                input.value = '';
                if (searchCountEl) searchCountEl.textContent = '';
                resultsContainer.innerHTML = '<p style="padding:1rem;color:var(--text-muted);">Type words to search paragraphs across the Catechism...</p>';
            }

            input.focus();
            ensureIndex(() => { });
        }

        if (openBtn) openBtn.onclick = openSearch;
        if (closeBtn) closeBtn.onclick = () => modal.close();

        // Backdrop click dismisses
        modal.addEventListener('click', (e) => {
            if (e.target === modal) modal.close();
        });

        // Keyboard shortcut '/' or 'Ctrl+K' / 'Cmd+K'
        document.addEventListener('keydown', (e) => {
            if ((e.key === '/' || ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k')) && !['INPUT', 'TEXTAREA'].includes(e.target.tagName)) {
                e.preventDefault();
                openSearch();
            }
        });

        // Debounced search
        let debounceTimer;
        function triggerSearch() {
            clearTimeout(debounceTimer);
            const query = input.value.trim();
            if (!query || query.length < 2) {
                if (searchCountEl) searchCountEl.textContent = '';
                resultsContainer.innerHTML = '<p style="padding:1rem;color:var(--text-muted);">Type at least 2 characters to search...</p>';
                return;
            }
            debounceTimer = setTimeout(() => {
                ensureIndex((index) => {
                    performSearch(query, index);
                });
            }, 150);
        }

        input.addEventListener('input', triggerSearch);

        // Dismiss mobile keyboard when user scrolls through search results
        resultsContainer.addEventListener('scroll', () => {
            if (document.activeElement === input) {
                input.blur();
            }
        }, { passive: true });

        function escapeRegex(str) {
            return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        }

        function buildTermRegex(term, isWholeWord) {
            const escaped = escapeRegex(term);
            if (!isWholeWord) {
                return new RegExp(escaped, 'iu');
            }
            return new RegExp('(?<![\\p{L}\\p{N}_])' + escaped + '(?![\\p{L}\\p{N}_])', 'iu');
        }

        function performSearch(query, index) {
            const terms = query.split(/\s+/).filter(Boolean);
            const isWhole = wholeWordsCheckbox ? wholeWordsCheckbox.checked : false;
            const matchRegexes = terms.map(t => buildTermRegex(t, isWhole));

            const matches = [];
            for (let i = 0; i < index.length; i++) {
                const item = index[i];
                let allMatch = true;
                for (const rx of matchRegexes) {
                    if (!rx.test(item.text)) {
                        allMatch = false;
                        break;
                    }
                }
                if (allMatch) {
                    matches.push(item);
                }
            }

            // Save active search state
            setActiveSearch({
                query,
                terms,
                isWhole,
                matches,
                currentIndex: -1
            });

            renderResults(matches, terms, isWhole);
        }

        let currentRenderSession = 0;

        function renderResults(matches, terms, isWhole) {
            currentRenderSession++;
            const sessionId = currentRenderSession;
            resultsContainer.innerHTML = '';

            const countFormatted = matches.length.toLocaleString();
            if (searchCountEl) {
                searchCountEl.textContent = matches.length > 0 ? `${countFormatted} match${matches.length === 1 ? '' : 'es'}` : '';
            }
            if (!matches.length) {
                resultsContainer.innerHTML = '<p style="padding:1rem;color:var(--text-muted);">No matching paragraphs found. Try other keywords or toggle "Whole words".</p>';
                return;
            }

            const rootPrefix = document.body.dataset.root || '';
            const highlightPattern = terms.map(t => {
                const esc = escapeRegex(t);
                return isWhole ? '(?<![\\p{L}\\p{N}_])' + esc + '(?![\\p{L}\\p{N}_])' : esc;
            }).join('|');
            const highlightRegex = new RegExp('(' + highlightPattern + ')', 'giu');

            const CHUNK_SIZE = 60;
            let renderedCount = 0;

            function renderChunk() {
                if (sessionId !== currentRenderSession) return;
                const nextLimit = Math.min(matches.length, renderedCount + CHUNK_SIZE);
                const frag = document.createDocumentFragment();

                for (let idx = renderedCount; idx < nextLimit; idx++) {
                    const m = matches[idx];
                    const a = document.createElement('a');
                    a.className = 'search-item';
                    a.href = rootPrefix + m.path + '#p-' + m.p;
                    a.onclick = (e) => {
                        e.preventDefault();
                        modal.close();
                        navigateToHit(idx);
                    };

                    const header = document.createElement('div');
                    header.className = 'search-item-header';
                    header.innerHTML = `<span>¶ ${m.p}</span><span>${escapeHtml(m.title)}</span>`;

                    // Context-centered snippet
                    const snippet = document.createElement('div');
                    snippet.className = 'search-item-snippet';
                    const snippetInfo = getMatchSnippet(m.text, terms, isWhole);
                    let snippetHTML = (snippetInfo.prefixEllipsis ? '… ' : '') + escapeHtml(snippetInfo.text) + (snippetInfo.suffixEllipsis ? ' …' : '');
                    snippet.innerHTML = snippetHTML.replace(highlightRegex, '<mark>$1</mark>');

                    a.append(header, snippet);
                    frag.append(a);
                }

                // Remove previous sentinel if present
                const oldSentinel = document.getElementById('search-sentinel');
                if (oldSentinel) oldSentinel.remove();

                resultsContainer.append(frag);
                renderedCount = nextLimit;

                if (renderedCount < matches.length) {
                    const remaining = matches.length - renderedCount;
                    const sentinel = document.createElement('div');
                    sentinel.id = 'search-sentinel';
                    sentinel.style.padding = '1rem';
                    sentinel.style.textAlign = 'center';
                    sentinel.innerHTML = `<button type="button" class="btn-runner" style="padding:0.5rem 1.25rem;border-radius:20px;cursor:pointer;">Show more matches (${remaining.toLocaleString()} remaining)</button>`;
                    sentinel.querySelector('button').onclick = () => renderChunk();
                    resultsContainer.append(sentinel);

                    if ('IntersectionObserver' in window) {
                        const observer = new IntersectionObserver((entries) => {
                            if (entries[0].isIntersecting) {
                                observer.disconnect();
                                renderChunk();
                            }
                        }, { root: resultsContainer, rootMargin: '120px' });
                        observer.observe(sentinel);
                    }
                }
            }

            renderChunk();
        }

        // --- On Page Load: Restore Runner Bar and Word Highlighting ---
        const activeSearch = getActiveSearch();
        if (activeSearch && activeSearch.matches && activeSearch.currentIndex >= 0) {
            updateRunnerBar();

            // If page loaded with target hash, highlight the matching words and center match
            if (window.location.hash) {
                const targetP = document.querySelector(window.location.hash);
                if (targetP) {
                    setTimeout(() => {
                        highlightWordsInElement(targetP, activeSearch.terms || activeSearch.query.split(/\s+/), activeSearch.isWhole);
                        scrollToSearchMatch(targetP);
                    }, 180);
                }
            }
        }
    });
})();
