/**
 * 关注模块 - 处理用户关注功能
 */

// 关注用户
async function followUser(userId) {
    if (!isLoggedIn()) {
        showLogin();
        return;
    }
    
    const followBtn = document.getElementById(`followBtn${userId}`);
    if (!followBtn) return;
    
    const originalText = followBtn.innerHTML;
    
    try {
        followBtn.disabled = true;
        followBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 处理中...';
        
        // 使用新的API路径
        await apiCall(`/follow/user/${userId}`, 'POST', null, true);
        
        // 更新按钮状态
        followBtn.innerHTML = '<i class="fas fa-user-check"></i> 已关注';
        followBtn.className = 'btn btn-sm btn-success';
        followBtn.onclick = function() { unfollowUser(userId); };
        
        showGlobalMessage('关注成功！', 'success', 3000);
        
    } catch (error) {
        console.error('关注失败:', error);
        
        followBtn.disabled = false;
        followBtn.innerHTML = originalText;
        
        if (error.message.includes('已经关注')) {
            showGlobalMessage('您已经关注了该用户', 'info', 3000);
        } else {
            showGlobalMessage(`关注失败: ${error.message}`, 'danger', 3000);
        }
    }
}

// 取消关注
async function unfollowUser(userId) {
    if (!confirm('确定要取消关注吗？')) {
        return;
    }
    
    const followBtn = document.getElementById(`followBtn${userId}`);
    if (!followBtn) return;
    
    const originalText = followBtn.innerHTML;
    
    try {
        followBtn.disabled = true;
        followBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 处理中...';
        
        // 使用新的API路径
        await apiCall(`/follow/user/${userId}`, 'DELETE', null, true);
        
        // 更新按钮状态
        followBtn.innerHTML = '<i class="fas fa-user-plus"></i> 关注';
        followBtn.className = 'btn btn-sm btn-primary';
        followBtn.onclick = function() { followUser(userId); };
        
        showGlobalMessage('已取消关注', 'info', 3000);
        
    } catch (error) {
        console.error('取消关注失败:', error);
        
        followBtn.disabled = false;
        followBtn.innerHTML = originalText;
        showGlobalMessage(`操作失败: ${error.message}`, 'danger', 3000);
    }
}

// 获取关注列表
async function getFollowingList(userId) {
    try {
        // 使用新的API路径
        return await apiCall(`/follow/user/${userId}/following`);
    } catch (error) {
        console.error('获取关注列表失败:', error);
        return [];
    }
}

// 获取粉丝列表
async function getFollowerList(userId) {
    try {
        // 使用新的API路径
        return await apiCall(`/follow/user/${userId}/followers`);
    } catch (error) {
        console.error('获取粉丝列表失败:', error);
        return [];
    }
}

// 获取关注统计
async function getFollowStats(userId) {
    try {
        // 使用新的API路径
        return await apiCall(`/follow/user/${userId}/stats`);
    } catch (error) {
        console.error('获取关注统计失败:', error);
        return { following_count: 0, follower_count: 0 };
    }
}

// 检查是否关注
async function checkFollowing(userId) {
    try {
        // 使用新的API路径
        const data = await apiCall(`/follow/user/${userId}/is-following`, 'GET', null, true);
        return data.is_following || false;
    } catch (error) {
        console.error('检查关注状态失败:', error);
        return false;
    }
}

// 渲染关注列表
function renderFollowList(users, type = 'following') {
    if (!users || users.length === 0) {
        return `
            <div class="text-center py-4">
                <i class="fas fa-user-friends fa-3x text-muted"></i>
                <p class="mt-2 text-muted">
                    ${type === 'following' ? '还没有关注任何人' : '还没有粉丝'}
                </p>
            </div>
        `;
    }
    
    let html = '<div class="row">';
    
    users.forEach(user => {
        // 获取用户显示名称
        const displayName = user.name || user.username;
        
        html += `
            <div class="col-md-6 mb-3">
                <div class="follow-item">
                    <div class="follow-item-header">
                        ${user.avatar_url ? 
                            `<img src="${user.avatar_url}" alt="${displayName}" class="follow-avatar">` :
                            `<div class="follow-default-avatar"><i class="fas fa-user-circle"></i></div>`
                        }
                        <div class="follow-user-info">
                            <h6 class="follow-user-name">${escapeHtml(displayName)}</h6>
                            <small class="follow-user-username">@${escapeHtml(user.username)}</small>
                        </div>
                    </div>
                    ${user.bio ? `<p class="follow-user-bio mt-2">${escapeHtml(truncateText(user.bio, 100))}</p>` : ''}
                    <div class="follow-item-actions mt-2">
                        <button class="btn btn-sm btn-outline-primary" onclick="viewUserProfile('${user.username}')">
                            查看资料
                        </button>
                        ${isLoggedIn() && STATE.currentUser && STATE.currentUser.id !== user.id ? `
                            ${type === 'followers' ? 
                                `<button class="btn btn-sm btn-primary" onclick="followUser(${user.id})" id="followBtn${user.id}">
                                    <i class="fas fa-user-plus"></i> 关注
                                </button>` : ''
                            }
                        ` : ''}
                    </div>
                </div>
            </div>
        `;
    });
    
    html += '</div>';
    return html;
}

// 显示用户关注页面
async function showUserFollowPage(username, tab = 'following') {
    try {
        // 首先获取用户信息
        const user = await apiCall(`/users/${username}`);
        
        // 显示用户关注页面
        if (!elementExists('userFollowPage')) {
            console.error('用户关注页面不存在');
            return;
        }
        
        showPage('userFollow');
        
        // 更新页面标题
        document.getElementById('followPageTitle').innerHTML = `
            <i class="fas fa-user-circle"></i> ${escapeHtml(user.name || user.username)} 的${tab === 'following' ? '关注' : '粉丝'}
        `;
        
        // 存储当前用户信息
        document.getElementById('currentUserId').value = user.id;
        document.getElementById('currentUsername').value = user.username;
        
        // 加载关注/粉丝列表
        await loadFollowList(user.id, tab);
        
        // 更新选项卡
        updateFollowTabs(username, tab);
        
    } catch (error) {
        console.error('加载用户关注页面失败:', error);
        showGlobalMessage(`加载失败: ${error.message}`, 'danger', 3000);
        
        // 返回首页
        setTimeout(() => showHome(), 2000);
    }
}

// 加载关注列表
async function loadFollowList(userId, type = 'following') {
    const container = document.getElementById('followListContainer');
    if (!container) {
        console.error('关注列表容器不存在');
        return;
    }
    
    container.innerHTML = `
        <div class="text-center py-5">
            <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">加载中...</span>
            </div>
            <p class="mt-2">正在加载...</p>
        </div>
    `;
    
    try {
        let users;
        if (type === 'following') {
            users = await getFollowingList(userId);
        } else {
            users = await getFollowerList(userId);
        }
        
        container.innerHTML = renderFollowList(users, type);
        
        // 对于粉丝列表，检查当前用户是否已经关注他们
        if (type === 'followers' && isLoggedIn()) {
            users.forEach(async user => {
                if (STATE.currentUser.id !== user.id) {
                    const isFollowing = await checkFollowing(user.id);
                    const followBtn = document.getElementById(`followBtn${user.id}`);
                    if (followBtn) {
                        if (isFollowing) {
                            followBtn.innerHTML = '<i class="fas fa-user-check"></i> 已关注';
                            followBtn.className = 'btn btn-sm btn-success';
                            followBtn.onclick = function() { unfollowUser(user.id); };
                        } else {
                            followBtn.innerHTML = '<i class="fas fa-user-plus"></i> 关注';
                            followBtn.className = 'btn btn-sm btn-primary';
                            followBtn.onclick = function() { followUser(user.id); };
                        }
                    }
                }
            });
        }
        
    } catch (error) {
        console.error('加载列表失败:', error);
        container.innerHTML = `
            <div class="alert alert-danger">
                <i class="fas fa-exclamation-triangle"></i> 加载失败: ${error.message}
            </div>
        `;
    }
}

// 更新关注选项卡
function updateFollowTabs(username, activeTab) {
    const tabs = document.getElementById('followTabs');
    if (tabs) {
        tabs.innerHTML = `
            <ul class="nav nav-pills nav-justified">
                <li class="nav-item">
                    <a class="nav-link ${activeTab === 'following' ? 'active' : ''}" 
                       onclick="showUserFollowPage('${username}', 'following')">
                        <i class="fas fa-user-plus"></i> 关注
                    </a>
                </li>
                <li class="nav-item">
                    <a class="nav-link ${activeTab === 'followers' ? 'active' : ''}" 
                       onclick="showUserFollowPage('${username}', 'followers')">
                        <i class="fas fa-user-friends"></i> 粉丝
                    </a>
                </li>
            </ul>
        `;
    }
}

// 查看用户公开资料
async function viewUserProfile(username) {
    try {
        const user = await apiCall(`/users/${username}`);
        
        // 创建一个模态框显示用户信息
        const modalHtml = `
            <div class="modal fade" id="userProfileModal" tabindex="-1">
                <div class="modal-dialog modal-lg">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">
                                <i class="fas fa-user-circle"></i> ${escapeHtml(user.name || user.username)} 的个人资料
                            </h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <div class="modal-body">
                            <div class="text-center mb-4">
                                ${user.avatar_url ? 
                                    `<img src="${user.avatar_url}" alt="${user.name || user.username}" 
                                          class="rounded-circle" style="width: 120px; height: 120px; object-fit: cover;">` :
                                    `<div class="rounded-circle bg-primary d-inline-flex align-items-center justify-content-center" 
                                          style="width: 120px; height: 120px;">
                                        <i class="fas fa-user-circle text-white" style="font-size: 80px;"></i>
                                     </div>`
                                }
                                <h4 class="mt-3">${escapeHtml(user.name || user.username)}</h4>
                                <p class="text-muted">@${escapeHtml(user.username)}</p>
                            </div>
                            
                            <div class="row mb-3">
                                <div class="col-md-6">
                                    <div class="card h-100">
                                        <div class="card-body text-center">
                                            <h5 class="card-title">关注</h5>
                                            <p class="card-text display-6">${user.following_count || 0}</p>
                                            <button class="btn btn-sm btn-outline-primary" 
                                                    onclick="$('#userProfileModal').modal('hide'); setTimeout(() => showUserFollowPage('${username}', 'following'), 300)">
                                                查看列表
                                            </button>
                                        </div>
                                    </div>
                                </div>
                                <div class="col-md-6">
                                    <div class="card h-100">
                                        <div class="card-body text-center">
                                            <h5 class="card-title">粉丝</h5>
                                            <p class="card-text display-6">${user.follower_count || 0}</p>
                                            <button class="btn btn-sm btn-outline-primary"
                                                    onclick="$('#userProfileModal').modal('hide'); setTimeout(() => showUserFollowPage('${username}', 'followers'), 300)">
                                                查看列表
                                            </button>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            
                            ${user.bio ? `
                                <div class="card mb-3">
                                    <div class="card-header">
                                        <i class="fas fa-comment-dots"></i> 个人简介
                                    </div>
                                    <div class="card-body">
                                        <p class="card-text">${escapeHtml(user.bio)}</p>
                                    </div>
                                </div>
                            ` : ''}
                            
                            <div class="card">
                                <div class="card-header">
                                    <i class="fas fa-info-circle"></i> 基本信息
                                </div>
                                <div class="card-body">
                                    <p><i class="fas fa-calendar-alt me-2"></i> 注册时间: ${formatDate(user.created_at)}</p>
                                </div>
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">关闭</button>
                            ${isLoggedIn() && STATE.currentUser && STATE.currentUser.id !== user.id ? `
                                ${await checkFollowing(user.id) ? 
                                    `<button type="button" class="btn btn-danger" onclick="unfollowUser(${user.id})">
                                        <i class="fas fa-user-minus"></i> 取消关注
                                    </button>` :
                                    `<button type="button" class="btn btn-primary" onclick="followUser(${user.id})">
                                        <i class="fas fa-user-plus"></i> 关注
                                    </button>`
                                }
                            ` : ''}
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        // 移除现有的模态框
        const existingModal = document.getElementById('userProfileModal');
        if (existingModal) {
            existingModal.remove();
        }
        
        // 添加新的模态框
        document.body.insertAdjacentHTML('beforeend', modalHtml);
        
        // 显示模态框
        const modal = new bootstrap.Modal(document.getElementById('userProfileModal'));
        modal.show();
        
    } catch (error) {
        console.error('查看用户资料失败:', error);
        showGlobalMessage(`加载用户资料失败: ${error.message}`, 'danger', 3000);
    }
}

// 在用户个人资料页显示关注/粉丝信息
async function loadUserFollowInfo(userId) {
    try {
        // 获取关注统计
        const stats = await getFollowStats(userId);
        
        // 创建关注信息HTML
        const followInfoHtml = `
            <div class="follow-info-card mt-4">
                <div class="follow-info-header">
                    <i class="fas fa-user-friends"></i> 关注信息
                </div>
                <div class="follow-info-body">
                    <div class="row text-center">
                        <div class="col">
                            <a href="#" onclick="showUserFollowPage('${STATE.currentUser.username}', 'following')" class="follow-stat-link">
                                <div class="follow-stat-number">${stats.following_count || 0}</div>
                                <div class="follow-stat-label">关注</div>
                            </a>
                        </div>
                        <div class="col">
                            <a href="#" onclick="showUserFollowPage('${STATE.currentUser.username}', 'followers')" class="follow-stat-link">
                                <div class="follow-stat-number">${stats.follower_count || 0}</div>
                                <div class="follow-stat-label">粉丝</div>
                            </a>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        // 添加到用户资料页面
        const profileContainer = document.querySelector('.profile-card-body');
        if (profileContainer) {
            // 查找合适的位置插入（在头像管理区块之后）
            const avatarManagement = profileContainer.querySelector('.avatar-management');
            if (avatarManagement && avatarManagement.parentNode) {
                avatarManagement.parentNode.insertAdjacentHTML('afterend', followInfoHtml);
            }
        }
        
    } catch (error) {
        console.error('加载关注信息失败:', error);
        // 不显示错误，因为这个功能是可选的
    }
}

// 初始化函数：在加载用户资料时调用
async function initUserFollow() {
    if (isLoggedIn() && STATE.currentUser) {
        // 加载用户的关注信息
        await loadUserFollowInfo(STATE.currentUser.id);
    }
}