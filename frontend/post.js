/**
 * 文章模块 - 处理文章的增删改查
 */

// 加载文章列表
async function loadPosts() {
    const container = document.getElementById('postsContainer');
    const pagination = document.getElementById('postsPagination');
    
    // 显示加载中
    container.innerHTML = `
        <div class="text-center py-5">
            <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">加载中...</span>
            </div>
            <p class="mt-2">正在加载文章...</p>
        </div>
    `;
    
    try {
        // 构建查询参数
        let url = `/posts?page=${STATE.currentPageNum}&size=${CONFIG.ITEMS_PER_PAGE}`;
        
        if (STATE.searchKeyword) {
            url += `&keyword=${encodeURIComponent(STATE.searchKeyword)}`;
        }
        
        if (STATE.currentCategory) {
            url += `&category_id=${STATE.currentCategory}`;
        }
        
        const data = await apiCall(url);
        
        if (!data.posts || data.posts.length === 0) {
            container.innerHTML = `
                <div class="text-center py-5">
                    <i class="bi bi-journal-x display-1 text-muted"></i>
                    <h4 class="mt-3">暂无文章</h4>
                    <p class="text-muted">${STATE.searchKeyword ? '没有找到相关文章' : '还没有人发表文章'}</p>
                    ${isLoggedIn() ? 
                        '<button class="btn btn-primary mt-3" onclick="showCreatePost()">发表第一篇文章</button>' : 
                        '<button class="btn btn-primary mt-3" onclick="showLogin()">登录后发表文章</button>'
                    }
                </div>
            `;
            pagination.classList.add('d-none');
            return;
        }
        
        // 渲染文章列表
        let html = '<div class="row">';
        data.posts.forEach(post => {
            html += `
                <div class="col-md-6 mb-4">
                    <div class="post-item">
                        <h5 class="post-title cursor-pointer" onclick="showPostDetail(${post.id})">
                            ${escapeHtml(post.title)}
                        </h5>
                        <div class="post-meta d-flex justify-content-between">
                            <div>
                                <span class="badge bg-primary me-2">${post.author_name}</span>
                                ${post.category ? 
                                    `<span class="badge bg-secondary">${post.category.name}</span>` : 
                                    ''
                                }
                            </div>
                            <div class="text-muted">
                                ${formatDate(post.created_at)}
                            </div>
                        </div>
                        <p class="post-excerpt">
                            ${post.summary ? escapeHtml(truncateText(post.summary, 150)) : 
                              escapeHtml(truncateText(post.content, 150))}
                        </p>
                        <div class="d-flex justify-content-between align-items-center">
                            <div class="post-stats">
                                <span class="me-3"><i class="bi bi-eye"></i> ${post.clicktimes || 0}</span>
                                <span class="me-3"><i class="bi bi-heart"></i> ${post.liketimes || 0}</span>
                                <span><i class="bi bi-chat"></i> ${post.comment_numbers || 0}</span>
                            </div>
                            <button class="btn btn-sm btn-outline-primary" onclick="showPostDetail(${post.id})">
                                阅读全文 <i class="bi bi-arrow-right"></i>
                            </button>
                        </div>
                    </div>
                </div>
            `;
        });
        html += '</div>';
        container.innerHTML = html;
        
        // 渲染分页
        STATE.totalPages = Math.ceil(data.total / CONFIG.ITEMS_PER_PAGE);
        renderPagination(pagination, STATE.currentPageNum, STATE.totalPages);
        pagination.classList.remove('d-none');
        
    } catch (error) {
        container.innerHTML = `
            <div class="alert alert-danger">
                <i class="bi bi-exclamation-triangle"></i> 加载文章失败: ${error.message}
            </div>
        `;
        pagination.classList.add('d-none');
    }
}

// 渲染分页组件
function renderPagination(container, currentPage, totalPages) {
    if (totalPages <= 1) {
        container.classList.add('d-none');
        return;
    }
    
    let html = '';
    const maxPages = 5;
    let startPage = Math.max(1, currentPage - Math.floor(maxPages / 2));
    let endPage = Math.min(totalPages, startPage + maxPages - 1);
    
    if (endPage - startPage + 1 < maxPages) {
        startPage = Math.max(1, endPage - maxPages + 1);
    }
    
    // 上一页按钮
    html += `
        <li class="page-item ${currentPage === 1 ? 'disabled' : ''}">
            <a class="page-link" href="#" ${currentPage > 1 ? `onclick="changePage(${currentPage - 1})"` : ''}>
                <i class="bi bi-chevron-left"></i>
            </a>
        </li>
    `;
    
    // 第一页
    if (startPage > 1) {
        html += `
            <li class="page-item">
                <a class="page-link" href="#" onclick="changePage(1)">1</a>
            </li>
        `;
        if (startPage > 2) {
            html += '<li class="page-item disabled"><span class="page-link">...</span></li>';
        }
    }
    
    // 页码
    for (let i = startPage; i <= endPage; i++) {
        html += `
            <li class="page-item ${i === currentPage ? 'active' : ''}">
                <a class="page-link" href="#" onclick="changePage(${i})">${i}</a>
            </li>
        `;
    }
    
    // 最后一页
    if (endPage < totalPages) {
        if (endPage < totalPages - 1) {
            html += '<li class="page-item disabled"><span class="page-link">...</span></li>';
        }
        html += `
            <li class="page-item">
                <a class="page-link" href="#" onclick="changePage(${totalPages})">${totalPages}</a>
            </li>
        `;
    }
    
    // 下一页按钮
    html += `
        <li class="page-item ${currentPage === totalPages ? 'disabled' : ''}">
            <a class="page-link" href="#" ${currentPage < totalPages ? `onclick="changePage(${currentPage + 1})"` : ''}>
                <i class="bi bi-chevron-right"></i>
            </a>
        </li>
    `;
    
    container.querySelector('ul').innerHTML = html;
}

// 切换页码
function changePage(page) {
    STATE.currentPageNum = page;
    loadPosts();
    
    // 滚动到顶部
    window.scrollTo({ top: 0, behavior: 'smooth' });
}

// 处理文章提交（创建/更新）
async function handlePostSubmit(e) {
    e.preventDefault();
    
    const postId = document.getElementById('postId').value;
    const title = document.getElementById('postTitle').value.trim();
    const content = document.getElementById('postContent').value.trim();
    const summary = document.getElementById('postSummary').value.trim();
    const categoryId = document.getElementById('postCategory').value;
    
    // 验证表单
    if (!title) {
        showMessage('postMessage', '请输入文章标题', 'warning');
        return;
    }
    
    if (!content) {
        showMessage('postMessage', '请输入文章内容', 'warning');
        return;
    }
    
    if (!categoryId) {
        showMessage('postMessage', '请选择文章分类', 'warning');
        return;
    }
    
    const isUpdate = !!postId;
    const postData = {
        title: title,
        content: content,
        category_id: parseInt(categoryId),
        tag_ids: [],
        visibility: "public"
    };
    
    if (summary) {
        postData.summary = summary;
    }
    
    try {
        showLoading('postSubmitBtn');
        
        let data;
        if (isUpdate) {
            data = await apiCall(`/posts/${postId}`, 'PUT', postData, true);
        } else {
            data = await apiCall('/posts', 'POST', postData, true);
        }
        
        showMessage('postMessage', 
            `${isUpdate ? '更新' : '发布'}成功！`, 
            'success');
        
        // 清空表单
        document.getElementById('postForm').reset();
        
        setTimeout(() => {
            if (isUpdate) {
                showPostDetail(postId);
            } else {
                showPosts();
            }
        }, 1500);
        
    } catch (error) {
        showMessage('postMessage', 
            `${isUpdate ? '更新' : '发布'}失败: ${error.message}`, 
            'danger');
    } finally {
        hideLoading('postSubmitBtn');
    }
}

// 加载文章详情
async function loadPostDetail(postId) {
    const container = document.querySelector('#postDetailPage .card-body');
    
    // 显示加载中
    container.innerHTML = `
        <div class="text-center py-5">
            <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">加载中...</span>
            </div>
            <p class="mt-2">正在加载文章...</p>
        </div>
    `;
    
    try {
        const post = await getPost(postId);
        
        // 检查marked库是否可用
        let markdownContent = post.content || '';
        if (typeof marked !== 'undefined') {
            markdownContent = marked.parse(markdownContent);
        } else {
            // 如果marked不可用，使用简单的HTML转义
            markdownContent = escapeHtml(markdownContent).replace(/\n/g, '<br>');
        }
        
        // 构建文章内容HTML
        let html = `
            <div class="mb-4">
                <h1 class="mb-3">${escapeHtml(post.title)}</h1>
                
                <div class="d-flex align-items-center mb-4 text-muted">
                    <div class="d-flex align-items-center me-4">
                        <i class="bi bi-person-circle me-2"></i>
                        <span>${post.author_name || post.author?.name || '未知作者'}</span>
                    </div>
                    <div class="d-flex align-items-center me-4">
                        <i class="bi bi-calendar me-2"></i>
                        <span>${formatDate(post.created_at || post.createdAt)}</span>
                    </div>
                    ${post.category ? `
                        <div class="d-flex align-items-center">
                            <i class="bi bi-tag me-2"></i>
                            <span class="badge bg-primary">${post.category.name}</span>
                        </div>
                    ` : ''}
                </div>
                
                <div class="post-stats d-flex mb-4">
                    <div class="me-4"><i class="bi bi-eye"></i> ${post.clicktimes || post.views || 0} 阅读</div>
                    <div class="me-4"><i class="bi bi-heart"></i> ${post.liketimes || post.likes || 0} 点赞</div>
                    <div><i class="bi bi-chat"></i> ${post.comment_numbers || post.comments || 0} 评论</div>
                </div>
            </div>
            
            ${post.summary ? `
                <div class="alert alert-info mb-4">
                    <i class="bi bi-info-circle"></i> ${escapeHtml(post.summary)}
                </div>
            ` : ''}
            
            <div class="markdown-content mb-5">
                ${markdownContent}
            </div>
            
            <div class="post-actions mt-5 pt-4 border-top">
                <div class="d-flex gap-2">
                    <button class="btn btn-outline-primary" onclick="likePost(${postId})" id="likeBtn${postId}">
                        <i class="bi bi-heart"></i> 点赞 (${post.liketimes || post.likes || 0})
                    </button>
                    <button class="btn btn-outline-success" onclick="starPost(${postId})" id="starBtn${postId}">
                        <i class="bi bi-star"></i> 收藏 (${post.staredtimes || post.stars || 0})
                    </button>
                </div>
            </div>
        `;
        
        container.innerHTML = html;
        
        // 更新操作按钮（如果是作者显示编辑删除按钮）
        const actionsContainer = document.getElementById('postActions');
        if (isLoggedIn() && STATE.currentUser && STATE.currentUser.id === (post.user_id || post.userId)) {
            actionsContainer.innerHTML = `
                <button class="btn btn-sm btn-outline-warning" onclick="showEditPost(${postId})">
                    <i class="bi bi-pencil"></i> 编辑
                </button>
                <button class="btn btn-sm btn-outline-danger" onclick="deletePost(${postId})">
                    <i class="bi bi-trash"></i> 删除
                </button>
            `;
        } else {
            actionsContainer.innerHTML = '';
        }
        
        // 检查点赞和收藏状态
        if (isLoggedIn()) {
            checkPostLikeStatus(postId);
            checkPostStarStatus(postId);
        }
        
        // 确保评论区域在文章内容加载后显示
        const commentForm = document.getElementById('commentForm');
        const commentsContainer = document.getElementById('commentsContainer');
        
        // 如果用户未登录，评论表单显示登录提示
        if (!isLoggedIn()) {
            if (commentForm) {
                const textarea = commentForm.querySelector('textarea');
                textarea.placeholder = '请登录后发表评论...';
                textarea.disabled = true;
                const submitBtn = commentForm.querySelector('button[type="submit"]');
                submitBtn.textContent = '请先登录';
                submitBtn.disabled = true;
                submitBtn.onclick = function(e) {
                    e.preventDefault();
                    showLogin();
                };
            }
        }
        
    } catch (error) {
        console.error('加载文章失败:', error);
        container.innerHTML = `
            <div class="alert alert-danger">
                <i class="bi bi-exclamation-triangle"></i> 加载文章失败: ${error.message}
            </div>
        `;
    }
}

// 点赞文章
async function likePost(postId) {
    if (!isLoggedIn()) {
        showLogin();
        return;
    }
    
    const likeBtn = document.getElementById(`likeBtn${postId}`);
    if (!likeBtn) return;
    
    const originalText = likeBtn.innerHTML;
    const originalClass = likeBtn.className;
    
    try {
        // 显示加载状态
        likeBtn.disabled = true;
        likeBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 处理中...';
        
        console.log(`尝试点赞文章 ${postId}...`);
        await apiCall(`/posts/${postId}/like`, 'POST', null, true);
        
        // 更新按钮状态
        const currentMatch = originalText.match(/\d+/);
        const currentCount = currentMatch ? parseInt(currentMatch[0]) : 0;
        likeBtn.innerHTML = `<i class="bi bi-heart-fill text-danger"></i> 已点赞 (${currentCount + 1})`;
        likeBtn.className = originalClass.replace('btn-outline-primary', 'btn-danger');
        
        // 显示成功提示
        showGlobalMessage('点赞成功！', 'success', 3000);
        
    } catch (error) {
        console.error('点赞失败:', error);
        
        // 恢复按钮状态
        likeBtn.disabled = false;
        likeBtn.innerHTML = originalText;
        likeBtn.className = originalClass;
        
        // 检查特定错误
        if (error.message.includes('已经点赞过') || error.message.includes('already liked')) {
            showGlobalMessage('您已经点赞过这篇文章了', 'info', 3000);
        } else {
            showGlobalMessage(`点赞失败: ${error.message}`, 'danger', 3000);
        }
    }
}

// 收藏文章
async function starPost(postId) {
    if (!isLoggedIn()) {
        showLogin();
        return;
    }
    
    const starBtn = document.getElementById(`starBtn${postId}`);
    if (!starBtn) return;
    
    const originalText = starBtn.innerHTML;
    const originalClass = starBtn.className;
    
    try {
        // 显示加载状态
        starBtn.disabled = true;
        starBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 处理中...';
        
        console.log(`尝试收藏文章 ${postId}...`);
        await apiCall(`/posts/${postId}/star`, 'POST', null, true);
        
        // 更新按钮状态
        const currentMatch = originalText.match(/\d+/);
        const currentCount = currentMatch ? parseInt(currentMatch[0]) : 0;
        starBtn.innerHTML = `<i class="bi bi-star-fill text-warning"></i> 已收藏 (${currentCount + 1})`;
        starBtn.className = originalClass.replace('btn-outline-success', 'btn-warning');
        
        // 显示成功提示
        showGlobalMessage('收藏成功！', 'success', 3000);
        
    } catch (error) {
        console.error('收藏失败:', error);
        
        // 恢复按钮状态
        starBtn.disabled = false;
        starBtn.innerHTML = originalText;
        starBtn.className = originalClass;
        
        // 检查特定错误
        if (error.message.includes('已经收藏过') || error.message.includes('already starred')) {
            showGlobalMessage('您已经收藏过这篇文章了', 'info', 3000);
        } else {
            showGlobalMessage(`收藏失败: ${error.message}`, 'danger', 3000);
        }
    }
}

// 删除文章
async function deletePost(postId) {
    if (!confirm('确定要删除这篇文章吗？此操作不可恢复。')) {
        return;
    }
    
    try {
        console.log(`尝试删除文章 ${postId}...`);
        await apiCall(`/posts/${postId}`, 'DELETE', null, true);
        
        showGlobalMessage('文章删除成功！', 'success', 3000);
        
        setTimeout(() => {
            showPosts();
        }, 1500);
        
    } catch (error) {
        console.error('删除文章失败:', error);
        showGlobalMessage(`删除失败: ${error.message}`, 'danger', 3000);
    }
}

// 检查文章点赞状态
async function checkPostLikeStatus(postId) {
    try {
        const data = await apiCall(`/posts/${postId}/stats`, 'GET');
        const likeBtn = document.getElementById(`likeBtn${postId}`);
        
        if (data.is_liked) {
            likeBtn.innerHTML = `<i class="bi bi-heart-fill text-danger"></i> 已点赞 (${data.likes})`;
            likeBtn.classList.remove('btn-outline-primary');
            likeBtn.classList.add('btn-danger');
        }
    } catch (error) {
        console.error('检查点赞状态失败:', error);
    }
}

// 检查文章收藏状态
async function checkPostStarStatus(postId) {
    try {
        const data = await apiCall(`/posts/${postId}/stats`, 'GET');
        const starBtn = document.getElementById(`starBtn${postId}`);
        
        if (data.is_starred) {
            starBtn.innerHTML = `<i class="bi bi-star-fill text-warning"></i> 已收藏 (${data.stars})`;
            starBtn.classList.remove('btn-outline-success');
            starBtn.classList.add('btn-warning');
        }
    } catch (error) {
        console.error('检查收藏状态失败:', error);
    }
}

// 获取文章
async function getPost(postId) {
    return await apiCall(`/posts/${postId}`);
}