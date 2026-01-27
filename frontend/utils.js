/**
 * 工具函数模块 - 通用工具和辅助函数
 * 修复版本：修复API调用和错误处理
 */

// API调用函数
async function apiCall(endpoint, method = 'GET', data = null, requiresAuth = false) {
    if (!endpoint.startsWith('/')) {
        endpoint = '/' + endpoint;
    }
    
    const url = `${CONFIG.API_BASE_URL}${endpoint}`;
    console.log('API调用:', {
        url,
        method,
        requiresAuth,
        currentToken: STATE.currentToken ? '有token' : '无token'
    });
    
    const headers = {
        'Content-Type': 'application/json',
    };
    
    // 添加认证头
    if (requiresAuth) {
        // 首先尝试使用 STATE.currentToken
        let token = STATE.currentToken;
        
        // 如果 STATE.currentToken 为空，尝试从本地存储获取
        if (!token) {
            token = localStorage.getItem(CONFIG.TOKEN_KEY);
            console.log('从localStorage获取token:', token ? '有token' : '无token');
            if (token) {
                STATE.currentToken = token; // 更新到STATE
            }
        }
        
        if (!token) {
            console.error('未找到token，用户未登录');
            throw new Error('用户未登录');
        }
        
        console.log('使用的token:', token.substring(0, 20) + '...'); // 只显示前20个字符
        headers['Authorization'] = `Bearer ${token}`;
    }
    
    const options = {
        method,
        headers,
    };
    
    if (data && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
        options.body = JSON.stringify(data);
    }
    
    try {
        const response = await fetch(url, options);
        
        // 处理204 No Content（删除、点赞等操作通常返回204）
        if (response.status === 204) {
            return { success: true, message: '操作成功' };
        }
        
        // 处理其他响应
        const responseText = await response.text();
        
        // 尝试解析JSON
        let responseData;
        try {
            responseData = responseText ? JSON.parse(responseText) : {};
        } catch (parseError) {
            // 如果不是JSON，返回原始文本
            responseData = { message: responseText };
        }
        
        if (!response.ok) {
            // 处理认证错误
            if (response.status === 401) {
                // 清除登录状态
                STATE.currentToken = null;
                STATE.currentUser = null;
                localStorage.removeItem(CONFIG.TOKEN_KEY);
                localStorage.removeItem(CONFIG.USER_KEY);
                updateNavigation();
                
                if (STATE.currentPage !== 'login') {
                    showGlobalMessage('登录已过期，请重新登录', 'warning');
                    setTimeout(() => showLogin(), 1000);
                }
            }
            
            const errorMessage = responseData.error || 
                               responseData.msg || 
                               responseData.message || 
                               responseData.details ||
                               `请求失败: ${response.status}`;
            
            throw new Error(errorMessage);
        }
        
        // 成功响应，返回数据
        return responseData;
        
    } catch (error) {
        console.error(`API调用失败 ${method} ${endpoint}:`, error);
        
        // 检查是否是网络错误
        if (error.name === 'TypeError' && error.message.includes('Failed to fetch')) {
            throw new Error('网络连接失败，请检查网络连接');
        }
        
        throw error;
    }
}

// 格式化日期
function formatDate(dateString) {
    if (!dateString) return '';
    
    try {
        const date = new Date(dateString);
        const now = new Date();
        const diff = now - date;
        
        // 小于1分钟
        if (diff < 60000) {
            return '刚刚';
        }
        
        // 小于1小时
        if (diff < 3600000) {
            return `${Math.floor(diff / 60000)}分钟前`;
        }
        
        // 小于1天
        if (diff < 86400000) {
            return `${Math.floor(diff / 3600000)}小时前`;
        }
        
        // 小于7天
        if (diff < 604800000) {
            return `${Math.floor(diff / 86400000)}天前`;
        }
        
        // 显示完整日期
        return date.toLocaleDateString('zh-CN', {
            year: 'numeric',
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    } catch (error) {
        return dateString;
    }
}

// HTML转义
function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// 截断文本
function truncateText(text, maxLength) {
    if (!text || text.length <= maxLength) {
        return text;
    }
    
    // 尝试在标点符号处截断
    const truncated = text.substr(0, maxLength);
    const lastPunctuation = Math.max(
        truncated.lastIndexOf('。'),
        truncated.lastIndexOf('！'),
        truncated.lastIndexOf('？'),
        truncated.lastIndexOf('.'),
        truncated.lastIndexOf('!'),
        truncated.lastIndexOf('?'),
        truncated.lastIndexOf('，'),
        truncated.lastIndexOf(',')
    );
    
    if (lastPunctuation > maxLength * 0.7) {
        return truncated.substr(0, lastPunctuation + 1) + '...';
    }
    
    return truncated + '...';
}

// 生成随机ID
function generateId() {
    return Date.now().toString(36) + Math.random().toString(36).substr(2);
}

// 复制文本到剪贴板
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showGlobalMessage('已复制到剪贴板', 'success');
    }).catch(err => {
        console.error('复制失败:', err);
        showGlobalMessage('复制失败', 'danger');
    });
}

// 防抖函数
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// 节流函数
function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
        if (!inThrottle) {
            func.apply(this, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// 验证邮箱格式
function isValidEmail(email) {
    const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return re.test(email);
}

// 验证URL格式
function isValidUrl(url) {
    try {
        new URL(url);
        return true;
    } catch (_) {
        return false;
    }
}

// 获取URL参数
function getUrlParam(name) {
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get(name);
}

// 设置URL参数
function setUrlParam(name, value) {
    const url = new URL(window.location);
    url.searchParams.set(name, value);
    window.history.pushState({}, '', url);
}

// 移除URL参数
function removeUrlParam(name) {
    const url = new URL(window.location);
    url.searchParams.delete(name);
    window.history.pushState({}, '', url);
}

// 显示全局消息
function showGlobalMessage(message, type = 'info', duration = 5000) {
    // 移除现有的全局消息
    const existing = document.getElementById('global-message');
    if (existing) existing.remove();
    
    // 创建新的消息
    const messageDiv = document.createElement('div');
    messageDiv.id = 'global-message';
    messageDiv.className = `alert alert-${type} alert-dismissible fade show position-fixed`;
    messageDiv.style.cssText = `
        top: 80px;
        right: 20px;
        z-index: 9999;
        min-width: 300px;
        box-shadow: 0 4px 12px rgba(0,0,0,0.15);
    `;
    
    messageDiv.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
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

// 检查元素是否存在
function elementExists(id) {
    return document.getElementById(id) !== null;
}

// 安全设置元素内容
function safeSetText(id, text) {
    const element = document.getElementById(id);
    if (element) {
        element.textContent = text;
    } else {
        console.warn(`元素 #${id} 不存在`);
    }
}

function safeSetValue(id, value) {
    const element = document.getElementById(id);
    if (element) {
        element.value = value;
    } else {
        console.warn(`元素 #${id} 不存在`);
    }
}

// 获取分类
async function getCategories() {
    try {
        const response = await apiCall('/categories/all');
        return Array.isArray(response) ? response : [];
    } catch (error) {
        console.error('获取分类失败:', error);
        return [];
    }
}

// 获取单个分类
async function getCategory(categoryId) {
    try {
        return await apiCall(`/categories/${categoryId}`);
    } catch (error) {
        console.error('获取分类失败:', error);
        throw error;
    }
}

// 获取文章
async function getPost(postId) {
    try {
        return await apiCall(`/posts/${postId}`);
    } catch (error) {
        console.error('获取文章失败:', error);
        throw error;
    }
}

// 在控制台暴露辅助函数
window.elementExists = elementExists;
window.safeSetText = safeSetText;
window.safeSetValue = safeSetValue;
window.apiCall = apiCall;
window.formatDate = formatDate;
window.escapeHtml = escapeHtml;
window.truncateText = truncateText;
window.showGlobalMessage = showGlobalMessage;