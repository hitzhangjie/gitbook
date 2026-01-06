// Resizer functionality
(function() {
    let isResizing = false;
    let currentResizer = null;
    let startX = 0;
    let startWidth = 0;

    function updateResizerPosition(resizer, sidebar) {
        const sidebarRect = sidebar.getBoundingClientRect();
        const isLeft = sidebar.classList.contains('sidebar-left');
        
        if (isLeft) {
            resizer.style.left = (sidebarRect.right - 2) + 'px';
        } else {
            resizer.style.left = (sidebarRect.left - 2) + 'px';
        }
        resizer.style.top = sidebarRect.top + 'px';
        resizer.style.height = sidebarRect.height + 'px';
    }

    function initResizer(resizerId, sidebarId, isLeft) {
        const resizer = document.getElementById(resizerId);
        const sidebar = document.getElementById(sidebarId);
        
        if (!resizer || !sidebar) return;

        // Update position initially
        updateResizerPosition(resizer, sidebar);

        // Update position on scroll
        sidebar.addEventListener('scroll', function() {
            if (!isResizing) {
                updateResizerPosition(resizer, sidebar);
            }
        });

        // Update position on window resize
        window.addEventListener('resize', function() {
            if (!isResizing) {
                updateResizerPosition(resizer, sidebar);
            }
        });

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

        const sidebar = document.getElementById(
            currentResizer.id === 'resizer-left' ? 'sidebar-left' : 'sidebar-right'
        );
        if (!sidebar) return;

        const isLeft = sidebar.classList.contains('sidebar-left');
        const deltaX = isLeft ? (e.clientX - startX) : (startX - e.clientX);
        const newWidth = Math.max(150, Math.min(500, startWidth + deltaX));
        
        sidebar.style.width = newWidth + 'px';
        updateResizerPosition(currentResizer, sidebar);
    });

    document.addEventListener('mouseup', function() {
        if (isResizing) {
            isResizing = false;
            if (currentResizer) {
                const sidebar = document.getElementById(
                    currentResizer.id === 'resizer-left' ? 'sidebar-left' : 'sidebar-right'
                );
                if (sidebar) {
                    updateResizerPosition(currentResizer, sidebar);
                }
                currentResizer.classList.remove('resizing');
            }
            currentResizer = null;
            document.body.style.cursor = '';
        }
    });

    // Initialize resizers
    initResizer('resizer-left', 'sidebar-left', true);
    initResizer('resizer-right', 'sidebar-right', false);

    // Update positions periodically to handle any layout changes
    // Use a longer interval to reduce unnecessary calculations
    setInterval(function() {
        if (!isResizing) {
            const leftResizer = document.getElementById('resizer-left');
            const leftSidebar = document.getElementById('sidebar-left');
            if (leftResizer && leftSidebar) {
                updateResizerPosition(leftResizer, leftSidebar);
            }
            const rightResizer = document.getElementById('resizer-right');
            const rightSidebar = document.getElementById('sidebar-right');
            if (rightResizer && rightSidebar) {
                updateResizerPosition(rightResizer, rightSidebar);
            }
        }
    }, 500);
})();

// Navigation tree link handling (partial page load)
(function() {
    // Expose initNavTreeLinks globally so it can be called after live reload
    window.initNavTreeLinks = function() {
        const navLinks = document.querySelectorAll('.nav-link');
        
        navLinks.forEach((link) => {
            // Remove existing listeners by cloning
            const newLink = link.cloneNode(true);
            link.parentNode.replaceChild(newLink, link);
            
            newLink.addEventListener('click', async function(e) {
                const href = this.getAttribute('href');
                if (!href || href.startsWith('#')) {
                    return; // Allow hash links to work normally
                }
                
                e.preventDefault();
                
                // Don't navigate if clicking the same page
                const currentPath = window.location.pathname;
                if (href === currentPath || href === currentPath + '/') {
                    return;
                }
                
                await loadPage(href);
            });
        });
    };
    
    // Load page content without refreshing navtree
    async function loadPage(url) {
        try {
            const contentArea = document.getElementById('main-content');
            if (!contentArea) {
                // Fallback to full page load
                window.location.href = url;
                return;
            }
            
            // Save scroll position
            const scrollTop = contentArea.scrollTop;
            
            // Show loading indicator
            const indicator = document.getElementById('live-reload-indicator');
            if (indicator) {
                indicator.textContent = '加载中...';
                indicator.style.display = 'block';
                indicator.style.opacity = '1';
            }
            
            // Fetch new page
            const response = await fetch(url, {
                headers: {
                    'Cache-Control': 'no-cache',
                    'Pragma': 'no-cache'
                }
            });
            
            if (!response.ok) {
                throw new Error('Failed to fetch page');
            }
            
            const newHTML = await response.text();
            const parser = new DOMParser();
            const newDoc = parser.parseFromString(newHTML, 'text/html');
            const body = newDoc.body || newDoc.documentElement;
            
            // Extract elements
            const newTitle = body.querySelector('.article-title');
            const newContent = body.querySelector('.article-content');
            const newTOC = body.querySelector('#toc');
            
            if (!newContent) {
                // Fallback to full page load
                window.location.href = url;
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
                const contentHTML = newContent.innerHTML;
                
                // Fade out
                oldContent.style.transition = 'opacity 0.2s';
                oldContent.style.opacity = '0';
                
                setTimeout(() => {
                    const currentContent = document.querySelector('.article-content');
                    if (!currentContent) {
                        window.location.href = url;
                        return;
                    }
                    
                    // Replace content
                    currentContent.innerHTML = contentHTML;
                    
                    // Force reflow
                    currentContent.offsetHeight;
                    
                    // Fade in
                    currentContent.style.opacity = '1';
                    
                    // Update TOC
                    const oldTOC = document.getElementById('toc');
                    if (oldTOC && newTOC) {
                        oldTOC.innerHTML = newTOC.innerHTML;
                    }
                    
                    // Update active state in navtree (without refreshing the whole tree)
                    updateNavTreeActiveState(url);
                    
                    // Update URL and history
                    window.history.pushState({ path: url }, '', url);
                    
                    // Update page title
                    const pageTitle = newTitle ? newTitle.textContent : '';
                    const bookTitle = document.querySelector('.sidebar-header h2')?.textContent || 'GitBook';
                    document.title = pageTitle ? `${pageTitle} - ${bookTitle}` : bookTitle;
                    
                    // Reinitialize TOC scroll spy and navtree links
                    setTimeout(() => {
                        if (typeof window.initTOCScrollSpy === 'function') {
                            window.initTOCScrollSpy();
                        }
                        // Reinitialize navtree links to ensure event handlers are attached
                        if (typeof window.initNavTreeLinks === 'function') {
                            window.initNavTreeLinks();
                        }
                        // Hide loading indicator
                        const indicator = document.getElementById('live-reload-indicator');
                        if (indicator) {
                            indicator.style.opacity = '0';
                            setTimeout(() => {
                                indicator.style.display = 'none';
                            }, 300);
                        }
                    }, 50);
                }, 200);
            }
        } catch (error) {
            console.error('Failed to load page:', error);
            // Fallback to full page load
            window.location.href = url;
        }
    }
    
    // Update active state in navtree without refreshing the whole tree
    function updateNavTreeActiveState(currentUrl) {
        const navLinks = document.querySelectorAll('.nav-link');
        const currentPath = currentUrl.replace(window.location.origin, '');
        
        navLinks.forEach((link) => {
            const href = link.getAttribute('href');
            if (href === currentPath || href === currentPath + '/') {
                link.classList.add('active');
            } else {
                link.classList.remove('active');
            }
        });
    }
    
    // Handle browser back/forward buttons
    window.addEventListener('popstate', function(e) {
        if (e.state && e.state.path) {
            loadPage(e.state.path);
        } else {
            loadPage(window.location.pathname);
        }
    });
    
    // Initialize on page load
    window.initNavTreeLinks();
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

            // Extract elements to update from the parsed document body
            // Query from body to ensure we get the correct elements
            const body = newDoc.body || newDoc.documentElement;
            const newTitle = body.querySelector('.article-title');
            const newContent = body.querySelector('.article-content');
            const newTOC = body.querySelector('#toc');
            const newNavTree = body.querySelector('.nav-tree');
            
            console.log('Parsed elements:', {
                hasBody: !!newDoc.body,
                hasTitle: !!newTitle,
                hasContent: !!newContent,
                hasTOC: !!newTOC,
                hasNavTree: !!newNavTree
            });

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
                
                // Debug: check if contentHTML has content
                if (!contentHTML || contentHTML.trim().length === 0) {
                    console.warn('New content is empty, falling back to full reload');
                    window.location.reload();
                    return;
                }
                
                console.log('Updating content, length:', contentHTML.length);
                
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

                    console.log('Replacing content, old length:', currentContent.innerHTML.length, 'new length:', contentHTML.length);
                    
                    // Replace content
                    currentContent.innerHTML = contentHTML;
                    
                    // Verify content was updated
                    if (currentContent.innerHTML.length === 0) {
                        console.error('Content update failed: content is empty after update');
                        window.location.reload();
                        return;
                    }
                    
                    // Force reflow to ensure transition works
                    currentContent.offsetHeight;
                    
                    // Fade in
                    currentContent.style.opacity = '1';
                    console.log('Content updated successfully');

                    // Update TOC after content is updated
                    const oldTOC = document.getElementById('toc');
                    if (oldTOC && newTOC) {
                        oldTOC.innerHTML = newTOC.innerHTML;
                    }

                    // Update navigation active state only (don't refresh the whole navtree)
                    // Only update if structure actually changed (e.g., new chapters added)
                    const oldNavTree = document.querySelector('.nav-tree');
                    if (oldNavTree && newNavTree) {
                        // Check if navigation structure changed by comparing link counts and URLs
                        const oldLinks = oldNavTree.querySelectorAll('.nav-link');
                        const newLinks = newNavTree.querySelectorAll('.nav-link');
                        let structureChanged = oldLinks.length !== newLinks.length;
                        
                        if (!structureChanged) {
                            // Check if URLs match
                            oldLinks.forEach((oldLink, index) => {
                                if (index < newLinks.length) {
                                    if (oldLink.getAttribute('href') !== newLinks[index].getAttribute('href')) {
                                        structureChanged = true;
                                    }
                                }
                            });
                        }
                        
                        if (structureChanged) {
                            // Only update if structure changed (e.g., new chapters)
                            oldNavTree.innerHTML = newNavTree.innerHTML;
                            // Reinitialize navtree link handlers
                            if (typeof window.initNavTreeLinks === 'function') {
                                window.initNavTreeLinks();
                            }
                        } else {
                            // Just update active state
                            const currentPath = window.location.pathname;
                            const oldLinksAfter = oldNavTree.querySelectorAll('.nav-link');
                            oldLinksAfter.forEach((link) => {
                                const href = link.getAttribute('href');
                                if (href === currentPath || href === currentPath + '/') {
                                    link.classList.add('active');
                                } else {
                                    link.classList.remove('active');
                                }
                            });
                        }
                    }

                    // Restore scroll position after content is fully updated and rendered
                    // Use multiple requestAnimationFrame and setTimeout to ensure DOM is fully rendered
                    const restoreScroll = () => {
                        const updatedContentArea = document.getElementById('main-content');
                        if (!updatedContentArea) {
                            console.warn('Content area not found for scroll restoration');
                            isReloading = false;
                            hideIndicator();
                            return;
                        }

                        // Calculate max scroll position
                        const maxScroll = updatedContentArea.scrollHeight - updatedContentArea.clientHeight;
                        // Restore scroll position, but don't exceed max scroll
                        const targetScroll = Math.min(scrollTop, maxScroll);
                        
                        // Use scrollTo for better compatibility
                        updatedContentArea.scrollTo({
                            top: targetScroll,
                            behavior: 'auto' // Use 'auto' instead of 'smooth' for instant scroll
                        });
                        
                        // Also set scrollTop directly as fallback
                        updatedContentArea.scrollTop = targetScroll;
                        
                        // Verify scroll position was set and ensure it sticks
                        setTimeout(() => {
                            const actualScroll = updatedContentArea.scrollTop;
                            if (Math.abs(actualScroll - targetScroll) > 1) {
                                console.warn('Scroll position mismatch, retrying...', {
                                    target: targetScroll,
                                    actual: actualScroll,
                                    scrollHeight: updatedContentArea.scrollHeight,
                                    clientHeight: updatedContentArea.clientHeight
                                });
                                // Retry once more
                                updatedContentArea.scrollTop = targetScroll;
                                updatedContentArea.scrollTo({ top: targetScroll, behavior: 'auto' });
                            } else {
                                console.log('Scroll position restored successfully:', actualScroll);
                            }
                            
                            // Force scroll position one more time to ensure it's set
                            updatedContentArea.scrollTop = targetScroll;
                            
                            // Reinitialize TOC scroll spy AFTER scroll position is restored
                            // Pass the target scroll position to ensure it's preserved
                            if (oldTOC && newTOC) {
                                // Delay initTOCScrollSpy to ensure scroll position is fully applied
                                setTimeout(() => {
                                    initTOCScrollSpy(targetScroll);
                                }, 10);
                            } else {
                                isReloading = false;
                                hideIndicator();
                            }
                        }, 50);
                    };

                    // Use requestAnimationFrame to ensure DOM is rendered, then add a small delay
                    requestAnimationFrame(() => {
                        requestAnimationFrame(() => {
                            // Add a small delay to ensure content is fully laid out
                            setTimeout(restoreScroll, 50);
                        });
                    });
                }, 200);
            } else {
                console.warn('Content update failed:', { oldContent: !!oldContent, newContent: !!newContent });
                // Fallback to full reload if content elements not found
                window.location.reload();
                return;
            }

        } catch (error) {
            console.error('Failed to reload page:', error);
            // Fallback to full reload
            window.location.reload();
        }
    }

    // Reinitialize TOC scroll spy after content update
    // scrollPosition: optional scroll position to restore after replacing element
    // Expose globally so it can be called from navtree link handler
    window.initTOCScrollSpy = function(scrollPosition) {
        // Wait a bit for DOM to update
        setTimeout(() => {
            const tocLinks = document.querySelectorAll('.toc-link');
            const headings = document.querySelectorAll('.article-content h1, .article-content h2, .article-content h3, .article-content h4');
            const contentArea = document.getElementById('main-content');

            if (tocLinks.length === 0 || headings.length === 0 || !contentArea) {
                if (scrollPosition !== undefined) {
                    isReloading = false;
                    hideIndicator();
                }
                return;
            }

            // Use provided scroll position, or save current scroll position
            const savedScrollTop = scrollPosition !== undefined ? scrollPosition : contentArea.scrollTop;
            console.log('initTOCScrollSpy: saving scroll position', savedScrollTop);

            // Remove existing scroll listener by cloning the element (removes all listeners)
            const newContentArea = contentArea.cloneNode(true);
            contentArea.parentNode.replaceChild(newContentArea, contentArea);
            
            // Restore scroll position after replacing element
            // Use both scrollTo and scrollTop for maximum compatibility
            newContentArea.scrollTo({ top: savedScrollTop, behavior: 'auto' });
            newContentArea.scrollTop = savedScrollTop;
            
            // Force a reflow to ensure scroll position is applied
            newContentArea.offsetHeight;
            
            // Verify scroll position was restored with multiple attempts
            const verifyScroll = () => {
                const actualScroll = newContentArea.scrollTop;
                if (Math.abs(actualScroll - savedScrollTop) > 1) {
                    console.warn('initTOCScrollSpy: scroll position mismatch, retrying...', {
                        target: savedScrollTop,
                        actual: actualScroll,
                        scrollHeight: newContentArea.scrollHeight,
                        clientHeight: newContentArea.clientHeight
                    });
                    // Retry with both methods
                    newContentArea.scrollTop = savedScrollTop;
                    newContentArea.scrollTo({ top: savedScrollTop, behavior: 'auto' });
                    // Force reflow again
                    newContentArea.offsetHeight;
                } else {
                    console.log('initTOCScrollSpy: scroll position restored successfully:', actualScroll);
                }
            };
            
            // Verify immediately and after a delay
            verifyScroll();
            requestAnimationFrame(() => {
                verifyScroll();
                // Mark reloading as complete if we were restoring scroll position
                if (scrollPosition !== undefined) {
                    setTimeout(() => {
                        isReloading = false;
                        hideIndicator();
                    }, 10);
                }
            });

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

