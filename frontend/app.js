/**
 * 小团体博客系统 - 主应用逻辑
 * 专为3-4人小团体设计
 */

// 全局配置
const CONFIG = {
    API_BASE_URL: 'http://localhost:8080/api',
    ITEMS_PER_PAGE: 10,
    DEBOUNCE_DELAY: 500,
    TOKEN_KEY: 'blog_token',
    USER_KEY: 'blog_user',
    // 头像配置
    AVATAR_TYPES: ['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp'],
    MAX_AVATAR_SIZE: 2 * 1024 * 1024 // 2MB
};

// 全局状态
const STATE = {
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
        targetPage.classList.add('active');
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
        }
    } else {
        console.error(`找不到页面: ${pageId}Page`);
    }
    
    updateNavigation();
}

// 快捷页面切换函数
function showHome() { showPage('home'); }
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

// 更新导航栏
function updateNavigation() {
    const userSection = document.getElementById('userSection');
    
    if (STATE.currentUser) {
        // 构建头像显示（如果有头像）
        const avatarHTML = STATE.currentUser.avatar_url ? 
            `<img src="${STATE.currentUser.avatar_url}" alt="${STATE.currentUser.name}" class="navbar-avatar">` :
            `<div class="navbar-default-avatar"><i class="fas fa-user-circle"></i></div>`;
        
        userSection.innerHTML = `
            <div class="dropdown">
                <button class="btn btn-outline-light dropdown-toggle d-flex align-items-center" type="button" data-bs-toggle="dropdown">
                    ${avatarHTML}
                    <span class="fw-bold ms-2" style="color: var(--text-primary)">${STATE.currentUser.name || STATE.currentUser.username}</span>
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
        const userName = STATE.currentUser.name || STATE.currentUser.username;
        document.getElementById('homeActions').innerHTML = `
            <div class="welcome-user-info text-center mb-4">
                <div class="mb-3">
                    ${STATE.currentUser.avatar_url ? 
                        `<img src="${STATE.currentUser.avatar_url}" alt="${userName}" class="home-avatar mb-2" style="width: 80px; height: 80px; border-radius: 50%; object-fit: cover; border: 3px solid var(--primary-color);">` :
                        `<i class="fas fa-user-circle fa-3x text-primary mb-2"></i>`
                    }
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
        
        document.getElementById('homeActions').innerHTML = `
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

// 应用初始化
document.addEventListener('DOMContentLoaded', function() {
    // 加载保存的用户信息
    const savedToken = localStorage.getItem(CONFIG.TOKEN_KEY);
    const savedUser = localStorage.getItem(CONFIG.USER_KEY);
    
    if (savedToken && savedUser) {
        try {
            STATE.currentToken = savedToken;
            STATE.currentUser = JSON.parse(savedUser);
        } catch (error) {
            localStorage.removeItem(CONFIG.TOKEN_KEY);
            localStorage.removeItem(CONFIG.USER_KEY);
            STATE.currentToken = null;
            STATE.currentUser = null;
        }
    }
    
    // 绑定表单事件
    document.getElementById('loginForm').addEventListener('submit', handleLogin);
    document.getElementById('registerForm').addEventListener('submit', handleRegister);
    document.getElementById('postForm').addEventListener('submit', handlePostSubmit);
    document.getElementById('categoryForm').addEventListener('submit', handleCategorySubmit);
    document.getElementById('commentForm').addEventListener('submit', handleCommentSubmit);
    
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
});

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
        const response = await fetch(`${CONFIG.API_BASE_URL}/users/count`);
        if (response.ok) {
            const data = await response.json();
            document.getElementById('userCount').textContent = data.count || 0;
        }
    } catch (error) {
        console.error('获取用户数失败:', error);
    }
}

// 首页更新
function updateHomePage() {
    const jumbotron = document.querySelector('.jumbotron-custom .jumbotron-header');
    if (jumbotron) {
        if (STATE.currentUser) {
            const userName = STATE.currentUser.name || STATE.currentUser.username;
            jumbotron.querySelector('.display-4').textContent = `欢迎回来，${userName}！`;
            jumbotron.querySelector('.lead').textContent = '今天有什么新想法要分享吗？';
        } else {
            jumbotron.querySelector('.display-4').textContent = '欢迎来到基沃托斯学园';
            jumbotron.querySelector('.lead').textContent = '专为学园生活设计的博客系统，记录每一天的成长与故事。';
        }
    }
}

// 加载分类到筛选器
async function loadCategoriesForFilter() {
    try {
        const categories = await getCategories();
        const filter = document.getElementById('categoryFilter');
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
    const spinner = button.querySelector('.spinner-border');
    if (spinner) {
        spinner.classList.remove('d-none');
    }
    button.disabled = true;
}

function hideLoading(buttonId) {
    const button = document.getElementById(buttonId);
    const spinner = button.querySelector('.spinner-border');
    if (spinner) {
        spinner.classList.add('d-none');
    }
    button.disabled = false;
}

function showMessage(elementId, message, type = 'info') {
    const element = document.getElementById(elementId);
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
            const bsAlert = new bootstrap.Alert(alert);
            bsAlert.close();
        }
    }, 5000);
}

// 头像上传函数
async function uploadAvatar(file) {
    if (!file) {
        throw new Error('请选择头像文件');
    }
    
    if (!CONFIG.AVATAR_TYPES.includes(file.type)) {
        throw new Error('只支持 JPG、PNG、GIF、WebP 格式的图片');
    }
    
    if (file.size > CONFIG.MAX_AVATAR_SIZE) {
        throw new Error('头像大小不能超过 2MB');
    }
    
    const formData = new FormData();
    formData.append('avatar', file);
    
    const url = `${CONFIG.API_BASE_URL}/user/avatar`;
    const headers = {
        'Authorization': `Bearer ${STATE.currentToken}`
    };
    
    try {
        const response = await fetch(url, {
            method: 'POST',
            headers: headers,
            body: formData
        });
        
        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || '上传失败');
        }
        
        const data = await response.json();
        return data;
    } catch (error) {
        console.error('上传头像失败:', error);
        throw error;
    }
}

// 删除头像
async function deleteAvatar() {
    if (!confirm('确定要删除头像吗？')) {
        return;
    }
    
    try {
        const response = await apiCall('/user/avatar', 'DELETE', null, true);
        
        if (response.success) {
            if (STATE.currentUser) {
                STATE.currentUser.avatar_url = '';
                localStorage.setItem(CONFIG.USER_KEY, JSON.stringify(STATE.currentUser));
                updateNavigation();
            }
            return true;
        }
        return false;
    } catch (error) {
        console.error('删除头像失败:', error);
        throw error;
    }
}

// 检查登录状态
function isLoggedIn() {
    return !!STATE.currentToken && !!STATE.currentUser;
}

// 获取当前用户ID
function getCurrentUserId() {
    return STATE.currentUser ? STATE.currentUser.id : null;
}