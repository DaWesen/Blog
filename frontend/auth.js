/**
 * 认证模块 - 处理用户登录、注册、Token管理等
 */

// 检查用户名可用性（防抖）
let usernameTimeout;
async function checkUsername(username) {
    clearTimeout(usernameTimeout);
    usernameTimeout = setTimeout(async () => {
        if (username.length < 2) return;
        
        const feedback = document.getElementById('usernameFeedback');
        feedback.innerHTML = '<small class="text-muted"><i class="fas fa-spinner fa-spin"></i> 检查中...</small>';
        
        try {
            const data = await apiCall(`/check-username?username=${encodeURIComponent(username)}`);
            if (data.exists) {
                feedback.innerHTML = '<small class="text-danger"><i class="fas fa-times-circle"></i> 用户名已存在</small>';
            } else {
                feedback.innerHTML = '<small class="text-success"><i class="fas fa-check-circle"></i> 用户名可用</small>';
            }
        } catch (error) {
            feedback.innerHTML = `<small class="text-warning"><i class="fas fa-exclamation-triangle"></i> ${error.message}</small>`;
        }
    }, CONFIG.DEBOUNCE_DELAY);
}

// 检查邮箱可用性（防抖）
let emailTimeout;
async function checkEmail(email) {
    clearTimeout(emailTimeout);
    emailTimeout = setTimeout(async () => {
        if (email.length < 5 || !email.includes('@')) return;
        
        const feedback = document.getElementById('emailFeedback');
        feedback.innerHTML = '<small class="text-muted"><i class="fas fa-spinner fa-spin"></i> 检查中...</small>';
        
        try {
            const data = await apiCall(`/check-email?email=${encodeURIComponent(email)}`);
            if (data.exists) {
                feedback.innerHTML = '<small class="text-danger"><i class="fas fa-times-circle"></i> 邮箱已注册</small>';
            } else {
                feedback.innerHTML = '<small class="text-success"><i class="fas fa-check-circle"></i> 邮箱可用</small>';
            }
        } catch (error) {
            feedback.innerHTML = `<small class="text-warning"><i class="fas fa-exclamation-triangle"></i> ${error.message}</small>`;
        }
    }, CONFIG.DEBOUNCE_DELAY);
}

// 处理登录
async function handleLogin(e) {
    e.preventDefault();
    
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;
    
    if (!username || !password) {
        showMessage('loginMessage', '请输入用户名和密码', 'warning');
        return;
    }
    
    const submitBtn = e.target.querySelector('button[type="submit"]');
    const originalText = submitBtn.innerHTML;
    
    try {
        // 显示登录中状态
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 登录中...';
        showMessage('loginMessage', '正在登录，请稍候...', 'info');
        
        const data = await apiCall('/login', 'POST', {
            username_or_email: username,
            password: password
        });
        
        // 保存用户信息和Token
        STATE.currentToken = data.token;
        STATE.currentUser = data.user;
        
        localStorage.setItem(CONFIG.TOKEN_KEY, data.token);
        localStorage.setItem(CONFIG.USER_KEY, JSON.stringify(data.user));
        
        showMessage('loginMessage', '登录成功！', 'success');
        showGlobalMessage('登录成功！', 'success', 2000);
        updateNavigation();
        
        setTimeout(() => {
            showHome();
        }, 1500);
        
    } catch (error) {
        console.error('登录失败:', error);
        showMessage('loginMessage', `登录失败: ${error.message}`, 'danger');
    } finally {
        // 恢复按钮状态
        submitBtn.disabled = false;
        submitBtn.innerHTML = originalText;
    }
}

// 处理注册
async function handleRegister(e) {
    e.preventDefault();
    
    const username = document.getElementById('registerUsername').value;
    const email = document.getElementById('registerEmail').value;
    const password = document.getElementById('registerPassword').value;
    const bio = document.getElementById('registerBio').value;
    
    // 表单验证
    if (!username || username.length < 2) {
        showMessage('registerMessage', '用户名至少需要2个字符', 'warning');
        return;
    }
    
    if (!email || !email.includes('@')) {
        showMessage('registerMessage', '请输入有效的邮箱地址', 'warning');
        return;
    }
    
    if (!password || password.length < 6) {
        showMessage('registerMessage', '密码至少需要6位', 'warning');
        return;
    }
    
    const submitBtn = e.target.querySelector('button[type="submit"]');
    const originalText = submitBtn.innerHTML;
    
    try {
        // 显示注册中状态
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 注册中...';
        showMessage('registerMessage', '正在注册，请稍候...', 'info');
        
        // 注册用户
        const data = await apiCall('/register', 'POST', {
            username: username,
            email: email,
            password: password,
            bio: bio || ''
        });
        
        showMessage('registerMessage', '注册成功！请登录。', 'success');
        showGlobalMessage('注册成功！请登录。', 'success', 3000);
        
        // 清空表单
        document.getElementById('registerForm').reset();
        document.getElementById('usernameFeedback').innerHTML = '';
        document.getElementById('emailFeedback').innerHTML = '';
        
        setTimeout(() => {
            showLogin();
        }, 2000);
        
    } catch (error) {
        console.error('注册失败:', error);
        showMessage('registerMessage', `注册失败: ${error.message}`, 'danger');
    } finally {
        // 恢复按钮状态
        submitBtn.disabled = false;
        submitBtn.innerHTML = originalText;
    }
}

// 处理个人资料更新
async function handleProfileUpdate(e) {
    e.preventDefault();
    
    const name = document.getElementById('profileName').value;
    const bio = document.getElementById('profileBio').value;
    const avatar = document.getElementById('profileAvatar').value;
    
    const submitBtn = e.target.querySelector('button[type="submit"]');
    const originalText = submitBtn.innerHTML;
    
    try {
        // 显示更新中状态
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 更新中...';
        showMessage('profileMessage', '更新中...', 'info');
        
        const updateData = {};
        if (name && name.trim()) updateData.name = name.trim();
        if (bio && bio.trim()) updateData.bio = bio.trim();
        if (avatar && avatar.trim()) updateData.avatar_url = avatar.trim();
        
        const data = await apiCall('/user/profile', 'PUT', updateData, true);
        
        // 更新本地用户信息
        STATE.currentUser = data;
        localStorage.setItem(CONFIG.USER_KEY, JSON.stringify(data));
        
        showMessage('profileMessage', '资料更新成功！', 'success');
        showGlobalMessage('资料更新成功！', 'success', 3000);
        updateNavigation();
        
    } catch (error) {
        console.error('更新资料失败:', error);
        showMessage('profileMessage', `更新失败: ${error.message}`, 'danger');
    } finally {
        // 恢复按钮状态
        submitBtn.disabled = false;
        submitBtn.innerHTML = originalText;
    }
}

// 加载用户资料 - 增强显示效果
async function loadUserProfile() {
    console.log('开始加载用户资料...');
    
    const container = document.getElementById('profilePage');
    if (!container) {
        console.error('找不到个人资料页面容器');
        return;
    }
    
    // 显示加载中
    const contentArea = container.querySelector('.card-body');
    if (contentArea) {
        const loadingHtml = `
            <div class="text-center py-5">
                <div class="spinner-border text-primary" style="width: 3rem; height: 3rem;" role="status">
                    <span class="visually-hidden">加载中...</span>
                </div>
                <p class="mt-3 fw-bold" style="color: var(--text-primary)">正在加载个人资料...</p>
            </div>
        `;
        contentArea.innerHTML = loadingHtml;
    }
    
    try {
        console.log('调用用户资料API...');
        const user = await apiCall('/user/profile', 'GET', null, true);
        console.log('用户资料API返回:', user);
        
        // 重新渲染整个个人资料页面内容
        if (contentArea) {
            contentArea.innerHTML = `
                <div class="text-center mb-4">
                    <div class="avatar-placeholder mb-3">
                        ${user.avatar_url || user.avatar ? 
                            `<img src="${user.avatar_url || user.avatar}" alt="${user.name}" 
                                  style="width:120px;height:120px;border-radius:50%;object-fit:cover; border: 4px solid white; box-shadow: 0 8px 25px rgba(0,0,0,0.2);">` :
                            `<div style="width:120px;height:120px;border-radius:50%;background:linear-gradient(135deg, var(--primary-color), var(--secondary-color)); 
                              display:flex;align-items:center;justify-content:center;color:white;font-size:3rem;margin:0 auto;border:4px solid white;box-shadow:0 8px 25px rgba(0,0,0,0.2);">
                                <i class="fas fa-user"></i>
                             </div>`
                        }
                    </div>
                    <h3 class="fw-bold mb-2" style="color: var(--text-primary)">${escapeHtml(user.name || user.username || '未设置')}</h3>
                    <p class="text-secondary mb-0"><i class="fas fa-envelope me-2"></i>${escapeHtml(user.email || '未设置')}</p>
                    ${user.bio ? `<p class="mt-3 text-muted"><i class="fas fa-quote-left me-2"></i>${escapeHtml(user.bio)}</p>` : ''}
                </div>
                
                <div class="card mb-4" style="border: 2px solid var(--border-color); border-radius: var(--border-radius);">
                    <div class="card-header bg-light fw-bold" style="color: var(--text-primary); border-bottom: 2px solid var(--border-color);">
                        <i class="fas fa-edit me-2"></i>编辑个人资料
                    </div>
                    <div class="card-body">
                        <form id="profileForm">
                            <div class="mb-3">
                                <label for="profileName" class="form-label fw-bold" style="color: var(--text-primary)">
                                    <i class="fas fa-user me-2"></i>显示名称
                                </label>
                                <input type="text" class="form-control" id="profileName" value="${escapeHtml(user.name || '')}"
                                       style="border: 2px solid var(--border-color);">
                            </div>
                            
                            <div class="mb-3">
                                <label for="profileBio" class="form-label fw-bold" style="color: var(--text-primary)">
                                    <i class="fas fa-comment-dots me-2"></i>个人简介
                                </label>
                                <textarea class="form-control" id="profileBio" rows="4" 
                                          style="border: 2px solid var(--border-color);">${escapeHtml(user.bio || '')}</textarea>
                            </div>
                            
                            <div class="mb-4">
                                <label for="profileAvatar" class="form-label fw-bold" style="color: var(--text-primary)">
                                    <i class="fas fa-image me-2"></i>头像URL
                                </label>
                                <input type="url" class="form-control" id="profileAvatar" 
                                       placeholder="https://example.com/avatar.jpg" value="${escapeHtml(user.avatar_url || user.avatar || '')}"
                                       style="border: 2px solid var(--border-color);">
                                <div class="form-text text-muted mt-1">支持JPG、PNG等图片格式链接</div>
                            </div>
                            
                            <div class="d-grid gap-2">
                                <button type="submit" class="btn-custom btn-primary-custom">
                                    <i class="fas fa-save me-2"></i>更新资料
                                </button>
                                <button type="button" class="btn-custom btn-outline-custom" onclick="logout()">
                                    <i class="fas fa-sign-out-alt me-2"></i>退出登录
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
                <div id="profileMessage" class="mt-3"></div>
            `;
            
            // 重新绑定表单提交事件
            const profileForm = document.getElementById('profileForm');
            if (profileForm) {
                // 移除旧的事件监听器
                const newForm = profileForm.cloneNode(true);
                profileForm.parentNode.replaceChild(newForm, profileForm);
                newForm.addEventListener('submit', handleProfileUpdate);
            }
        }
        
        console.log('用户资料加载完成');
        
    } catch (error) {
        console.error('加载用户资料失败:', error);
        
        // 显示错误信息
        if (contentArea) {
            contentArea.innerHTML = `
                <div class="alert alert-danger" style="border: 2px solid var(--danger-color); border-radius: var(--border-radius);">
                    <div class="d-flex align-items-center">
                        <i class="fas fa-exclamation-triangle fa-2x me-3"></i>
                        <div>
                            <h5 class="fw-bold mb-2">加载个人资料失败</h5>
                            <p class="mb-3">${error.message || '未知错误'}</p>
                            <div class="d-flex gap-2">
                                <button class="btn-custom btn-primary-custom" onclick="loadUserProfile()">
                                    <i class="fas fa-redo me-2"></i>重试
                                </button>
                                <button class="btn-custom btn-outline-custom" onclick="showHome()">
                                    <i class="fas fa-home me-2"></i>返回首页
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }
    }
}


// 退出登录
function logout() {
    if (confirm('确定要退出登录吗？')) {
        STATE.currentToken = null;
        STATE.currentUser = null;
        
        localStorage.removeItem(CONFIG.TOKEN_KEY);
        localStorage.removeItem(CONFIG.USER_KEY);
        
        updateNavigation();
        showHome();
        
        // 显示退出成功提示
        showGlobalMessage('已成功退出登录。', 'info', 3000);
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