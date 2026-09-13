/**
 * Catechism of the Catholic Church — Simple Reader Application Logic
 */
(() => {
  'use strict';

  const $ = id => document.getElementById(id);
  const paragraphs = [...document.querySelectorAll('p[id^="p-"]')];
  const chapters = [...document.querySelectorAll('.chapter')];
  const bar = document.querySelector('.tools');
  const toc = $('contents');

  // Precompute clean text representation of paragraphs
  const plain = paragraphs.map(p => {
    const copy = p.cloneNode(true);
    copy.querySelectorAll('.note-ref, .paragraph-number, sup').forEach(n => n.remove());
    return copy.textContent.replace(/\s+/g, ' ').trim();
  });
  const text = plain.map(t => t.toLowerCase());

  // Chapter metadata cache
  const chapterInfo = new Map(chapters.map(c => {
    const link = document.querySelector('.toc a[href="#' + c.id + '"]') ||
                 document.querySelector('.toc a[data-aliases~="' + c.id + '"]');
    const group = link?.closest('[data-part]');
    return [c, {
      title: c.querySelector('h2').textContent,
      group,
      part: group?.querySelector('summary [data-toc-title]')?.textContent || 'Catechism',
      link
    }];
  }));

  let matches = [];
  let position = -1;
  let size = 1.65;
  let current = chapters[0];
  let currentHeading = null;
  let currentPara = '';
  let lastFocus = null;
  let words = [];
  let wholeWords = true;

  // Regular expression helper with proper character escaping
  function matchPattern(terms, whole = wholeWords) {
    const escaped = terms
      .map(w => w.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
      .sort((a, b) => b.length - a.length)
      .join('|');
    return new RegExp((whole ? '(?<![\\p{L}\\p{N}_])' : '') + '(' + escaped + ')' + (whole ? '(?![\\p{L}\\p{N}_])' : ''), 'giu');
  }

  // Font size configuration & persistence
  try {
    const n = Number(localStorage.getItem('simple-catechism-size'));
    if (n >= 1.2 && n <= 3.5) size = n;
  } catch {}

  function setSize(n) {
    size = Math.round(Math.max(1.2, Math.min(3.5, n)) * 100) / 100;
    document.documentElement.style.setProperty('--reading-size', size + 'rem');
    try {
      localStorage.setItem('simple-catechism-size', String(size));
    } catch {}
  }

  setSize(size);
  $('smaller').onclick = () => setSize(size - 0.15);
  $('larger').onclick = () => setSize(size + 0.15);

  // Reveal hidden / details parent containers and scroll into view
  function reveal(el) {
    if (!el) return;
    for (let p = el.parentElement; p; p = p.parentElement) {
      if (p.tagName === 'DETAILS') p.open = true;
    }
    el.scrollIntoView({ block: 'start' });
    el.focus({ preventScroll: true });
  }

  function hash() {
    if (location.hash) {
      try {
        reveal($(decodeURIComponent(location.hash.slice(1))));
      } catch {}
    }
  }

  window.addEventListener('hashchange', hash);

  function choose(i) {
    if (position >= 0) matches[position]?.el.classList.remove('hit');
    position = i;
    const target = matches[i].el;
    target.classList.add('hit');
    if ($('results').open) $('results').close();
    location.hash = target.id;
    reveal(target);
    $('status').textContent = (i + 1) + ' of ' + matches.length + ' matching paragraphs';
  }

  function show(step) {
    if (matches.length) {
      choose((position + step + matches.length) % matches.length);
    }
  }

  function snippet(value) {
    if (!words.length) return value.slice(0, 240);
    const at = matchPattern(words).exec(value)?.index ?? Infinity;
    if (!Number.isFinite(at)) return value.slice(0, 240);
    let start = value.lastIndexOf('. ', at);
    start = start < 0 ? 0 : start + 2;
    let end = value.indexOf('. ', at);
    end = end < 0 ? value.length : end + 1;
    if (at - start > 110) start = at - 90;
    if (end - start > 300) end = Math.max(at + (words[0]?.length || 5) + 100, start + 260);
    end = Math.min(end, value.length);
    return (start ? '…' : '') + value.slice(start, end) + (end < value.length ? '…' : '');
  }

  function highlighted(el, value) {
    if (!words.length) {
      el.textContent = value;
      return;
    }
    let end = 0;
    for (const m of value.matchAll(matchPattern(words))) {
      el.append(document.createTextNode(value.slice(end, m.index)));
      const mark = document.createElement('mark');
      mark.textContent = m[0];
      el.append(mark);
      end = m.index + m[0].length;
    }
    el.append(document.createTextNode(value.slice(end)));
  }

  function results() {
    const root = $('result-list');
    const nav = $('result-groups');
    root.replaceChildren();
    nav.replaceChildren();
    $('result-title').textContent = 'Find: ' + $('query').value.trim();
    $('result-count').textContent = matches.length + ' matching paragraphs';

    const groups = new Map();
    matches.forEach((item, matchIdx) => {
      const info = chapterInfo.get(item.el.closest('.chapter'));
      if (!groups.has(info.part)) groups.set(info.part, []);
      groups.get(info.part).push({ el: item.el, matchIdx, pIndex: item.pIndex, info });
    });

    let g = 0;
    for (const [part, items] of groups) {
      const section = document.createElement('section');
      section.id = 'result-group-' + g++;
      const heading = document.createElement('h3');
      heading.textContent = part + ' (' + items.length + ')';
      section.append(heading);

      const jump = document.createElement('a');
      jump.href = '#' + section.id;
      jump.textContent = part + ' · ' + items.length;
      jump.onclick = e => {
        e.preventDefault();
        section.scrollIntoView({ block: 'start' });
      };
      nav.append(jump);

      let last = '';
      for (const { el, matchIdx, pIndex, info } of items) {
        if (info.title !== last) {
          const h = document.createElement('h4');
          h.textContent = info.title;
          section.append(h);
          last = info.title;
        }
        const button = document.createElement('button');
        button.className = 'result-item';
        const label = document.createElement('strong');
        label.textContent = '¶ ' + el.id.slice(2);
        const sentence = document.createElement('span');
        highlighted(sentence, snippet(plain[pIndex]));
        button.append(label, sentence);
        button.onclick = () => choose(matchIdx);
        section.append(button);
      }
      root.append(section);
    }

    if (!matches.length) {
      root.textContent = words.length ? 'No matches. Try another word.' : 'Enter a word to find.';
    }
    if (!$('results').open) $('results').showModal();
  }

  // Precompile patterns once per search for high performance
  function runFind() {
    matches.forEach(m => m.el.classList.remove('hit'));
    position = -1;
    words = [...new Set($('query').value.toLowerCase().match(/[\p{L}\p{N}]+/gu) || [])];

    if (!words.length) {
      matches = [];
    } else {
      const patterns = words.map(w => matchPattern([w]));
      matches = [];
      for (let i = 0; i < paragraphs.length; i++) {
        const t = text[i];
        let allMatch = true;
        for (const pat of patterns) {
          pat.lastIndex = 0;
          if (!pat.test(t)) {
            allMatch = false;
            break;
          }
        }
        if (allMatch) {
          matches.push({ el: paragraphs[i], pIndex: i });
        }
      }
    }

    $('previous').disabled = $('next').disabled = $('show-results').disabled = !matches.length;
    $('status').textContent = matches.length + ' matching paragraphs';
    results();
  }

  $('find').onsubmit = e => {
    e.preventDefault();
    runFind();
  };

  for (const id of ['whole-words', 'result-whole-words']) {
    $(id).onchange = e => {
      wholeWords = e.target.checked;
      $('whole-words').checked = $('result-whole-words').checked = wholeWords;
      if ($('query').value.trim()) runFind();
    };
  }

  $('show-results').onclick = () => {
    if (!$('results').open) $('results').showModal();
  };
  $('close-results').onclick = () => $('results').close();

  // Backdrop click dismissal for Find Results dialog
  $('results').addEventListener('click', e => {
    if (e.target === $('results')) {
      const r = $('results').getBoundingClientRect();
      if (e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom) {
        $('results').close();
      }
    }
  });

  $('next').onclick = () => show(1);
  $('previous').onclick = () => show(-1);

  // Jump to paragraph handler
  $('jump').onsubmit = e => {
    e.preventDefault();
    const n = Number($('paragraph').value);
    if (!Number.isInteger(n) || n < 1 || n > 2865) {
      $('jump-status').textContent = 'Enter a number from 1 to 2865.';
      return;
    }
    // Changing hash triggers hashchange which calls reveal()
    location.hash = 'p-' + n;
    $('jump-status').textContent = 'Paragraph ' + n;
  };

  // Table of Contents toggle and navigation
  function openContents(button, sectionOnly) {
    lastFocus = button;
    const info = chapterInfo.get(current);
    const selected = (currentHeading && (toc.querySelector('a[href="#' + currentHeading.id + '"]') ||
                     toc.querySelector('a[data-aliases~="' + currentHeading.id + '"]'))) || info.link;

    toc.querySelectorAll('[aria-current]').forEach(a => {
      a.removeAttribute('aria-current');
      a.querySelector('.you-are-here')?.remove();
    });

    if (selected) {
      selected.setAttribute('aria-current', 'location');
      const badge = document.createElement('span');
      badge.className = 'you-are-here';
      badge.textContent = 'You are here' + (currentPara ? ' · ¶ ' + currentPara : '');
      selected.append(badge);
    }

    toc.open = !toc.open;
    if (toc.open) {
      toc.querySelectorAll('nav details').forEach(d => d.open = false);
      if (sectionOnly && selected) {
        for (let ancestor = selected.closest('summary')?.parentElement?.parentElement || selected.parentElement;
             ancestor && ancestor !== toc; ancestor = ancestor.parentElement) {
          if (ancestor.tagName === 'DETAILS') ancestor.open = true;
        }
      }
      requestAnimationFrame(() => {
        toc.scrollTop = 0;
        if (sectionOnly && selected) {
          selected.scrollIntoView({ block: 'center' });
          selected.focus({ preventScroll: true });
        }
      });
    }
    updateExpanded();
  }

  function updateExpanded() {
    $('location-link').setAttribute('aria-expanded', String(toc.open));
  }

  $('location-link').onclick = () => openContents($('location-link'), true);
  toc.addEventListener('toggle', updateExpanded);

  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && toc.open) {
      toc.open = false;
      lastFocus?.focus();
    }
  });

  document.addEventListener('click', e => {
    if (toc.open && !toc.contains(e.target) && !$('location').contains(e.target)) {
      toc.open = false;
    }
  });

  toc.querySelectorAll('nav a').forEach(a => a.addEventListener('click', e => {
    e.preventDefault();
    e.stopPropagation();
    const id = a.getAttribute('href').slice(1);
    const target = $(id);
    if (!target) return;
    toc.open = false;
    updateExpanded();
    target.setAttribute('tabindex', '-1');
    location.hash = id;
    requestAnimationFrame(() => {
      reveal(target);
      locate();
    });
  }));

  // Dynamic bar height tracking
  new ResizeObserver(() => document.documentElement.style.setProperty('--bar-height', bar.offsetHeight + 'px')).observe(bar);

  // High-performance location tracker using binary search across chapters
  let scheduled = false;
  function locate() {
    scheduled = false;
    const y = bar.getBoundingClientRect().bottom + 25;

    // Binary search over 442 chapters instead of sequential scan
    let low = 0;
    let high = chapters.length - 1;
    let best = chapters[0];

    while (low <= high) {
      const mid = (low + high) >> 1;
      if (chapters[mid].getBoundingClientRect().top <= y) {
        best = chapters[mid];
        low = mid + 1;
      } else {
        high = mid - 1;
      }
    }
    current = best;

    const info = chapterInfo.get(current);
    let activeHeading = null;
    for (const h of current.querySelectorAll('[data-subheading]')) {
      if (h.getBoundingClientRect().top > y + 10) break;
      activeHeading = h;
    }

    let para = '';
    for (const p of current.querySelectorAll('p[id^="p-"]')) {
      if (p.getBoundingClientRect().top > y + 80) break;
      para = p.id.slice(2);
    }

    currentHeading = activeHeading;
    currentPara = para;
    const trail = [info.part, activeHeading ? activeHeading.textContent : info.title, para ? '¶ ' + para : ''].filter(Boolean).join(' › ');
    $('location-link').textContent = trail;
    $('location-link').title = trail + ' — open contents';
  }

  window.addEventListener('scroll', () => {
    if (!scheduled) {
      scheduled = true;
      requestAnimationFrame(locate);
    }
  }, { passive: true });

  window.addEventListener('resize', locate);
  hash();
  locate();
})();

/**
 * Footnotes Glossary & Abbreviations Logic
 */
(() => {
  'use strict';

  const dialog = document.getElementById('reference-dialog');
  const query = document.getElementById('reference-query');
  const root = document.getElementById('reference-entries');
  const refDataEl = document.getElementById('reference-data');
  if (!dialog || !query || !root || !refDataEl) return;

  const data = JSON.parse(refDataEl.textContent);

  // Pre-sort definitions
  const allDefs = Object.entries(data.definitions).sort(([a], [b]) => a.localeCompare(b));

  function render() {
    const q = query.value.trim().toLowerCase();
    root.replaceChildren();

    const defs = q
      ? allDefs.filter(([k, v]) => (k + ' ' + v).toLowerCase().includes(q))
      : allDefs;

    for (const [key, value] of defs) {
      const row = document.createElement('div');
      row.className = 'reference-entry';
      const title = document.createElement('strong');
      title.textContent = key;
      const description = document.createElement('p');
      description.textContent = value;
      row.append(title, description);
      root.append(row);
    }

    const notes = q ? data.notes.filter(n => n.text.toLowerCase().includes(q)) : [];
    document.getElementById('reference-count').textContent =
      defs.length + ' glossary entries' +
      (q ? ' · ' + notes.length + ' footnotes' : ' · ' + data.notes.length + ' footnotes searchable');

    if (notes.length) {
      const title = document.createElement('h3');
      title.textContent = 'Matching footnotes';
      root.append(title);

      const maxNotes = 100;
      const visibleNotes = notes.slice(0, maxNotes);

      for (const n of visibleNotes) {
        const a = document.createElement('a');
        a.className = 'reference-note';
        a.href = '#' + n.id;
        a.textContent = n.text;
        a.onclick = () => {
          dialog.close();
          const target = document.getElementById(n.id);
          if (!target) return;
          for (let p = target.parentElement; p; p = p.parentElement) {
            if (p.tagName === 'DETAILS') p.open = true;
          }
          requestAnimationFrame(() => target.scrollIntoView({ block: 'start' }));
        };
        root.append(a);
      }

      if (notes.length > maxNotes) {
        const moreNotice = document.createElement('p');
        moreNotice.style.fontStyle = 'italic';
        moreNotice.style.color = '#526677';
        moreNotice.textContent = `Showing first ${maxNotes} of ${notes.length} matching footnotes. Type more characters to narrow down.`;
        root.append(moreNotice);
      }
    }

    if (!defs.length && !notes.length) {
      root.textContent = 'No matching abbreviation or footnote. Try fewer words.';
    }
  }

  // Debounced input search to avoid UI freeze on short queries
  let debounceTimer;
  query.oninput = () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(render, 150);
  };

  document.querySelectorAll('.open-references').forEach(b => {
    b.onclick = () => {
      query.value = '';
      render();
      dialog.showModal();
      query.focus();
    };
  });

  document.getElementById('close-references').onclick = () => dialog.close();

  // Backdrop click dismissal for Reference Dialog
  dialog.addEventListener('click', e => {
    if (e.target === dialog) {
      const r = dialog.getBoundingClientRect();
      if (e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom) {
        dialog.close();
      }
    }
  });

  // Helper to explain terms in footnote text
  function explainTerms(text) {
    return Object.entries(data.definitions)
      .filter(([key]) => {
        const escaped = key.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        return new RegExp('(?<![\\p{L}\\p{N}])' + escaped + '(?![\\p{L}\\p{N}])', 'u').test(text);
      })
      .sort(([a], [b]) => b.length - a.length)
      .filter(([key], i, all) => !all.slice(0, i).some(([long]) => long.includes(key)));
  }

  // Lazy Footnote Explanations: initialize per section when details opens
  const notesById = new Map(data.notes.map(n => [n.id, n]));

  function initFootnotesSection(section) {
    if (section.dataset.initialized) return;
    section.dataset.initialized = 'true';

    for (const p of section.querySelectorAll('p[id]')) {
      const note = notesById.get(p.id);
      if (!note) continue;

      const button = document.createElement('button');
      button.className = 'explain-note';
      button.textContent = 'Explain';
      button.setAttribute('aria-expanded', 'false');
      button.setAttribute('aria-controls', note.id + '-explanation');
      p.append(button);

      let panel;
      button.onclick = () => {
        if (!panel) {
          panel = document.createElement('div');
          panel.id = note.id + '-explanation';
          panel.className = 'note-explanation';
          panel.setAttribute('role', 'region');
          panel.setAttribute('aria-label', 'Explanation of footnote ' + note.text.split('.')[0]);

          const heading = document.createElement('strong');
          heading.textContent = 'About this reference';
          panel.append(heading);

          const terms = explainTerms(note.text);
          for (const [key, value] of terms) {
            const row = document.createElement('p');
            const label = document.createElement('b');
            label.textContent = key + ': ';
            row.append(label, document.createTextNode(value));
            panel.append(row);
          }

          if (!terms.length) {
            const message = document.createElement('p');
            message.textContent = 'No verified expansion is available for this reference yet. The citation above is preserved as printed.';
            panel.append(message);
          }

          const sources = [...p.querySelectorAll('a[href^="https:"]')];
          if (sources.length) {
            const label = document.createElement('p');
            label.textContent = 'Read the source (opens in a new tab):';
            panel.append(label);
            for (const source of sources) {
              const copy = source.cloneNode(true);
              copy.className = 'explained-source';
              panel.append(copy);
            }
          }

          const browse = document.createElement('button');
          browse.className = 'browse-references';
          browse.textContent = 'Browse all abbreviations';
          browse.onclick = () => {
            query.value = '';
            render();
            dialog.showModal();
            query.focus();
          };
          panel.append(browse);
          p.after(panel);
        } else {
          panel.hidden = !panel.hidden;
        }

        const open = !panel.hidden;
        button.setAttribute('aria-expanded', String(open));
        button.textContent = open ? 'Hide explanation' : 'Explain';
      };
    }
  }

  document.querySelectorAll('details.footnotes').forEach(d => {
    d.addEventListener('toggle', () => {
      if (d.open) initFootnotesSection(d);
    });
    if (d.open) initFootnotesSection(d);
  });
})();

