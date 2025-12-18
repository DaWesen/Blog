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
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 登录中...';
        showMessage('loginMessage', '正在登录，请稍候...', 'info');
        
        const data = await apiCall('/login', 'POST', {
            username_or_email: username,
            password: password
        });
        
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
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 注册中...';
        showMessage('registerMessage', '正在注册，请稍候...', 'info');
        
        const data = await apiCall('/register', 'POST', {
            username: username,
            email: email,
            password: password,
            bio: bio || ''
        });
        
        showMessage('registerMessage', '注册成功！请登录。', 'success');
        showGlobalMessage('注册成功！请登录。', 'success', 3000);
        
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
    const email = document.getElementById('profileEmail').value;
    
    const submitBtn = e.target.querySelector('button[type="submit"]');
    const originalText = submitBtn.innerHTML;
    
    try {
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 更新中...';
        showMessage('profileMessage', '更新中...', 'info');
        
        const updateData = {};
        if (name && name.trim()) updateData.name = name.trim();
        if (bio && bio.trim()) updateData.bio = bio.trim();
        if (avatar && avatar.trim()) updateData.avatar_url = avatar.trim();
        if (email && email.trim()) updateData.email = email.trim();
        
        const data = await apiCall('/user/profile', 'PUT', updateData, true);
        
        STATE.currentUser = data;
        localStorage.setItem(CONFIG.USER_KEY, JSON.stringify(data));
        
        showMessage('profileMessage', '资料更新成功！', 'success');
        showGlobalMessage('资料更新成功！', 'success', 3000);
        updateNavigation();
        
        setTimeout(() => {
            loadUserProfile();
        }, 500);
        
    } catch (error) {
        console.error('更新资料失败:', error);
        showMessage('profileMessage', `更新失败: ${error.message}`, 'danger');
    } finally {
        submitBtn.disabled = false;
        submitBtn.innerHTML = originalText;
    }
}

// 加载用户资料 - 包含头像管理功能
async function loadUserProfile() {
    const container = document.getElementById('profilePage');
    if (!container) {
        console.error('找不到个人资料页面容器');
        return;
    }
    
    container.innerHTML = `
        <div class="row justify-content-center">
            <div class="col-md-10">
                <div class="profile-card">
                    <div class="profile-card-body">
                        <div class="text-center py-5">
                            <div class="spinner-border text-primary" style="width: 3rem; height: 3rem;" role="status">
                                <span class="visually-hidden">加载中...</span>
                            </div>
                            <p class="mt-3 fw-bold" style="color: var(--text-primary); font-size: 1.2rem;">正在加载个人资料...</p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    try {
        const user = await apiCall('/user/profile', 'GET', null, true);
        
        container.innerHTML = `
            <div class="row justify-content-center">
                <div class="col-md-10">
                    <!-- 用户信息概览卡片 -->
                    <div class="profile-card mb-4">
                        <div class="profile-card-header">
                            <div class="profile-card-icon">
                                ${user.avatar_url ? 
                                    `<img src="${user.avatar_url}" alt="${user.name || user.username}" 
                                          class="profile-avatar" 
                                          onerror="this.onerror=null; this.style.display='none'; this.parentElement.innerHTML='<div class=\"default-avatar\"><i class=\"fas fa-user-circle\"></i></div>';" 
                                          style="width: 120px; height: 120px; border-radius: 50%; object-fit: cover; border: 4px solid white;">` :
                                    `<div class="default-avatar">
                                        <i class="fas fa-user-circle"></i>
                                     </div>`
                                }
                            </div>
                            <h3 class="profile-card-title">${escapeHtml(user.name || user.username || '未设置用户名')}</h3>
                            <p class="profile-card-subtitle">
                                <i class="fas fa-user me-2"></i>@${escapeHtml(user.username || 'unknown')}
                            </p>
                            <div class="profile-badges mt-3">
                                <span class="badge-custom bg-primary">
                                    <i class="fas fa-star me-1"></i> 学员
                                </span>
                                <span class="badge-custom bg-secondary ms-2">
                                    <i class="fas fa-calendar me-1"></i> 加入于 ${formatDate(user.created_at || user.createdAt)}
                                </span>
                            </div>
                        </div>
                        <div class="profile-card-body">
                            <!-- 基本信息区块 -->
                            <div class="info-section">
                                <h5 class="section-title">
                                    <i class="fas fa-info-circle me-2"></i>基本信息
                                </h5>
                                <div class="info-grid">
                                    <div class="info-item">
                                        <div class="info-label">
                                            <i class="fas fa-id-card me-2"></i>用户名
                                        </div>
                                        <div class="info-value">${escapeHtml(user.username || '未设置')}</div>
                                    </div>
                                    <div class="info-item">
                                        <div class="info-label">
                                            <i class="fas fa-user-tag me-2"></i>显示名称
                                        </div>
                                        <div class="info-value">${escapeHtml(user.name || '未设置')}</div>
                                    </div>
                                    <div class="info-item">
                                        <div class="info-label">
                                            <i class="fas fa-envelope me-2"></i>邮箱
                                        </div>
                                        <div class="info-value">${escapeHtml(user.email || '未设置')}</div>
                                    </div>
                                    <div class="info-item">
                                        <div class="info-label">
                                            <i class="fas fa-calendar-alt me-2"></i>注册时间
                                        </div>
                                        <div class="info-value">${formatDate(user.created_at || user.createdAt)}</div>
                                    </div>
                                </div>
                            </div>
                            
                            <!-- 个人简介区块 -->
                            <div class="info-section mt-4">
                                <h5 class="section-title">
                                    <i class="fas fa-comment-dots me-2"></i>个人简介
                                </h5>
                                <div class="bio-content ${user.bio ? 'has-content' : 'no-content'}">
                                    ${user.bio ? escapeHtml(user.bio) : 
                                        '<div class="text-center text-muted py-3">' +
                                        '<i class="fas fa-quote-left fa-2x mb-3"></i>' +
                                        '<p>还没有个人简介，添加一些介绍吧！</p>' +
                                        '</div>'
                                    }
                                </div>
                            </div>
                            
                            <!-- 头像管理区块 -->
                            <div class="info-section mt-4">
                                <h5 class="section-title">
                                    <i class="fas fa-image me-2"></i>头像管理
                                </h5>
                                <div class="avatar-management">
                                    <!-- 当前头像预览 -->
                                    <div class="current-avatar-preview text-center mb-4">
                                        <div id="currentAvatar" style="position: relative; display: inline-block;">
                                            ${user.avatar_url ? 
                                                `<img src="${user.avatar_url}" 
                                                     alt="当前头像" 
                                                     class="avatar-img mb-2"
                                                     id="avatarPreviewImg"
                                                     onerror="this.onerror=null; this.src='data:image/svg+xml,<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 100 100\"><circle cx=\"50\" cy=\"50\" r=\"45\" fill=\"%232A7BFF\"/><text x=\"50\" y=\"55\" text-anchor=\"middle\" fill=\"white\" font-size=\"40\">${user.name ? user.name.charAt(0) : 'U'}</text></svg>';"
                                                     style="width: 150px; height: 150px; border-radius: 50%; object-fit: cover; border: 4px solid var(--primary-color); box-shadow: 0 6px 20px rgba(42, 123, 255, 0.3);">` :
                                                `<div class="default-avatar d-flex align-items-center justify-content-center mb-2"
                                                      style="width: 150px; height: 150px; border-radius: 50%; background: linear-gradient(135deg, var(--primary-color), var(--secondary-color)); border: 4px solid var(--primary-color); box-shadow: 0 6px 20px rgba(42, 123, 255, 0.3);">
                                                    <i class="fas fa-user-circle text-white" style="font-size: 80px;"></i>
                                                 </div>`
                                            }
                                        </div>
                                        <p class="text-muted small mt-2">当前头像</p>
                                    </div>

                                    <!-- 头像上传表单 -->
                                    <div class="avatar-upload-form mb-4">
                                        <div class="upload-card" style="border: 2px dashed var(--border-color); border-radius: var(--border-radius); padding: 2rem; text-align: center;">
                                            <div class="upload-icon mb-3">
                                                <i class="fas fa-cloud-upload-alt fa-3x text-primary"></i>
                                            </div>
                                            <h5 class="mb-3">上传新头像</h5>
                                            <p class="text-muted mb-3">支持 JPG、PNG、GIF、WebP 格式，最大 2MB</p>
                                            
                                            <div class="d-flex flex-column align-items-center gap-3">
                                                <!-- 文件选择 -->
                                                <div class="file-input-wrapper">
                                                    <input type="file" id="avatarFileInput" accept="image/*" 
                                                           class="form-control" style="display: none;">
                                                    <button class="btn-custom btn-primary-custom" onclick="document.getElementById('avatarFileInput').click()">
                                                        <i class="fas fa-folder-open me-2"></i>选择图片
                                                    </button>
                                                    <span id="selectedFileName" class="ms-2 text-muted"></span>
                                                </div>
                                                
                                                <!-- 预览区域 -->
                                                <div id="avatarUploadPreview" class="mt-3" style="display: none;">
                                                    <img id="newAvatarPreview" 
                                                         src="" 
                                                         alt="新头像预览"
                                                         style="width: 100px; height: 100px; border-radius: 50%; object-fit: cover; border: 2px solid var(--primary-color);">
                                                    <p class="text-muted small mt-2">新头像预览</p>
                                                </div>
                                                
                                                <!-- 上传按钮 -->
                                                <button id="uploadAvatarBtn" class="btn-custom btn-success-custom" onclick="handleAvatarUpload()" disabled>
                                                    <i class="fas fa-upload me-2"></i>上传头像
                                                </button>
                                            </div>
                                            
                                            <!-- 进度条 -->
                                            <div id="uploadProgress" class="mt-3" style="display: none;">
                                                <div class="progress" style="height: 8px; border-radius: 4px;">
                                                    <div id="uploadProgressBar" class="progress-bar progress-bar-striped progress-bar-animated" 
                                                         role="progressbar" style="width: 0%"></div>
                                                </div>
                                                <div class="text-center mt-2">
                                                    <small id="uploadStatus" class="text-muted">上传中...</small>
                                                </div>
                                            </div>
                                        </div>
                                    </div>

                                    <!-- 删除头像按钮（如果有头像） -->
                                    ${user.avatar_url ? `
                                    <div class="avatar-delete text-center">
                                        <button class="btn-custom btn-danger-custom" onclick="handleAvatarDelete()">
                                            <i class="fas fa-trash-alt me-2"></i>删除头像
                                        </button>
                                        <p class="text-muted small mt-2">删除后将使用默认头像</p>
                                    </div>
                                    ` : ''}
                                    
                                    <!-- 头像操作消息 -->
                                    <div id="avatarMessage" class="mt-3"></div>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <!-- 编辑资料卡片 -->
                    <div class="profile-card">
                        <div class="profile-card-header">
                            <h3 class="profile-card-title">
                                <i class="fas fa-edit me-2"></i>编辑个人资料
                            </h3>
                            <p class="profile-card-subtitle">更新您的个人信息</p>
                        </div>
                        <div class="profile-card-body">
                            <form id="profileForm">
                                <div class="row">
                                    <div class="col-md-6 mb-3">
                                        <div class="form-group-custom">
                                            <label for="profileName" class="form-label">
                                                <i class="fas fa-user-tag me-2"></i>显示名称
                                            </label>
                                            <div class="input-with-icon">
                                                <input type="text" class="form-control" id="profileName" 
                                                       value="${escapeHtml(user.name || '')}"
                                                       placeholder="请输入显示名称"
                                                       style="border: 2px solid var(--border-color);">
                                                <div class="input-icon">
                                                    <i class="fas fa-user-edit"></i>
                                                </div>
                                            </div>
                                            <div class="form-text text-muted">显示名称将在文章和评论中显示</div>
                                        </div>
                                    </div>
                                    
                                    <div class="col-md-6 mb-3">
                                        <div class="form-group-custom">
                                            <label for="profileEmail" class="form-label">
                                                <i class="fas fa-envelope me-2"></i>邮箱地址
                                            </label>
                                            <div class="input-with-icon">
                                                <input type="email" class="form-control" id="profileEmail" 
                                                       value="${escapeHtml(user.email || '')}"
                                                       placeholder="请输入邮箱地址"
                                                       style="border: 2px solid var(--border-color);">
                                                <div class="input-icon">
                                                    <i class="fas fa-at"></i>
                                                </div>
                                            </div>
                                            <div class="form-text text-muted">用于接收通知和找回密码</div>
                                        </div>
                                    </div>
                                </div>
                                
                                <div class="mb-3">
                                    <div class="form-group-custom">
                                        <label for="profileBio" class="form-label">
                                            <i class="fas fa-comment-dots me-2"></i>个人简介
                                        </label>
                                        <div class="input-with-icon">
                                            <textarea class="form-control" id="profileBio" rows="4" 
                                                      placeholder="介绍一下自己吧..."
                                                      style="border: 2px solid var(--border-color);">${escapeHtml(user.bio || '')}</textarea>
                                            <div class="input-icon">
                                                <i class="fas fa-pen"></i>
                                            </div>
                                        </div>
                                        <div class="form-text text-muted">最多500个字符，支持Markdown格式</div>
                                    </div>
                                </div>
                                
                                <div class="d-grid gap-2">
                                    <button type="submit" class="btn-custom btn-primary-custom btn-lg">
                                        <i class="fas fa-save me-2"></i>保存更改
                                    </button>
                                    <div class="btn-group-custom mt-3">
                                        <button type="button" class="btn-custom btn-outline-custom" onclick="showHome()">
                                            <i class="fas fa-home me-2"></i>返回首页
                                        </button>
                                        <button type="button" class="btn-custom btn-outline-custom" onclick="refreshProfile()">
                                            <i class="fas fa-redo me-2"></i>刷新资料
                                        </button>
                                        <button type="button" class="btn-custom btn-danger-custom" onclick="logout()">
                                            <i class="fas fa-sign-out-alt me-2"></i>退出登录
                                        </button>
                                    </div>
                                </div>
                            </form>
                            <div id="profileMessage" class="mt-3"></div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        // 重新绑定表单提交事件
        const profileForm = document.getElementById('profileForm');
        if (profileForm) {
            const newForm = profileForm.cloneNode(true);
            profileForm.parentNode.replaceChild(newForm, profileForm);
            newForm.addEventListener('submit', handleProfileUpdate);
        }
        
        // 绑定头像文件选择事件
        const avatarFileInput = document.getElementById('avatarFileInput');
        if (avatarFileInput) {
            avatarFileInput.addEventListener('change', function() {
                const file = this.files[0];
                const uploadBtn = document.getElementById('uploadAvatarBtn');
                const fileNameSpan = document.getElementById('selectedFileName');
                const previewContainer = document.getElementById('avatarUploadPreview');
                const previewImg = document.getElementById('newAvatarPreview');
                
                if (file) {
                    fileNameSpan.textContent = file.name;
                    
                    if (!CONFIG.AVATAR_TYPES.includes(file.type)) {
                        showMessage('avatarMessage', '只支持 JPG、PNG、GIF、WebP 格式的图片', 'warning');
                        uploadBtn.disabled = true;
                        return;
                    }
                    
                    if (file.size > CONFIG.MAX_AVATAR_SIZE) {
                        showMessage('avatarMessage', '图片大小不能超过 2MB', 'warning');
                        uploadBtn.disabled = true;
                        return;
                    }
                    
                    const reader = new FileReader();
                    reader.onload = function(e) {
                        previewImg.src = e.target.result;
                        previewContainer.style.display = 'block';
                        uploadBtn.disabled = false;
                    };
                    reader.readAsDataURL(file);
                } else {
                    fileNameSpan.textContent = '';
                    previewContainer.style.display = 'none';
                    uploadBtn.disabled = true;
                }
            });
        }
        
    } catch (error) {
        console.error('加载用户资料失败:', error);
        
        container.innerHTML = `
            <div class="row justify-content-center">
                <div class="col-md-10">
                    <div class="profile-card">
                        <div class="profile-card-body">
                            <div class="alert alert-danger" style="border: 2px solid var(--danger-color); border-radius: var(--border-radius);">
                                <div class="d-flex align-items-center">
                                    <i class="fas fa-exclamation-triangle fa-2x me-3"></i>
                                    <div>
                                        <h5 class="fw-bold mb-2">加载个人资料失败</h5>
                                        <p class="mb-3">${error.message || '未知错误，请检查网络连接或登录状态'}</p>
                                        <div class="d-flex gap-2">
                                            <button class="btn-custom btn-primary-custom" onclick="loadUserProfile()">
                                                <i class="fas fa-redo me-2"></i>重新加载
                                            </button>
                                            <button class="btn-custom btn-outline-custom" onclick="showHome()">
                                                <i class="fas fa-home me-2"></i>返回首页
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }
}

// 刷新个人资料
function refreshProfile() {
    showMessage('profileMessage', '正在刷新资料...', 'info');
    setTimeout(() => {
        loadUserProfile();
    }, 500);
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
        
        showGlobalMessage('已成功退出登录。', 'info', 3000);
    }
}

// 处理头像上传
async function handleAvatarUpload() {
    const fileInput = document.getElementById('avatarFileInput');
    const uploadBtn = document.getElementById('uploadAvatarBtn');
    const progressBar = document.getElementById('uploadProgressBar');
    const progressContainer = document.getElementById('uploadProgress');
    const statusText = document.getElementById('uploadStatus');
    
    if (!fileInput.files || fileInput.files.length === 0) {
        showMessage('avatarMessage', '请先选择图片文件', 'warning');
        return;
    }
    
    const file = fileInput.files[0];
    
    try {
        uploadBtn.disabled = true;
        uploadBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 上传中...';
        
        progressContainer.style.display = 'block';
        progressBar.style.width = '30%';
        statusText.textContent = '正在上传...';
        
        const xhr = new XMLHttpRequest();
        const formData = new FormData();
        formData.append('avatar', file);
        
        return new Promise((resolve, reject) => {
            xhr.open('POST', CONFIG.API_BASE_URL + '/user/avatar');
            xhr.setRequestHeader('Authorization', `Bearer ${STATE.currentToken}`);
            
            xhr.upload.onprogress = function(e) {
                if (e.lengthComputable) {
                    const percentComplete = Math.round((e.loaded / e.total) * 100);
                    progressBar.style.width = percentComplete + '%';
                    statusText.textContent = `上传中... ${percentComplete}%`;
                }
            };
            
            xhr.onload = function() {
                progressBar.style.width = '100%';
                statusText.textContent = '上传完成！';
                
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        const data = JSON.parse(xhr.responseText);
                        
                        if (data.success && data.avatar_url) {
                            STATE.currentUser.avatar_url = data.avatar_url;
                            localStorage.setItem(CONFIG.USER_KEY, JSON.stringify(STATE.currentUser));
                            
                            const previewImg = document.getElementById('avatarPreviewImg');
                            if (previewImg) {
                                previewImg.src = data.avatar_url + '?t=' + Date.now();
                            }
                            
                            const defaultAvatar = document.querySelector('.default-avatar');
                            if (defaultAvatar) {
                                defaultAvatar.style.display = 'none';
                            }
                            
                            showMessage('avatarMessage', '头像上传成功！', 'success');
                            
                            setTimeout(() => {
                                updateNavigation();
                                refreshProfile();
                            }, 1000);
                        }
                    } catch (error) {
                        showMessage('avatarMessage', '解析响应失败', 'danger');
                    }
                } else {
                    try {
                        const errorData = JSON.parse(xhr.responseText);
                        showMessage('avatarMessage', `上传失败: ${errorData.error || '未知错误'}`, 'danger');
                    } catch (e) {
                        showMessage('avatarMessage', `上传失败: ${xhr.statusText}`, 'danger');
                    }
                }
                
                setTimeout(() => {
                    uploadBtn.disabled = false;
                    uploadBtn.innerHTML = '<i class="fas fa-upload me-2"></i>上传头像';
                    progressContainer.style.display = 'none';
                    progressBar.style.width = '0%';
                    
                    fileInput.value = '';
                    document.getElementById('selectedFileName').textContent = '';
                    document.getElementById('avatarUploadPreview').style.display = 'none';
                }, 1500);
                
                resolve();
            };
            
            xhr.onerror = function() {
                showMessage('avatarMessage', '网络错误，上传失败', 'danger');
                
                uploadBtn.disabled = false;
                uploadBtn.innerHTML = '<i class="fas fa-upload me-2"></i>上传头像';
                progressContainer.style.display = 'none';
                progressBar.style.width = '0%';
                
                reject(new Error('Network error'));
            };
            
            xhr.send(formData);
        });
        
    } catch (error) {
        console.error('上传头像失败:', error);
        showMessage('avatarMessage', `上传失败: ${error.message}`, 'danger');
        
        uploadBtn.disabled = false;
        uploadBtn.innerHTML = '<i class="fas fa-upload me-2"></i>上传头像';
        progressContainer.style.display = 'none';
        progressBar.style.width = '0%';
    }
}

// 处理头像删除
async function handleAvatarDelete() {
    if (!confirm('确定要删除头像吗？删除后将会使用默认头像。')) {
        return;
    }
    
    const deleteBtn = document.querySelector('.avatar-delete button');
    const originalText = deleteBtn.innerHTML;
    
    try {
        deleteBtn.disabled = true;
        deleteBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 删除中...';
        
        const response = await apiCall('/user/avatar', 'DELETE', null, true);
        
        if (response.success) {
            STATE.currentUser.avatar_url = '';
            localStorage.setItem(CONFIG.USER_KEY, JSON.stringify(STATE.currentUser));
            
            const avatarImg = document.getElementById('avatarPreviewImg');
            if (avatarImg) {
                avatarImg.style.display = 'none';
            }
            
            const defaultAvatar = document.querySelector('.default-avatar');
            if (defaultAvatar) {
                defaultAvatar.style.display = 'flex';
            }
            
            showMessage('avatarMessage', '头像删除成功！', 'success');
            
            setTimeout(() => {
                updateNavigation();
            }, 500);
        }
        
    } catch (error) {
        console.error('删除头像失败:', error);
        showMessage('avatarMessage', `删除失败: ${error.message}`, 'danger');
    } finally {
        deleteBtn.disabled = false;
        deleteBtn.innerHTML = originalText;
    }
}