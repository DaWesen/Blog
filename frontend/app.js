// app.js 顶部修改
console.log('app.js 正在加载...');

// 检查全局变量是否存在
if (!window.CONFIG) {
    console.error('警告: CONFIG 未定义，使用默认值');
    window.CONFIG = {
        API_BASE_URL: 'http://localhost:8080/api',
        ITEMS_PER_PAGE: 10,
        DEBOUNCE_DELAY: 500,
        TOKEN_KEY: 'blog_token',
        USER_KEY: 'blog_user',
        AVATAR_TYPES: ['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp'],
        MAX_AVATAR_SIZE: 2 * 1024 * 1024
    };
}

if (!window.STATE) {
    console.error('警告: STATE 未定义，使用默认值');
    window.STATE = {
        currentUser: null,
        currentToken: null,
        currentPage: 'home',
        currentPostId: null,
        categories: [],
        searchKeyword: '',
        currentCategory: '',
        currentPageNum: 1,
        totalPages: 1
    };
}

// 引用全局变量
const CONFIG = window.CONFIG;
const STATE = window.STATE;

console.log('app.js: 使用以下配置:');
console.log('- CONFIG:', CONFIG);
console.log('- STATE:', STATE);
// 安全的获取头像URL函数
function getAvatarUrl(user, size = 'normal') {
    if (!user || !user.username) {
        return null;
    }
    
    if (user.avatar_url && user.avatar_url.startsWith('http')) {
        return user.avatar_url;
    }
    
    // 使用默认头像
    return null;
}

// 页面切换函数
function showPage(pageId) {
    // 隐藏所有页面
    document.querySelectorAll('.page').forEach(page => {
        page.classList.remove('active');
        page.classList.add('d-none');
    });
    
    // 显示目标页面
    const targetPage = document.getElementById(pageId + 'Page');
    if (targetPage) {
        targetPage.classList.remove('d-none');
        setTimeout(() => {
            targetPage.classList.add('active');
        }, 10);
        STATE.currentPage = pageId;
        
        // 执行页面特定的初始化
        switch(pageId) {
            case 'home':
                updateHomePage();
                break;
            case 'posts':
                loadCategoriesForFilter();
                loadPosts();
                break;
            case 'categories':
                if (isLoggedIn()) {
                    loadCategories();
                } else {
                    showLogin();
                }
                break;
            case 'postDetail':
                if (STATE.currentPostId) {
                    loadPostDetail(STATE.currentPostId);
                    setTimeout(() => {
                        loadPostComments(STATE.currentPostId);
                    }, 300);
                } else {
                    showPosts();
                }
                break;
            case 'login':
                if (isLoggedIn()) {
                    showHome();
                }
                break;
            case 'register':
                if (isLoggedIn()) {
                    showHome();
                }
                break;
            case 'profile':
                if (isLoggedIn()) {
                    loadUserProfile();
                } else {
                    showLogin();
                }
                break;
            case 'editPost':
                if (isLoggedIn()) {
                    loadCategoriesForPost();
                    if (STATE.currentPostId) {
                        loadPostForEdit(STATE.currentPostId);
                    }
                } else {
                    showLogin();
                }
                break;
            case 'editCategory':
                if (isLoggedIn()) {
                    // 编辑分类页面的初始化
                } else {
                    showLogin();
                }
                break;
            // 页面切换函数 - 修改Feed部分
case 'feed':
    if (isLoggedIn()) {
        // 已登录：先尝试显示用户动态
        STATE.feedType = 'user';
        updateFeedNavigation('user');
        
        // 延迟加载，确保DOM渲染完成
        setTimeout(() => {
            loadUserFeed('me');
        }, 100);
    } else {
        // 未登录：显示热门文章
        STATE.feedType = 'hot';
        updateFeedNavigation('hot');
        
        setTimeout(() => {
            // 尝试加载热门，如果失败则显示提示
            loadHotFeed();
        }, 100);
    }
    break;
            case 'search':
                initializeSearchForm();
                break;
        }
    } else {
        console.error(`找不到页面: ${pageId}Page`);
    }
    
    updateNavigation();
}

// 快捷页面切换函数
function showHome() { 
    if (window.showPage) {
        showPage('home'); 
    } else {
        location.reload();
    }
}
function showLogin() { showPage('login'); }
function showRegister() { showPage('register'); }
function showPosts() { showPage('posts'); }
function showPostDetail(postId) { 
    STATE.currentPostId = postId;
    showPage('postDetail'); 
}
function showCreatePost() { 
    loadCategoriesForPost();
    document.getElementById('editPostTitle').innerHTML = '<i class="fas fa-pen-fancy"></i> 写文章';
    document.getElementById('postForm').reset();
    document.getElementById('postId').value = '';
    document.getElementById('postSubmitBtn').innerHTML = '<i class="fas fa-paper-plane"></i> 发布文章';
    showPage('editPost'); 
}
function showEditPost(postId) {
    STATE.currentPostId = postId;
    loadCategoriesForPost();
    document.getElementById('editPostTitle').innerHTML = '<i class="fas fa-edit"></i> 编辑文章';
    loadPostForEdit(postId);
    showPage('editPost');
}
function showCategories() { showPage('categories'); }
function showCreateCategory() { 
    document.getElementById('editCategoryTitle').innerHTML = '<i class="fas fa-tag"></i> 新建分类';
    document.getElementById('categoryForm').reset();
    document.getElementById('categoryId').value = '';
    showPage('editCategory'); 
}
function showEditCategory(categoryId) {
    document.getElementById('editCategoryTitle').innerHTML = '<i class="fas fa-edit"></i> 编辑分类';
    loadCategoryForEdit(categoryId);
    showPage('editCategory');
}
function showProfile() { showPage('profile'); }
function showFeed(type = 'user') {
    if (type === 'user' && !isLoggedIn()) {
        showLogin();
        return;
    }
    STATE.feedType = type;
    showPage('feed');
}
function showSearch() {
    showPage('search');
}

// 更新导航栏 - 修复头像URL问题
function updateNavigation() {
    const userSection = document.getElementById('userSection');
    
    if (!userSection) {
        console.warn('userSection元素不存在');
        return;
    }
    
    if (isLoggedIn()) {
        // 安全地获取用户信息
        const userName = STATE.currentUser?.name || STATE.currentUser?.username || '用户';
        const userAvatar = getAvatarUrl(STATE.currentUser);
        
        // 构建头像显示
        const avatarHTML = userAvatar ? 
            `<img src="${userAvatar}" alt="${userName}" class="navbar-avatar" onerror="this.style.display='none'; this.parentElement.innerHTML='<div class=\"navbar-default-avatar\"><i class=\"fas fa-user-circle\"></i></div>';">` :
            `<div class="navbar-default-avatar"><i class="fas fa-user-circle"></i></div>`;
        
        userSection.innerHTML = `
            <div class="dropdown">
                <button class="btn btn-outline-light dropdown-toggle d-flex align-items-center" type="button" data-bs-toggle="dropdown">
                    ${avatarHTML}
                    <span class="fw-bold ms-2" style="color: var(--text-primary)">${userName}</span>
                </button>
                <ul class="dropdown-menu dropdown-menu-end">
                    <li><a class="dropdown-item" href="#" onclick="showProfile()"><i class="fas fa-user me-2"></i> 个人资料</a></li>
                    <li><a class="dropdown-item" href="#" onclick="showCreatePost()"><i class="fas fa-pen me-2"></i> 写文章</a></li>
                    <li><a class="dropdown-item" href="#" onclick="showCategories()"><i class="fas fa-tags me-2"></i> 分类管理</a></li>
                    <li><hr class="dropdown-divider"></li>
                    <li><a class="dropdown-item text-danger" href="#" onclick="logout()"><i class="fas fa-sign-out-alt me-2"></i> 退出登录</a></li>
                </ul>
            </div>
        `;
        
        // 更新首页按钮
        const homeActions = document.getElementById('homeActions');
        if (homeActions) {
            const homeAvatar = userAvatar ? 
                `<img src="${userAvatar}" alt="${userName}" class="home-avatar mb-2" style="width: 80px; height: 80px; border-radius: 50%; object-fit: cover; border: 3px solid var(--primary-color);" onerror="this.style.display='none'; this.parentElement.innerHTML='<i class=\"fas fa-user-circle fa-3x text-primary mb-2\"></i>';">` :
                `<i class="fas fa-user-circle fa-3x text-primary mb-2"></i>`;
            
            homeActions.innerHTML = `
                <div class="welcome-user-info text-center mb-4">
                    <div class="mb-3">
                        ${homeAvatar}
                        <h3 class="fw-bold" style="color: var(--text-primary)">欢迎回来，${userName}！</h3>
                        <p class="text-secondary">今天有什么新想法要分享吗？</p>
                    </div>
                    <div class="d-flex gap-3 justify-content-center">
                        <button class="btn-custom btn-primary-custom btn-lg" onclick="showCreatePost()">
                            <i class="fas fa-pen-fancy"></i> 写文章
                        </button>
                        <button class="btn-custom btn-outline-custom btn-lg" onclick="showPosts()">
                            <i class="fas fa-newspaper"></i> 查看文章
                        </button>
                    </div>
                </div>
            `;
        }
    } else {
        userSection.innerHTML = `
            <div class="d-flex gap-2">
                <button class="btn-custom btn-outline-custom" onclick="showLogin()">
                    <i class="fas fa-sign-in-alt"></i> 登录
                </button>
                <button class="btn-custom btn-primary-custom" onclick="showRegister()">
                    <i class="fas fa-user-plus"></i> 注册
                </button>
            </div>
        `;
        
        const homeActions = document.getElementById('homeActions');
        if (homeActions) {
            homeActions.innerHTML = `
                <div class="text-center">
                    <div class="mb-4">
                        <i class="fas fa-graduation-cap fa-4x text-primary mb-3"></i>
                        <h3 class="fw-bold" style="color: var(--text-primary)">加入基沃托斯学园</h3>
                        <p class="text-secondary mb-4">记录学园生活的每一刻美好时光</p>
                    </div>
                    <div class="d-flex gap-3 justify-content-center">
                        <button class="btn-custom btn-primary-custom btn-lg" onclick="showLogin()">
                            <i class="fas fa-sign-in-alt"></i> 立即登录
                        </button>
                        <button class="btn-custom btn-outline-custom btn-lg" onclick="showRegister()">
                            <i class="fas fa-user-plus"></i> 注册账号
                        </button>
                    </div>
                </div>
            `;
        }
    }
}

// 应用初始化
function initializeApp() {
        console.log('初始化博客系统...');
    
    // 加载保存的用户信息 - 修复：优先从localStorage恢复
    const savedToken = localStorage.getItem(CONFIG.TOKEN_KEY);
    const savedUser = localStorage.getItem(CONFIG.USER_KEY);
    
    if (savedToken) {
        STATE.currentToken = savedToken;
        console.log('已恢复token:', savedToken.substring(0, 20) + '...');
    }
    
    if (savedUser) {
        try {
            STATE.currentUser = JSON.parse(savedUser);
            console.log('已恢复用户:', STATE.currentUser?.username || 'unknown');
        } catch (error) {
            console.error('解析用户数据失败:', error);
            localStorage.removeItem(CONFIG.USER_KEY);
        }
    }
    
    // 验证token是否有效
    validateTokenOnStartup();
    // 绑定表单事件 - 确保事件监听器工作
    setTimeout(() => {
        bindFormEvents();
    }, 100);
    
    // 绑定搜索输入框
    const searchInput = document.getElementById('searchKeyword');
    if (searchInput) {
        searchInput.addEventListener('input', function() {
            STATE.searchKeyword = this.value;
            debouncedSearch();
        });
    }
    
    // 绑定分类筛选
    const categoryFilter = document.getElementById('categoryFilter');
    if (categoryFilter) {
        categoryFilter.addEventListener('change', function() {
            STATE.currentCategory = this.value;
            STATE.currentPageNum = 1;
            loadPosts();
        });
    }
    
    // 显示首页
    showPage('home');
    updateNavigation();
    
    // 加载初始数据
    setTimeout(() => {
        loadCategories();
        updateUserCount();
    }, 500);
}

// 绑定表单事件
function bindFormEvents() {
    // 登录表单
    const loginForm = document.getElementById('loginForm');
    if (loginForm) {
        loginForm.addEventListener('submit', handleLogin);
    }
    
    // 注册表单
    const registerForm = document.getElementById('registerForm');
    if (registerForm) {
        registerForm.addEventListener('submit', handleRegister);
    }
    
    // 文章表单
    const postForm = document.getElementById('postForm');
    if (postForm) {
        postForm.addEventListener('submit', handlePostSubmit);
    }
    
    // 分类表单
    const categoryForm = document.getElementById('categoryForm');
    if (categoryForm) {
        categoryForm.addEventListener('submit', handleCategorySubmit);
    }
    
    // 评论表单
    const commentForm = document.getElementById('commentForm');
    if (commentForm) {
        commentForm.addEventListener('submit', handleCommentSubmit);
    }
}

// DOM加载完成后初始化
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeApp);
} else {
    initializeApp();
}

// 防抖搜索函数
let searchTimeout;
function debouncedSearch() {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
        STATE.currentPageNum = 1;
        loadPosts();
    }, CONFIG.DEBOUNCE_DELAY);
}

// Markdown快捷输入
function insertText(text) {
    const textarea = document.getElementById('postContent');
    if (!textarea) return;
    
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const selectedText = textarea.value.substring(start, end);
    
    textarea.value = textarea.value.substring(0, start) + 
                     (selectedText ? text.replace('SELECTED', selectedText) : text) + 
                     textarea.value.substring(end);
    
    textarea.focus();
    textarea.setSelectionRange(start + text.length, start + text.length);
}

// 更新用户统计
async function updateUserCount() {
    try {
        // 简单统计，显示固定数字
        const count = STATE.currentUser ? 1 : 0;
        const countElement = document.getElementById('userCount');
        if (countElement) {
            countElement.textContent = count;
        }
    } catch (error) {
        console.error('更新用户统计失败:', error);
    }
}

// 首页更新
function updateHomePage() {
    const jumbotron = document.querySelector('.jumbotron-custom .jumbotron-header');
    if (jumbotron) {
        if (STATE.currentUser) {
            const userName = STATE.currentUser.name || STATE.currentUser.username;
            const title = jumbotron.querySelector('.display-4');
            const lead = jumbotron.querySelector('.lead');
            if (title) title.textContent = `欢迎回来，${userName}！`;
            if (lead) lead.textContent = '今天有什么新想法要分享吗？';
        }
    }
}

// 加载分类到筛选器
async function loadCategoriesForFilter() {
    try {
        const categories = await getCategories();
        const filter = document.getElementById('categoryFilter');
        if (!filter) return;
        
        filter.innerHTML = '<option value="">全部分类</option>';
        
        categories.forEach(category => {
            const option = document.createElement('option');
            option.value = category.id;
            option.textContent = category.name;
            filter.appendChild(option);
        });
    } catch (error) {
        console.error('加载分类失败:', error);
    }
}

// 加载分类到文章表单
async function loadCategoriesForPost() {
    try {
        const categories = await getCategories();
        const select = document.getElementById('postCategory');
        if (!select) return;
        
        select.innerHTML = '<option value="">请选择分类</option>';
        
        categories.forEach(category => {
            const option = document.createElement('option');
            option.value = category.id;
            option.textContent = category.name;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('加载分类失败:', error);
    }
}

// 加载文章用于编辑
async function loadPostForEdit(postId) {
    try {
        showLoading('postSubmitBtn');
        const post = await getPost(postId);
        
        document.getElementById('postId').value = post.id;
        document.getElementById('postTitle').value = post.title;
        document.getElementById('postContent').value = post.content;
        document.getElementById('postSummary').value = post.summary || '';
        document.getElementById('postCategory').value = post.category_id;
        
        document.getElementById('postSubmitBtn').innerHTML = '<i class="fas fa-save"></i> 更新文章';
        hideLoading('postSubmitBtn');
    } catch (error) {
        hideLoading('postSubmitBtn');
        showMessage('postMessage', `加载文章失败: ${error.message}`, 'danger');
    }
}

// 加载分类用于编辑
async function loadCategoryForEdit(categoryId) {
    try {
        const category = await getCategory(categoryId);
        
        document.getElementById('categoryId').value = category.id;
        document.getElementById('categoryName').value = category.name;
        document.getElementById('categorySlug').value = category.slug || '';
    } catch (error) {
        showMessage('categoryMessage', `加载分类失败: ${error.message}`, 'danger');
    }
}

// 工具函数
function showLoading(buttonId) {
    const button = document.getElementById(buttonId);
    if (!button) return;
    
    const spinner = button.querySelector('.spinner-border');
    if (spinner) {
        spinner.classList.remove('d-none');
    }
    button.disabled = true;
}

function hideLoading(buttonId) {
    const button = document.getElementById(buttonId);
    if (!button) return;
    
    const spinner = button.querySelector('.spinner-border');
    if (spinner) {
        spinner.classList.add('d-none');
    }
    button.disabled = false;
}

function showMessage(elementId, message, type = 'info') {
    const element = document.getElementById(elementId);
    if (!element) {
        console.warn(`元素 ${elementId} 不存在`);
        return;
    }
    
    element.innerHTML = `
        <div class="alert alert-${type} alert-dismissible fade show" role="alert">
            <div class="d-flex align-items-center">
                <i class="fas fa-${type === 'success' ? 'check-circle' : type === 'danger' ? 'exclamation-triangle' : type === 'warning' ? 'exclamation-circle' : 'info-circle'} me-2"></i>
                <div>${message}</div>
            </div>
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        </div>
    `;
    
    setTimeout(() => {
        const alert = element.querySelector('.alert');
        if (alert) {
            try {
                const bsAlert = bootstrap.Alert.getOrCreateInstance(alert);
                bsAlert.close();
            } catch (error) {
                element.innerHTML = '';
            }
        }
    }, 5000);
}

// 检查登录状态
function isLoggedIn() {
    // 确保检查 token 和 user 都存在
    const hasToken = !!STATE.currentToken || !!localStorage.getItem(CONFIG.TOKEN_KEY);
    const hasUser = !!STATE.currentUser || !!localStorage.getItem(CONFIG.USER_KEY);
    
    console.log('isLoggedIn检查:', { hasToken, hasUser });
    
    return hasToken && hasUser;
}

// 获取当前用户ID
function getCurrentUserId() {
    return STATE.currentUser ? STATE.currentUser.id : null;
}

// 全局消息函数
function showGlobalMessage(message, type = 'info', duration = 3000) {
    // 移除现有的全局消息
    const existing = document.getElementById('global-message');
    if (existing) existing.remove();
    
    // 创建消息容器
    const messageDiv = document.createElement('div');
    messageDiv.id = 'global-message';
    messageDiv.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
    messageDiv.style.cssText = `
        top: 20px;
        right: 20px;
        z-index: 9999;
        min-width: 300px;
        max-width: 500px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    `;
    
    const icon = type === 'success' ? 'check-circle' : 
                 type === 'danger' ? 'exclamation-triangle' : 
                 type === 'warning' ? 'exclamation-circle' : 'info-circle';
    
    messageDiv.innerHTML = `
        <div class="d-flex align-items-center">
            <i class="fas fa-${icon} me-2"></i>
            <div>${message}</div>
            <button type="button" class="btn-close ms-auto" data-bs-dismiss="alert"></button>
        </div>
    `;
    
    document.body.appendChild(messageDiv);
    
    // 自动消失
    if (duration > 0) {
        setTimeout(() => {
            if (messageDiv.parentNode) {
                try {
                    const bsAlert = bootstrap.Alert.getOrCreateInstance(messageDiv);
                    bsAlert.close();
                } catch (error) {
                    messageDiv.remove();
                }
            }
        }, duration);
    }
}
async function validateTokenOnStartup() {
    if (!STATE.currentToken) return;
    
    try {
        // 尝试一个简单的API调用来验证token
        await apiCall('/user/profile', 'GET', null, true);
        console.log('token验证成功，用户已登录');
    } catch (error) {
        console.log('token无效或已过期:', error.message);
        // 清除无效的token
        STATE.currentToken = null;
        STATE.currentUser = null;
        localStorage.removeItem(CONFIG.TOKEN_KEY);
        localStorage.removeItem(CONFIG.USER_KEY);
        updateNavigation();
    }
}

// 暴露函数到全局
window.showPage = showPage;
window.showHome = showHome;
window.showLogin = showLogin;
window.showRegister = showRegister;
window.showPosts = showPosts;
window.showPostDetail = showPostDetail;
window.showCreatePost = showCreatePost;
window.showEditPost = showEditPost;
window.showCategories = showCategories;
window.showCreateCategory = showCreateCategory;
window.showEditCategory = showEditCategory;
window.showProfile = showProfile;
window.showFeed = showFeed;
window.showSearch = showSearch;
window.insertText = insertText;
window.logout = logout;