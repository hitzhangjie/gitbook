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

// Live Reload functionality
(function() {
    // Check if we're in development mode (has WebSocket support)
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${window.location.host}/ws`;
    
    let ws = null;
    let reconnectTimer = null;
    let isReloading = false;
    let reloadDebounceTimer = null;

    // Create loading indicator
    function createLoadingIndicator() {
        let indicator = document.getElementById('live-reload-indicator');
        if (!indicator) {
            indicator = document.createElement('div');
            indicator.id = 'live-reload-indicator';
            indicator.style.cssText = `
                position: fixed;
                top: 20px;
                right: 20px;
                background: #4CAF50;
                color: white;
                padding: 10px 20px;
                border-radius: 4px;
                box-shadow: 0 2px 8px rgba(0,0,0,0.2);
                z-index: 10000;
                font-size: 14px;
                display: none;
                transition: opacity 0.3s;
            `;
            document.body.appendChild(indicator);
        }
        return indicator;
    }

    function showIndicator(message, type = 'info') {
        const indicator = createLoadingIndicator();
        indicator.textContent = message;
        indicator.style.display = 'block';
        indicator.style.background = type === 'error' ? '#f44336' : type === 'success' ? '#4CAF50' : '#2196F3';
        indicator.style.opacity = '1';
    }

    function hideIndicator() {
        const indicator = document.getElementById('live-reload-indicator');
        if (indicator) {
            indicator.style.opacity = '0';
            setTimeout(() => {
                indicator.style.display = 'none';
            }, 300);
        }
    }

    // Connect to WebSocket
    function connect() {
        try {
            ws = new WebSocket(wsUrl);
            
            ws.onopen = function() {
                console.log('Live reload connected');
                hideIndicator();
            };

            ws.onmessage = function(event) {
                try {
                    const message = JSON.parse(event.data);
                    handleMessage(message);
                } catch (e) {
                    console.error('Failed to parse message:', e);
                }
            };

            ws.onerror = function(error) {
                console.error('WebSocket error:', error);
            };

            ws.onclose = function() {
                console.log('Live reload disconnected, reconnecting...');
                ws = null;
                // Reconnect after 1 second
                reconnectTimer = setTimeout(connect, 1000);
            };
        } catch (e) {
            // WebSocket not available, silently fail
            console.log('WebSocket not available, live reload disabled');
        }
    }

    // Handle WebSocket messages
    function handleMessage(message) {
        switch (message.type) {
            case 'connected':
                console.log('Live reload ready');
                break;
            case 'rebuild_start':
                showIndicator('正在重建...', 'info');
                break;
            case 'rebuild_complete':
                showIndicator('重建完成', 'success');
                // Debounce reload to avoid multiple rapid reloads
                if (reloadDebounceTimer) {
                    clearTimeout(reloadDebounceTimer);
                }
                reloadDebounceTimer = setTimeout(() => {
                    reloadPage(message.path);
                }, 100);
                break;
            case 'rebuild_error':
                showIndicator('重建失败: ' + message.message, 'error');
                setTimeout(hideIndicator, 3000);
                break;
        }
    }

    // Reload page content (partial reload)
    async function reloadPage(changedPath) {
        if (isReloading) return;
        isReloading = true;

        try {
            const contentArea = document.getElementById('main-content');
            if (!contentArea) {
                // Fallback to full reload
                window.location.reload();
                return;
            }

            // Save current state
            const scrollTop = contentArea.scrollTop;
            const currentUrl = window.location.href;

            // Fetch new page content
            const response = await fetch(currentUrl, {
                headers: {
                    'Cache-Control': 'no-cache',
                    'Pragma': 'no-cache'
                }
            });

            if (!response.ok) {
                throw new Error('Failed to fetch new content');
            }

            const newHTML = await response.text();

            // Parse new HTML using DOMParser to handle full HTML documents
            const parser = new DOMParser();
            const newDoc = parser.parseFromString(newHTML, 'text/html');

            // Extract elements to update from the parsed document
            const newTitle = newDoc.querySelector('.article-title');
            const newContent = newDoc.querySelector('.article-content');
            const newTOC = newDoc.querySelector('#toc');
            const newNavTree = newDoc.querySelector('.nav-tree');

            // Debug: log if elements are found
            if (!newContent) {
                console.warn('New content element not found, falling back to full reload');
                window.location.reload();
                return;
            }

            // Update title
            const oldTitle = document.querySelector('.article-title');
            if (oldTitle && newTitle) {
                oldTitle.textContent = newTitle.textContent;
            }

            // Update content with fade animation
            const oldContent = document.querySelector('.article-content');
            if (oldContent && newContent) {
                // Store reference to avoid closure issues
                const contentHTML = newContent.innerHTML;
                
                // Fade out
                oldContent.style.transition = 'opacity 0.2s';
                oldContent.style.opacity = '0';

                setTimeout(() => {
                    // Check if oldContent still exists (might have been removed)
                    const currentContent = document.querySelector('.article-content');
                    if (!currentContent) {
                        console.error('Content element removed during update');
                        window.location.reload();
                        return;
                    }

                    // Replace content
                    currentContent.innerHTML = contentHTML;
                    
                    // Force reflow to ensure transition works
                    currentContent.offsetHeight;
                    
                    // Fade in
                    currentContent.style.opacity = '1';

                    // Update TOC after content is updated
                    const oldTOC = document.getElementById('toc');
                    if (oldTOC && newTOC) {
                        oldTOC.innerHTML = newTOC.innerHTML;
                        // Reinitialize TOC scroll spy
                        initTOCScrollSpy();
                    }

                    // Update navigation if needed (only if structure changed)
                    const oldNavTree = document.querySelector('.nav-tree');
                    if (oldNavTree && newNavTree) {
                        // Check if navigation structure changed
                        if (oldNavTree.innerHTML !== newNavTree.innerHTML) {
                            oldNavTree.innerHTML = newNavTree.innerHTML;
                        }
                    }
                }, 200);
            } else {
                console.warn('Content update failed:', { oldContent: !!oldContent, newContent: !!newContent });
                // Fallback to full reload if content elements not found
                window.location.reload();
                return;
            }

            // Restore scroll position
            setTimeout(() => {
                contentArea.scrollTop = scrollTop;
                isReloading = false;
                hideIndicator();
            }, 300);

        } catch (error) {
            console.error('Failed to reload page:', error);
            // Fallback to full reload
            window.location.reload();
        }
    }

    // Reinitialize TOC scroll spy after content update
    function initTOCScrollSpy() {
        // Wait a bit for DOM to update
        setTimeout(() => {
            const tocLinks = document.querySelectorAll('.toc-link');
            const headings = document.querySelectorAll('.article-content h1, .article-content h2, .article-content h3, .article-content h4');
            const contentArea = document.getElementById('main-content');

            if (tocLinks.length === 0 || headings.length === 0 || !contentArea) return;

            // Remove existing scroll listener by cloning the element (removes all listeners)
            const newContentArea = contentArea.cloneNode(true);
            contentArea.parentNode.replaceChild(newContentArea, contentArea);

            // Re-query elements after DOM update
            const newTocLinks = document.querySelectorAll('.toc-link');
            const newHeadings = document.querySelectorAll('.article-content h1, .article-content h2, .article-content h3, .article-content h4');

            function updateActiveTOC() {
                let current = '';
                const scrollPos = newContentArea.scrollTop + 100;

                newHeadings.forEach((heading) => {
                    const headingRect = heading.getBoundingClientRect();
                    const contentRect = newContentArea.getBoundingClientRect();
                    const relativeTop = headingRect.top - contentRect.top + newContentArea.scrollTop;
                    const id = heading.id;
                    
                    if (relativeTop <= scrollPos) {
                        current = id;
                    }
                });

                newTocLinks.forEach((link) => {
                    link.classList.remove('active');
                    if (link.getAttribute('href') === '#' + current) {
                        link.classList.add('active');
                    }
                });
            }

            newContentArea.addEventListener('scroll', updateActiveTOC);
            updateActiveTOC();

            // Reinitialize smooth scroll for TOC links
            newTocLinks.forEach((link) => {
                link.addEventListener('click', function(e) {
                    e.preventDefault();
                    const targetId = this.getAttribute('href').substring(1);
                    const target = document.getElementById(targetId);
                    if (target && newContentArea) {
                        const targetRect = target.getBoundingClientRect();
                        const contentRect = newContentArea.getBoundingClientRect();
                        const offsetTop = targetRect.top - contentRect.top + newContentArea.scrollTop;
                        
                        newContentArea.scrollTo({
                            top: offsetTop - 20,
                            behavior: 'smooth'
                        });
                    }
                });
            });
        }, 100);
    }

    // Initialize connection
    connect();

    // Cleanup on page unload
    window.addEventListener('beforeunload', function() {
        if (ws) {
            ws.close();
        }
        if (reconnectTimer) {
            clearTimeout(reconnectTimer);
        }
        if (reloadDebounceTimer) {
            clearTimeout(reloadDebounceTimer);
        }
    });
})();

