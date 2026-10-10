// Full-text search over the index generated at build time.
(function() {
    const input = document.getElementById('search-input');
    const resultsEl = document.getElementById('search-results');
    const nav = document.getElementById('nav-tree');
    const hint = document.getElementById('search-kbd');
    if (!input || !resultsEl || !nav) return;

    let index = null;
    let loadError = false;
    let hits = [];
    let active = -1;
    let requestID = 0;

    function normalize(value) {
        return String(value || '').toLowerCase().replace(/\s+/g, ' ').trim();
    }

    function termsOf(query) {
        return normalize(query).split(' ').filter(Boolean);
    }

    async function loadIndex() {
        const current = ++requestID;
        loadError = false;
        try {
            const response = await fetch('/static/search-index.json', { cache: 'no-store' });
            if (!response.ok) {
                throw new Error('status ' + response.status);
            }
            const data = await response.json();
            if (current !== requestID) return;
            index = data && Array.isArray(data.pages) ? data : { pages: [] };
        } catch (e) {
            if (current !== requestID) return;
            index = null;
            loadError = true;
        }
        if (input.value.trim()) {
            render();
        }
    }

    window.gitbookReloadSearchIndex = loadIndex;

    function sectionMatches(section, terms) {
        const blob = normalize((section.heading || '') + '\n' + (section.text || ''));
        return terms.every(function(term) {
            return blob.indexOf(term) !== -1;
        });
    }

    function scoreHit(page, section, terms) {
        const title = normalize(page.title);
        const heading = normalize(section.heading);
        const text = normalize(section.text);
        let score = 1;
        terms.forEach(function(term) {
            if (title.indexOf(term) !== -1) score += 5;
            if (heading.indexOf(term) !== -1) score += 3;
            const idx = text.indexOf(term);
            if (idx !== -1) score += 2 + Math.max(0, 1 - idx / 400);
        });
        return score;
    }

    function search(query) {
        const terms = termsOf(query);
        if (!terms.length || !index) return [];

        const grouped = new Map();
        (index.pages || []).forEach(function(page) {
            const pageHits = [];
            (page.sections || []).forEach(function(section) {
                if (!sectionMatches(section, terms)) return;
                pageHits.push({
                    page: page,
                    section: section,
                    score: scoreHit(page, section, terms)
                });
            });
            const titleBlob = normalize(page.title);
            if (pageHits.length === 0 && terms.every(function(term) { return titleBlob.indexOf(term) !== -1; })) {
                const section = (page.sections && page.sections[0]) || { heading: '', text: '', anchor: '' };
                pageHits.push({
                    page: page,
                    section: section,
                    score: 6,
                    titleOnly: true
                });
            }
            if (!pageHits.length) return;
            pageHits.sort(function(a, b) { return b.score - a.score; });
            grouped.set(page.url, pageHits.slice(0, 3));
        });

        const list = [];
        grouped.forEach(function(pageHits) {
            pageHits.forEach(function(hit) { list.push(hit); });
        });
        list.sort(function(a, b) {
            if (b.score !== a.score) return b.score - a.score;
            return String(a.page.title).localeCompare(String(b.page.title), 'zh');
        });
        return list.slice(0, 30);
    }

    function snippetSource(hit, terms) {
        const text = hit.section.text || '';
        const heading = hit.section.heading || '';
        const title = hit.page.title || '';
        if (terms.some(function(term) { return text.toLowerCase().indexOf(term) !== -1; })) return text;
        if (terms.some(function(term) { return heading.toLowerCase().indexOf(term) !== -1; })) return heading;
        if (terms.some(function(term) { return title.toLowerCase().indexOf(term) !== -1; })) return title;
        return text || heading || title;
    }

    function excerpt(text, terms) {
        const source = String(text || '').replace(/\s+/g, ' ').trim();
        if (!source) return '';
        const lower = source.toLowerCase();
        let idx = -1;
        let matched = '';
        terms.forEach(function(term) {
            const at = lower.indexOf(term);
            if (at !== -1 && (idx < 0 || at < idx)) {
                idx = at;
                matched = term;
            }
        });
        if (idx < 0) return source.slice(0, 110);
        const start = Math.max(0, idx - 28);
        const end = Math.min(source.length, idx + matched.length + 72);
        return (start > 0 ? '…' : '') + source.slice(start, end) + (end < source.length ? '…' : '');
    }

    function highlightInto(parent, text, terms) {
        const lower = text.toLowerCase();
        const ranges = [];
        terms.forEach(function(term) {
            if (!term) return;
            let from = 0;
            while (from < lower.length) {
                const at = lower.indexOf(term, from);
                if (at < 0) break;
                ranges.push([at, at + term.length]);
                from = at + term.length;
            }
        });
        ranges.sort(function(a, b) { return a[0] - b[0] || b[1] - a[1]; });
        const merged = [];
        ranges.forEach(function(range) {
            const last = merged[merged.length - 1];
            if (!last || range[0] > last[1]) {
                merged.push([range[0], range[1]]);
            } else {
                last[1] = Math.max(last[1], range[1]);
            }
        });
        let cursor = 0;
        merged.forEach(function(range) {
            if (range[0] > cursor) {
                parent.appendChild(document.createTextNode(text.slice(cursor, range[0])));
            }
            const mark = document.createElement('mark');
            mark.textContent = text.slice(range[0], range[1]);
            parent.appendChild(mark);
            cursor = range[1];
        });
        if (cursor < text.length) {
            parent.appendChild(document.createTextNode(text.slice(cursor)));
        }
    }

    function hitURL(hit) {
        const anchor = hit.titleOnly ? '' : (hit.section.anchor || '');
        if (!anchor) return hit.page.url;
        return hit.page.url + '#' + encodeURIComponent(anchor);
    }

    function setActive(next) {
        const buttons = resultsEl.querySelectorAll('.search-result');
        if (!buttons.length) {
            active = -1;
            return;
        }
        if (next < 0) next = buttons.length - 1;
        if (next >= buttons.length) next = 0;
        active = next;
        buttons.forEach(function(button, i) {
            button.classList.toggle('is-active', i === active);
        });
        if (buttons[active].scrollIntoView) {
            buttons[active].scrollIntoView({ block: 'nearest' });
        }
    }

    function openHit(hit) {
        const url = hitURL(hit);
        if (typeof window.gitbookLoadPage === 'function') {
            window.gitbookLoadPage(url);
            return;
        }
        window.location.href = url;
    }

    function syncHint() {
        if (!hint) return;
        hint.hidden = document.activeElement === input || input.value.length > 0;
    }

    function render() {
        const terms = termsOf(input.value);
        resultsEl.replaceChildren();
        active = -1;
        syncHint();
        if (!terms.length) {
            resultsEl.hidden = true;
            nav.hidden = false;
            hits = [];
            return;
        }

        nav.hidden = true;
        resultsEl.hidden = false;
        if (!index) {
            const note = document.createElement('p');
            note.className = 'search-empty';
            note.textContent = loadError ? '无法加载搜索索引' : '正在加载索引…';
            resultsEl.appendChild(note);
            return;
        }

        hits = search(input.value);
        const status = document.createElement('p');
        status.className = 'search-status';
        status.textContent = hits.length ? (hits.length + ' 个结果') : '没有找到相关内容';
        resultsEl.appendChild(status);

        hits.forEach(function(hit, i) {
            const button = document.createElement('button');
            button.type = 'button';
            button.className = 'search-result';

            const title = document.createElement('div');
            title.className = 'search-result-title';
            title.textContent = hit.page.title || hit.page.url;
            button.appendChild(title);

            const heading = hit.section.heading || '';
            if (!hit.titleOnly && heading && heading !== hit.page.title) {
                const sub = document.createElement('div');
                sub.className = 'search-result-heading';
                sub.textContent = heading;
                button.appendChild(sub);
            }

            const snippet = excerpt(snippetSource(hit, terms), terms);
            if (snippet) {
                const body = document.createElement('div');
                body.className = 'search-result-snippet';
                highlightInto(body, snippet, terms);
                button.appendChild(body);
            }

            button.addEventListener('click', function() {
                openHit(hit);
            });
            button.addEventListener('mouseenter', function() {
                setActive(i);
            });
            resultsEl.appendChild(button);
        });
    }

    input.addEventListener('input', render);
    input.addEventListener('focus', syncHint);
    input.addEventListener('blur', syncHint);
    input.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            if (input.value) {
                input.value = '';
                render();
            } else {
                input.blur();
            }
            e.preventDefault();
            e.stopPropagation();
            return;
        }
        if (!hits.length || resultsEl.hidden) return;
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            setActive(active + 1);
            return;
        }
        if (e.key === 'ArrowUp') {
            e.preventDefault();
            setActive(active < 0 ? hits.length - 1 : active - 1);
            return;
        }
        if (e.key === 'Enter') {
            e.preventDefault();
            openHit(hits[active >= 0 ? active : 0]);
        }
    });

    document.addEventListener('keydown', function(e) {
        const tag = (e.target && e.target.tagName) || '';
        const typing = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || (e.target && e.target.isContentEditable);
        const slash = e.key === '/' && !typing && !e.ctrlKey && !e.metaKey && !e.altKey;
        const find = (e.ctrlKey || e.metaKey) && (e.key === 'k' || e.key === 'K');
        if (!slash && !find) return;
        e.preventDefault();
        input.focus();
        input.select();
    });

    loadIndex();
})();
