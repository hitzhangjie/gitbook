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
    const contentArea = document.getElementById('main-content');

    if (tocLinks.length === 0 || headings.length === 0 || !contentArea) return;

    function updateActiveTOC() {
        let current = '';
        // Get scroll position of the content area, not the window
        const scrollPos = contentArea.scrollTop + 100;

        headings.forEach((heading) => {
            // Calculate position relative to the content area
            const headingRect = heading.getBoundingClientRect();
            const contentRect = contentArea.getBoundingClientRect();
            const relativeTop = headingRect.top - contentRect.top + contentArea.scrollTop;
            const id = heading.id;
            
            if (relativeTop <= scrollPos) {
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

    // Add scroll event listener to content area instead of window
    contentArea.addEventListener('scroll', updateActiveTOC);
    updateActiveTOC();

    // Smooth scroll for TOC links - scroll within content area
    tocLinks.forEach((link) => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const targetId = this.getAttribute('href').substring(1);
            const target = document.getElementById(targetId);
            if (target && contentArea) {
                // Calculate the position of target relative to content area
                const targetRect = target.getBoundingClientRect();
                const contentRect = contentArea.getBoundingClientRect();
                const offsetTop = targetRect.top - contentRect.top + contentArea.scrollTop;
                
                // Smooth scroll the content area to the target position
                contentArea.scrollTo({
                    top: offsetTop - 20, // Add a small offset for better visibility
                    behavior: 'smooth'
                });
            }
        });
    });
})();

