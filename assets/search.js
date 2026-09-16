/**
 * Catechism Reader - Client Search Modal
 */
(() => {
    'use strict';

    let searchIndex = null;
    let isFetching = false;

    document.addEventListener('DOMContentLoaded', () => {
        const modal = document.getElementById('search-modal');
        const input = document.getElementById('search-query');
        const resultsContainer = document.getElementById('search-results');
        const openBtn = document.getElementById('btn-open-search');
        const closeBtn = document.getElementById('btn-close-search');

        if (!modal || !input || !resultsContainer) return;

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
                .catch(err => {
                    isFetching = false;
                    resultsContainer.innerHTML = '<p style="padding:1rem;color:red;">Error loading search index.</p>';
                });
        }

        function openSearch() {
            modal.showModal();
            input.value = '';
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
        input.addEventListener('input', () => {
            clearTimeout(debounceTimer);
            const query = input.value.trim().toLowerCase();
            if (!query || query.length < 2) {
                resultsContainer.innerHTML = '<p style="padding:1rem;color:var(--text-muted);">Type at least 2 characters to search...</p>';
                return;
            }
            debounceTimer = setTimeout(() => {
                ensureIndex((index) => {
                    performSearch(query, index);
                });
            }, 150);
        });

        function performSearch(query, index) {
            const terms = query.split(/\s+/).filter(Boolean);
            const matches = [];

            for (let i = 0; i < index.length; i++) {
                const item = index[i];
                const text = item.text.toLowerCase();
                let allMatch = true;
                for (const t of terms) {
                    if (!text.includes(t)) {
                        allMatch = false;
                        break;
                    }
                }
                if (allMatch) {
                    matches.push(item);
                    if (matches.length >= 80) break; // Limit to 80 matches for performance
                }
            }

            renderResults(matches, terms);
        }

        function renderResults(matches, terms) {
            resultsContainer.innerHTML = '';
            if (!matches.length) {
                resultsContainer.innerHTML = '<p style="padding:1rem;color:var(--text-muted);">No matching paragraphs found. Try other keywords.</p>';
                return;
            }

            const rootPrefix = document.body.dataset.root || '';
            const regex = new RegExp('(' + terms.map(t => t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|') + ')', 'gi');

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
                let highlighted = m.text.slice(0, 220);
                if (m.text.length > 220) highlighted += '…';
                snippet.innerHTML = highlighted.replace(regex, '<mark>$1</mark>');

                a.append(header, snippet);
                frag.append(a);
            }

            const countP = document.createElement('p');
            countP.style.padding = '0.5rem 0.8rem';
            countP.style.fontSize = '0.85rem';
            countP.style.color = 'var(--text-muted)';
            countP.textContent = `${matches.length} matches found`;
            resultsContainer.append(countP, frag);
        }
    });
})();

