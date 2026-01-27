// config.js - 修复版
console.log('config.js 正在加载...');

// 确保全局变量存在
if (!window.CONFIG) {
    console.log('创建全局CONFIG对象');
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

// 确保STATE存在
if (!window.STATE) {
    console.log('创建全局STATE对象');
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

console.log('config.js 加载完成');
console.log('CONFIG:', window.CONFIG);
console.log('STATE:', window.STATE);