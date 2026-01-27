/**
 * 搜索模块 - 处理高级搜索功能
 */

// 显示搜索页面
function showSearchPage() {
    showPage('search');
    initializeSearchForm();
}

// 初始化搜索表单
function initializeSearchForm() {
    // 加载分类到搜索表单
    loadCategoriesForSearch();
    
    // 设置默认搜索选项
    const searchForm = document.getElementById('searchForm');
    if (searchForm) {
        searchForm.reset();
        document.getElementById('searchResults').innerHTML = '';
    }
}

// 加载分类到搜索表单
async function loadCategoriesForSearch() {
    try {
        const categories = await getCategories();
        const categorySelect = document.getElementById('searchCategory');
        categorySelect.innerHTML = '<option value="">全部分类</option>';
        
        categories.forEach(category => {
            const option = document.createElement('option');
            option.value = category.id;
            option.textContent = category.name;
            categorySelect.appendChild(option);
        });
    } catch (error) {
        console.error('加载分类失败:', error);
    }
}

// 处理高级搜索
async function handleAdvancedSearch(e) {
    e.preventDefault();
    
    const formData = new FormData(e.target);
    const searchParams = {
        keyword: formData.get('keyword'),
        category_id: formData.get('category'),
        author: formData.get('author'),
        tag: formData.get('tag'),
        sort_by: formData.get('sortBy'),
        order: formData.get('order'),
        page: 1,
        size: 20
    };
    
    // 移除空参数
    Object.keys(searchParams).forEach(key => {
        if (!searchParams[key]) {
            delete searchParams[key];
        }
    });
    
    try {
        const results = await apiCall('/search/posts?' + new URLSearchParams(searchParams));
        renderSearchResults(results);
    } catch (error) {
        console.error('搜索失败:', error);
        showMessage('searchMessage', `搜索失败: ${error.message}`, 'danger');
    }
}

// 渲染搜索结果
function renderSearchResults(results) {
    const container = document.getElementById('searchResults');
    
    if (!results || results.length === 0) {
        container.innerHTML = `
            <div class="text-center py-5">
                <i class="fas fa-search fa-3x text-muted mb-3"></i>
                <h4 class="text-muted">没有找到相关文章</h4>
                <p class="text-muted">尝试使用其他关键词搜索</p>
            </div>
        `;
        return;
    }
    
    let html = '<div class="row">';
    
    results.forEach(post => {
        html += `
            <div class="col-md-6 mb-4">
                <div class="search-result-item">
                    <h5 class="result-title">
                        <a href="#" onclick="showPostDetail(${post.id})">${escapeHtml(post.title)}</a>
                    </h5>
                    <div class="result-meta">
                        <span class="badge bg-primary me-2">${post.author_name}</span>
                        ${post.category ? `<span class="badge bg-secondary me-2">${post.category.name}</span>` : ''}
                        <small class="text-muted">${formatDate(post.created_at)}</small>
                    </div>
                    <p class="result-summary mt-2">
                        ${post.summary ? escapeHtml(truncateText(post.summary, 150)) : 
                          escapeHtml(truncateText(post.content, 150))}
                    </p>
                    <div class="result-stats">
                        <span class="me-3"><i class="fas fa-eye"></i> ${post.clicktimes || 0}</span>
                        <span class="me-3"><i class="fas fa-heart"></i> ${post.liketimes || 0}</span>
                        <span><i class="fas fa-comment"></i> ${post.comment_numbers || 0}</span>
                    </div>
                </div>
            </div>
        `;
    });
    
    html += '</div>';
    container.innerHTML = html;
}

// 搜索用户
async function searchUsers(keyword) {
    try {
        const users = await apiCall(`/search/users?keyword=${encodeURIComponent(keyword)}`);
        renderUserSearchResults(users);
    } catch (error) {
        console.error('搜索用户失败:', error);
    }
}

// 渲染用户搜索结果
function renderUserSearchResults(users) {
    const container = document.getElementById('userSearchResults');
    
    if (!users || users.length === 0) {
        container.innerHTML = `
            <div class="text-center py-5">
                <i class="fas fa-user-slash fa-3x text-muted mb-3"></i>
                <p class="text-muted">没有找到相关用户</p>
            </div>
        `;
        return;
    }
    
    let html = '<div class="row">';
    
    users.forEach(user => {
        html += `
            <div class="col-md-4 mb-3">
                <div class="user-search-result">
                    <div class="user-search-result-header">
                        ${user.avatar_url ? 
                            `<img src="${user.avatar_url}" alt="${user.name || user.username}" class="user-search-avatar">` :
                            `<div class="user-search-default-avatar"><i class="fas fa-user-circle"></i></div>`
                        }
                        <div class="user-search-info">
                            <h6 class="user-search-name">${escapeHtml(user.name || user.username)}</h6>
                            <small class="user-search-username">@${escapeHtml(user.username)}</small>
                        </div>
                    </div>
                    ${user.bio ? `<p class="user-search-bio mt-2">${escapeHtml(truncateText(user.bio, 100))}</p>` : ''}
                    <div class="user-search-actions mt-2">
                        <button class="btn btn-sm btn-outline-primary" onclick="viewUserProfile('${user.username}')">
                            查看资料
                        </button>
                        ${isLoggedIn() && STATE.currentUser.id !== user.id ? `
                            <button class="btn btn-sm btn-primary" onclick="followUser(${user.id})" id="followBtn${user.id}">
                                <i class="fas fa-user-plus"></i> 关注
                            </button>
                        ` : ''}
                    </div>
                </div>
            </div>
        `;
    });
    
    html += '</div>';
    container.innerHTML = html;
}

// 查看用户资料
function viewUserProfile(username) {
    // 跳转到用户公共资料页
    window.location.href = `${CONFIG.API_BASE_URL}/users/${username}`;
}

// 处理快速搜索（文章页面）
async function handleQuickSearch() {
    const keyword = document.getElementById('quickSearchInput').value.trim();
    if (!keyword) return;
    
    try {
        const results = await apiCall(`/posts/search?keyword=${encodeURIComponent(keyword)}`);
        displayQuickSearchResults(results);
    } catch (error) {
        console.error('快速搜索失败:', error);
    }
}