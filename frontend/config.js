const API_CONFIG = {
    // 分类接口
    CATEGORIES: {
        LIST: '/api/categories/all',  // 使用纯数组格式
        // 或者使用分页格式
        // LIST: '/api/categories?no_pagination=true'
    },
    
    // 统计接口
    STATS: {
        USER_COUNT: '/api/stats/users/count'
    }
};
window.CONFIG = {
    API_BASE_URL: 'http://localhost:8080/api',
    ITEMS_PER_PAGE: 10,
    DEBOUNCE_DELAY: 500,
    TOKEN_KEY: 'blog_token',
    USER_KEY: 'blog_user',
    AVATAR_TYPES: ['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp'],
    MAX_AVATAR_SIZE: 2 * 1024 * 1024
};