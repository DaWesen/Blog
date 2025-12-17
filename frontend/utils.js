/**
 * 工具函数模块 - 通用工具和辅助函数
 */

// API调用函数
async function apiCall(endpoint, method = 'GET', data = null, requiresAuth = false) {
    const url = `${CONFIG.API_BASE_URL}${endpoint}`;
    const headers = {
        'Content-Type': 'application/json',
    };
    
    // 添加认证头
    if (requiresAuth) {
        if (!STATE.currentToken) {
            // 尝试从本地存储获取
            const savedToken = localStorage.getItem(CONFIG.TOKEN_KEY);
            if (!savedToken) {
                throw new Error('用户未登录');
            }
            STATE.currentToken = savedToken;
        }
        headers['Authorization'] = `Bearer ${STATE.currentToken}`;
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
        
        // 记录API调用（开发环境）
        if (window.location.hostname === 'localhost') {
            console.log(`[API] ${method} ${endpoint}`, {
                status: response.status,
                statusText: response.statusText
            });
        }
        
        // 处理204 No Content（删除、点赞等操作通常返回204）
        if (response.status === 204) {
            console.log(`[API] ${method} ${endpoint}: 操作成功 (204 No Content)`);
            return { success: true, message: '操作成功' };
        }
        
        // 处理空响应
        const contentType = response.headers.get('content-type');
        if (!contentType || !contentType.includes('application/json')) {
            // 不是JSON响应
            if (response.ok) {
                console.log(`[API] ${method} ${endpoint}: 操作成功 (非JSON响应)`);
                return { success: true, message: '操作成功' };
            } else {
                throw new Error(`请求失败: ${response.status} ${response.statusText}`);
            }
        }
        
        // 解析JSON响应
        let responseData;
        try {
            responseData = await response.json();
        } catch (parseError) {
            console.warn(`[API] ${method} ${endpoint}: JSON解析失败，但状态码为${response.status}`);
            if (response.ok) {
                return { success: true };
            }
            throw new Error('服务器响应格式错误');
        }
        
        // 记录完整响应数据
        if (window.location.hostname === 'localhost') {
            console.log(`[API] ${method} ${endpoint} response:`, responseData);
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
                    showMessage('globalMessage', '登录已过期，请重新登录', 'warning');
                    setTimeout(() => showLogin(), 1000);
                }
            }
            
            // 从响应数据中提取错误信息
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
        console.error(`[API Error] ${method} ${endpoint}:`, error);
        
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
}

// HTML转义
function escapeHtml(text) {
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
        showMessage('globalMessage', '已复制到剪贴板', 'success');
    }).catch(err => {
        console.error('复制失败:', err);
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
                const bsAlert = bootstrap.Alert.getOrCreateInstance(messageDiv);
                bsAlert.close();
            }
        }, duration);
    }
}

// 显示消息的简写函数
function showMessage(elementId, message, type = 'info') {
    const element = document.getElementById(elementId);
    if (element) {
        element.innerHTML = `
            <div class="alert alert-${type} alert-dismissible fade show" role="alert">
                ${message}
                <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
            </div>
        `;
        
        // 5秒后自动消失
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
}

// 在HTML中增加全局消息容器
document.addEventListener('DOMContentLoaded', function() {
    if (!document.getElementById('global-message-container')) {
        const container = document.createElement('div');
        container.id = 'global-message-container';
        container.style.cssText = 'position: fixed; top: 80px; right: 20px; z-index: 9999;';
        document.body.appendChild(container);
    }
});

// Marked.js 配置
if (typeof marked !== 'undefined') {
    marked.setOptions({
        breaks: true,
        gfm: true,
        highlight: function(code, lang) {
            if (window.hljs) {
                const language = hljs.getLanguage(lang) ? lang : 'plaintext';
                return hljs.highlight(code, { language }).value;
            }
            return code;
        }
    });
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

// 在控制台暴露辅助函数
window.elementExists = elementExists;
window.safeSetText = safeSetText;
window.safeSetValue = safeSetValue;