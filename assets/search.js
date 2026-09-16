/**
 * Catechism Reader - Client Search Modal
 */
(() => {
    'use strict';

    let searchIndex = null;
    let isFetching = false;
    const WHOLE_WORDS_KEY = 'ccc-search-whole-words';

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
                    if (matches.length >= 80) break; // Limit to 80 matches for responsiveness
                }
            }

            renderResults(matches, terms, isWhole);
        }

        function renderResults(matches, terms, isWhole) {
            resultsContainer.innerHTML = '';
            if (searchCountEl) {
                searchCountEl.textContent = matches.length > 0 ? `${matches.length}${matches.length >= 80 ? '+' : ''} matches` : '';
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

            const frag = document.createDocumentFragment();
            for (const m of matches) {
                const a = document.createElement('a');
                a.className = 'search-item';
                a.href = rootPrefix + m.path + '#p-' + m.p;
                a.onclick = () => modal.close();

                const header = document.createElement('div');
                header.className = 'search-item-header';
                header.innerHTML = `<span>¶ ${m.p}</span><span>${m.title}</span>`;

                // Highlight snippet
                const snippet = document.createElement('div');
                snippet.className = 'search-item-snippet';
                let highlighted = m.text.slice(0, 240);
                if (m.text.length > 240) highlighted += '…';
                snippet.innerHTML = highlighted.replace(highlightRegex, '<mark>$1</mark>');

                a.append(header, snippet);
                frag.append(a);
            }

            resultsContainer.append(frag);
        }
    });
})();
