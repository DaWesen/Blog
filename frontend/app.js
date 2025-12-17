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
    USER_KEY: 'blog_user'
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
    console.log(`切换到页面: ${pageId}，当前用户:`, STATE.currentUser ? STATE.currentUser.name : '未登录');
    
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
                    console.log('加载文章详情，ID:', STATE.currentPostId);
                    loadPostDetail(STATE.currentPostId);
                    // 延迟加载评论，确保文章内容先显示
                    setTimeout(() => {
                        loadPostComments(STATE.currentPostId);
                    }, 300);
                } else {
                    showPosts();
                }
                break;
            case 'login':
                // 如果已登录，跳转到首页
                if (isLoggedIn()) {
                    showHome();
                }
                break;
            case 'register':
                // 如果已登录，跳转到首页
                if (isLoggedIn()) {
                    showHome();
                }
                break;
            case 'profile':
                console.log('加载个人资料页面');
                if (isLoggedIn()) {
                    // 直接调用 loadUserProfile 函数，它会重新渲染整个页面
                    loadUserProfile();
                } else {
                    console.log('用户未登录，跳转到登录页面');
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

// 更新导航栏 - 增强用户信息显示
function updateNavigation() {
    const userSection = document.getElementById('userSection');
    
    if (STATE.currentUser) {
        // 增强的用户显示，包含清晰的用户名
        userSection.innerHTML = `
            <div class="dropdown">
                <button class="btn btn-outline-light dropdown-toggle" type="button" data-bs-toggle="dropdown">
                    <i class="fas fa-user-circle me-2"></i>
                    <span class="fw-bold" style="color: var(--text-primary)">${STATE.currentUser.name || STATE.currentUser.username}</span>
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
        
        // 更新首页按钮 - 增强欢迎信息
        const userName = STATE.currentUser.name || STATE.currentUser.username;
        document.getElementById('homeActions').innerHTML = `
            <div class="welcome-user-info text-center mb-4">
                <div class="mb-3">
                    <i class="fas fa-user-circle fa-3x text-primary mb-2"></i>
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
    console.log('应用初始化开始...');
    
    // 检查 marked 库是否已加载
    if (typeof marked === 'undefined') {
        console.warn('marked.js 未加载，文章内容将使用纯文本显示');
    }
    
    // 加载保存的用户信息
    const savedToken = localStorage.getItem(CONFIG.TOKEN_KEY);
    const savedUser = localStorage.getItem(CONFIG.USER_KEY);
    
    if (savedToken && savedUser) {
        try {
            STATE.currentToken = savedToken;
            STATE.currentUser = JSON.parse(savedUser);
            console.log('从本地存储恢复用户:', STATE.currentUser);
        } catch (error) {
            console.error('解析用户信息失败:', error);
            localStorage.removeItem(CONFIG.TOKEN_KEY);
            localStorage.removeItem(CONFIG.USER_KEY);
            STATE.currentToken = null;
            STATE.currentUser = null;
        }
    } else {
        console.log('本地存储中没有用户信息');
    }
    
    // 检查DOM元素是否存在
    const requiredElements = [
        'loginForm', 'registerForm', 'postForm', 
        'categoryForm', 'profileForm', 'commentForm'
    ];
    
    requiredElements.forEach(id => {
        const element = document.getElementById(id);
        if (!element) {
            console.warn(`找不到元素: #${id}`);
        }
    });
    
    // 绑定表单事件
    try {
        document.getElementById('loginForm').addEventListener('submit', handleLogin);
        document.getElementById('registerForm').addEventListener('submit', handleRegister);
        document.getElementById('postForm').addEventListener('submit', handlePostSubmit);
        document.getElementById('categoryForm').addEventListener('submit', handleCategorySubmit);
        document.getElementById('commentForm').addEventListener('submit', handleCommentSubmit);
        console.log('表单事件绑定完成');
    } catch (error) {
        console.error('绑定表单事件失败:', error);
    }
    
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
    console.log('显示首页...');
    showPage('home');
    updateNavigation();
    
    // 加载初始数据
    setTimeout(() => {
        loadCategories();
        updateUserCount();
    }, 500);
    
    console.log('应用初始化完成');
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

// 首页更新 - 增强欢迎信息
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
    
    // 5秒后自动消失
    setTimeout(() => {
        const alert = element.querySelector('.alert');
        if (alert) {
            const bsAlert = new bootstrap.Alert(alert);
            bsAlert.close();
        }
    }, 5000);
}

// 调试工具函数
function debugAPI(url) {
    console.group('API调试信息');
    console.log('请求URL:', CONFIG.API_BASE_URL + url);
    console.log('当前用户:', STATE.currentUser);
    console.log('当前Token:', STATE.currentToken ? '已设置' : '未设置');
    console.groupEnd();
}

// 在页面切换时添加调试
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
        
        console.log('切换到页面:', pageId);
        
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
                loadCategories();
                break;
            case 'postDetail':
                if (STATE.currentPostId) {
                    console.log('加载文章详情，ID:', STATE.currentPostId);
                    debugAPI(`/posts/${STATE.currentPostId}`);
                    loadPostDetail(STATE.currentPostId);
                    
                    // 延迟加载评论，确保文章内容先显示
                    setTimeout(() => {
                        console.log('开始加载评论，文章ID:', STATE.currentPostId);
                        debugAPI(`/posts/${STATE.currentPostId}/comments`);
                        loadPostComments(STATE.currentPostId);
                    }, 300);
                }
                break;
            case 'profile':
                loadUserProfile();
                break;
        }
    }
    
    updateNavigation();
}

// 调试工具：检查API响应
function debugAPICall(url, method = 'GET', data = null) {
    console.group('API调用调试');
    console.log('URL:', CONFIG.API_BASE_URL + url);
    console.log('Method:', method);
    console.log('Data:', data);
    console.log('Token:', STATE.currentToken ? '已设置' : '未设置');
    console.log('User:', STATE.currentUser);
    
    fetch(CONFIG.API_BASE_URL + url, {
        method: method,
        headers: {
            'Content-Type': 'application/json',
            'Authorization': STATE.currentToken ? `Bearer ${STATE.currentToken}` : ''
        },
        body: data ? JSON.stringify(data) : null
    })
    .then(response => {
        console.log('Status:', response.status, response.statusText);
        console.log('Headers:', Object.fromEntries(response.headers.entries()));
        
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
            return response.json().then(data => {
                console.log('Response JSON:', data);
            });
        } else {
            return response.text().then(text => {
                console.log('Response Text:', text || '(空)');
            });
        }
    })
    .catch(error => {
        console.error('Fetch Error:', error);
    })
    .finally(() => {
        console.groupEnd();
    });
}

// 在控制台暴露调试函数
window.debugAPI = debugAPICall;

function checkPageElements() {
    const pages = [
        'homePage', 'loginPage', 'registerPage', 'postsPage', 
        'editPostPage', 'postDetailPage', 'categoriesPage', 
        'editCategoryPage', 'profilePage'
    ];
    
    console.group('页面元素检查');
    pages.forEach(pageId => {
        const element = document.getElementById(pageId);
        console.log(`${pageId}: ${element ? '✓ 存在' : '✗ 不存在'}`);
    });
    console.groupEnd();
}

// 检查导航按钮
function checkNavigation() {
    console.group('导航按钮检查');
    const navLinks = document.querySelectorAll('.nav-link');
    navLinks.forEach(link => {
        console.log(`导航: ${link.textContent.trim()} - ${link.onclick ? '有事件' : '无事件'}`);
    });
    console.groupEnd();
}

// 在控制台暴露检查函数
window.checkPages = checkPageElements;
window.checkNav = checkNavigation;