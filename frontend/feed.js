// 加载用户Feed流 - 修复版
async function loadUserFeed(userId) {
    try {
        let url;
        if (userId === 'me') {
            if (!isLoggedIn()) {
                showLogin();
                return;
            }
            url = '/feed/user/me';
        } else {
            url = `/feed/user/${userId}`;
        }
        
        const data = await apiCall(url, 'GET', null, userId === 'me');
        console.log('用户Feed返回数据:', data); // 调试用
        renderFeed(data);
    } catch (error) {
        console.error('加载Feed失败:', error);
        showMessage('feedMessage', `加载失败: ${error.message}`, 'danger');
    }
}

// 加载热门Feed - 修复版
async function loadHotFeed() {
    try {
        const data = await apiCall('/feed/hot');
        console.log('热门Feed返回数据:', data); // 调试用
        renderFeed(data);
    } catch (error) {
        console.error('加载热门Feed失败:', error);
        showMessage('feedMessage', `加载失败: ${error.message}`, 'danger');
    }
}
// 渲染Feed列表 - 修复版
function renderFeed(responseData) {
    console.log('renderFeed接收到数据:', responseData);
    
    const container = document.getElementById('feedContainer');
    const messageDiv = document.getElementById('feedMessage');
    
    if (!container) {
        console.error('找不到feedContainer元素');
        return;
    }
    
    // 清除之前的消息
    if (messageDiv) {
        messageDiv.innerHTML = '';
    }
    
    // 1. 处理错误情况
    if (responseData && responseData.error) {
        console.error('后端返回错误:', responseData.error);
        container.innerHTML = `
            <div class="alert alert-danger">
                <i class="fas fa-exclamation-triangle"></i>
                <strong>加载失败</strong>
                <p class="mt-2">${escapeHtml(responseData.error)}</p>
                <p class="small text-muted mt-2">热门文章功能暂时不可用，请稍后再试。</p>
            </div>
        `;
        return;
    }
    
    // 2. 提取文章数组
    let posts = [];
    
    // 情况1：直接是数组
    if (Array.isArray(responseData)) {
        posts = responseData;
    }
    // 情况2：包含posts字段的对象
    else if (responseData && responseData.posts && Array.isArray(responseData.posts)) {
        posts = responseData.posts;
    }
    // 情况3：其他格式，尝试提取data字段
    else if (responseData && responseData.data && Array.isArray(responseData.data)) {
        posts = responseData.data;
    }
    // 情况4：无效数据
    else {
        console.error('无法识别的数据格式:', responseData);
        posts = [];
    }
    
    console.log('提取到的文章数组:', posts);
    console.log('文章数量:', posts.length);
    
    // 3. 渲染结果
    if (posts.length === 0) {
        container.innerHTML = `
            <div class="text-center py-5">
                <i class="fas fa-newspaper fa-3x text-muted mb-3"></i>
                <h4 class="text-muted">暂时没有动态</h4>
                <p class="text-muted">
                    ${responseData && responseData.total === 0 ? 
                        '关注其他用户或发表文章后，这里会显示动态' : 
                        '暂时没有找到相关文章'}
                </p>
            </div>
        `;
        return;
    }
    
    // 4. 渲染文章列表
    let html = '<div class="row">';
    
    posts.forEach(post => {
        if (!post || !post.id) {
            console.warn('跳过无效文章数据:', post);
            return;
        }
        
        // 安全获取所有字段
        const title = escapeHtml(post.title || '无标题');
        const authorName = escapeHtml(
            post.author_name || 
            (post.author && post.author.name) || 
            (post.user && post.user.name) || 
            '匿名用户'
        );
        const summary = post.summary ? 
            escapeHtml(truncateText(post.summary, 150)) : 
            (post.content ? escapeHtml(truncateText(post.content, 150)) : '');
        
        const categoryName = post.category ? 
            (post.category.name || post.category) : '';
        
        const viewCount = post.clicktimes || post.views || post.view_count || 0;
        const likeCount = post.liketimes || post.likes || post.like_count || 0;
        const commentCount = post.comment_numbers || post.comments || post.comment_count || 0;
        const createdDate = formatDate(post.created_at || post.createdAt);
        
        html += `
            <div class="col-md-6 mb-4">
                <div class="feed-item">
                    <div class="feed-item-header">
                        ${post.author_avatar || (post.author && post.author.avatar_url) || (post.user && post.user.avatar_url) ? 
                            `<img src="${post.author_avatar || (post.author && post.author.avatar_url) || (post.user && post.user.avatar_url)}" 
                                  alt="${authorName}" class="feed-author-avatar">` :
                            `<div class="feed-default-avatar"><i class="fas fa-user-circle"></i></div>`
                        }
                        <div class="feed-author-info">
                            <h6 class="feed-author-name">${authorName}</h6>
                            <small class="feed-time">${createdDate}</small>
                        </div>
                    </div>
                    <div class="feed-item-body">
                        <h5 class="feed-title">
                            <a href="#" onclick="showPostDetail(${post.id})">${title}</a>
                        </h5>
                        ${summary ? `<p class="feed-summary">${summary}</p>` : ''}
                        ${categoryName ? `<span class="feed-category badge bg-primary">${escapeHtml(categoryName)}</span>` : ''}
                    </div>
                    <div class="feed-item-footer">
                        <div class="feed-stats">
                            <span class="me-3"><i class="fas fa-eye"></i> ${viewCount}</span>
                            <span class="me-3"><i class="fas fa-heart"></i> ${likeCount}</span>
                            <span><i class="fas fa-comment"></i> ${commentCount}</span>
                        </div>
                        <button class="btn btn-sm btn-outline-primary" onclick="showPostDetail(${post.id})">
                            阅读 <i class="fas fa-arrow-right"></i>
                        </button>
                    </div>
                </div>
            </div>
        `;
    });
    
    html += '</div>';
    container.innerHTML = html;
}
// 显示Feed页面
function showFeed(type = 'user') {
    if (type === 'user' && !isLoggedIn()) {
        showLogin();
        return;
    }
    
    showPage('feed');
    updateFeedNavigation(type);
    
    // 加载对应的Feed数据
    if (type === 'user') {
        loadUserFeed('me');
    } else {
        loadHotFeed();
    }
}

// 更新Feed导航
function updateFeedNavigation(activeTab) {
    const feedTabs = document.getElementById('feedTabs');
    if (feedTabs) {
        feedTabs.innerHTML = `
            <ul class="nav nav-pills nav-justified">
                <li class="nav-item">
                    <a class="nav-link ${activeTab === 'user' ? 'active' : ''}" 
                       onclick="showFeed('user')">
                        <i class="fas fa-user-friends"></i> 我的动态
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link ${activeTab === 'hot' ? 'active' : ''}" 
                       onclick="showFeed('hot')">
                        <i class="fas fa-fire"></i> 热门文章
                    </a>
                </li>
            </ul>
        `;
    }
}
// 加载热门Feed - 带降级方案
async function loadHotFeed() {
    console.log('加载热门文章...');
    
    const container = document.getElementById('feedContainer');
    const messageDiv = document.getElementById('feedMessage');
    
    // 显示加载中
    container.innerHTML = `
        <div class="text-center py-5">
            <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">加载中...</span>
            </div>
            <p class="mt-2">正在加载热门文章...</p>
        </div>
    `;
    
    try {
        // 尝试调用API
        const data = await apiCall('/feed/hot');
        console.log('热门文章API返回:', data);
        
        // 检查是否有错误
        if (data && data.error) {
            console.error('热门文章API错误:', data.error);
            
            // 降级方案：使用普通文章列表代替
            console.log('降级到普通文章列表...');
            await loadFallbackHotPosts();
            return;
        }
        
        // 正常渲染
        renderFeed(data);
        
    } catch (error) {
        console.error('加载热门文章失败:', error);
        
        // 显示错误信息
        container.innerHTML = `
            <div class="alert alert-warning">
                <i class="fas fa-exclamation-triangle"></i>
                <strong>热门文章功能暂时不可用</strong>
                <p class="mt-2">${error.message || '未知错误'}</p>
                <div class="mt-3">
                    <button class="btn btn-sm btn-primary" onclick="loadFallbackHotPosts()">
                        <i class="fas fa-redo"></i> 查看最新文章
                    </button>
                    <button class="btn btn-sm btn-outline-secondary ms-2" onclick="loadUserFeed('me')">
                        查看我的动态
                    </button>
                </div>
            </div>
        `;
    }
}

// 降级方案：加载最新的文章代替热门文章
async function loadFallbackHotPosts() {
    console.log('加载降级方案：最新文章');
    
    try {
        // 调用普通的文章列表接口，按时间倒序
        const data = await apiCall('/posts?page=1&size=20&order=desc&sort_by=created_at');
        console.log('降级方案返回数据:', data);
        
        // 渲染文章
        if (data && data.posts) {
            renderFeed(data.posts);
            showGlobalMessage('正在显示最新文章（热门文章功能维护中）', 'info');
        } else {
            throw new Error('无法获取文章列表');
        }
    } catch (error) {
        console.error('降级方案也失败:', error);
        
        const container = document.getElementById('feedContainer');
        container.innerHTML = `
            <div class="alert alert-danger">
                <i class="fas fa-exclamation-circle"></i>
                <strong>加载失败</strong>
                <p class="mt-2">${error.message}</p>
                <p class="small text-muted">请检查网络连接或稍后再试。</p>
            </div>
        `;
    }
}