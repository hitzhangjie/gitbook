// Resizer functionality
(function() {
    let isResizing = false;
    let currentResizer = null;
    let startX = 0;
    let startWidth = 0;

    function initResizer(resizerId, sidebarId, isLeft) {
        const resizer = document.getElementById(resizerId);
        const sidebar = document.getElementById(sidebarId);
        
        if (!resizer || !sidebar) return;

        resizer.addEventListener('mousedown', function(e) {
            isResizing = true;
            currentResizer = resizer;
            startX = e.clientX;
            startWidth = sidebar.offsetWidth;
            document.body.style.cursor = 'col-resize';
            resizer.classList.add('resizing');
            e.preventDefault();
        });
    }

    document.addEventListener('mousemove', function(e) {
        if (!isResizing || !currentResizer) return;

        const sidebar = currentResizer.parentElement;
        const isLeft = sidebar.classList.contains('sidebar-left');
        const deltaX = isLeft ? (e.clientX - startX) : (startX - e.clientX);
        const newWidth = Math.max(150, Math.min(500, startWidth + deltaX));
        
        sidebar.style.width = newWidth + 'px';
    });

    document.addEventListener('mouseup', function() {
        if (isResizing) {
            isResizing = false;
            if (currentResizer) {
                currentResizer.classList.remove('resizing');
            }
            currentResizer = null;
            document.body.style.cursor = '';
        }
    });

    // Initialize resizers
    initResizer('resizer-left', 'sidebar-left', true);
    initResizer('resizer-right', 'sidebar-right', false);
})();

// TOC scroll spy
(function() {
    const tocLinks = document.querySelectorAll('.toc-link');
    const headings = document.querySelectorAll('.article-content h1, .article-content h2, .article-content h3, .article-content h4');

    if (tocLinks.length === 0 || headings.length === 0) return;

    function updateActiveTOC() {
        let current = '';
        const scrollPos = window.scrollY + 100;

        headings.forEach((heading) => {
            const top = heading.offsetTop;
            const id = heading.id;
            if (top <= scrollPos) {
                current = id;
            }
        });

        tocLinks.forEach((link) => {
            link.classList.remove('active');
            if (link.getAttribute('href') === '#' + current) {
                link.classList.add('active');
            }
        });
    }

    // Add scroll event listener
    window.addEventListener('scroll', updateActiveTOC);
    updateActiveTOC();

    // Smooth scroll for TOC links
    tocLinks.forEach((link) => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const targetId = this.getAttribute('href').substring(1);
            const target = document.getElementById(targetId);
            if (target) {
                target.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
        });
    });
})();

