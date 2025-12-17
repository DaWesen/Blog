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