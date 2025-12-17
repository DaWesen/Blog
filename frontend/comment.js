/**
 * 评论模块 - 处理评论的增删改查
 */

// 加载文章评论
async function loadPostComments(postId) {
    const container = document.getElementById('commentsContainer');
    
    // 显示加载中
    container.innerHTML = `
        <div class="text-center py-3">
            <div class="spinner-border spinner-border-sm text-primary" role="status">
                <span class="visually-hidden">加载中...</span>
            </div>
            <p class="mt-2 text-muted">加载评论中...</p>
        </div>
    `;
    
    try {
        const response = await apiCall(`/posts/${postId}/comments?page=1&size=50`);
        
        console.log('评论接口返回数据:', response); // 调试日志
        
        // 确保返回的数据结构正确
        if (!response || typeof response !== 'object') {
            throw new Error('返回的数据格式不正确');
        }
        
        // 从响应数据中获取评论数组
        let comments = [];
        if (Array.isArray(response)) {
            comments = response;
        } else if (response.comments && Array.isArray(response.comments)) {
            comments = response.comments;
        } else if (response.data && Array.isArray(response.data)) {
            comments = response.data;
        }
        
        if (!comments || comments.length === 0) {
            container.innerHTML = `
                <div class="text-center py-4">
                    <i class="bi bi-chat-text display-6 text-muted"></i>
                    <p class="mt-2 text-muted">暂无评论，快来抢沙发吧！</p>
                </div>
            `;
            return;
        }
        
        // 渲染评论列表
        let html = '';
        comments.forEach(comment => {
            if (comment && comment.id) { // 确保评论对象有效
                html += renderComment(comment, 0);
            }
        });
        
        container.innerHTML = html;
        
    } catch (error) {
        console.error('加载评论失败:', error);
        container.innerHTML = `
            <div class="alert alert-warning">
                <i class="bi bi-exclamation-triangle"></i> 
                ${error.message === '文章不存在或已被删除' ? 
                    '文章不存在或已被删除' : 
                    `加载评论失败: ${error.message || '未知错误'}`
                }
            </div>
        `;
    }
}

// 渲染单个评论（支持嵌套）
function renderComment(comment, depth = 0) {
    const marginLeft = depth * 30;
    const replyStyle = depth > 0 ? 'border-start border-3 border-light ps-3' : '';
    
    // 获取用户信息
    let userName = '匿名用户';
    let userId = null;
    
    if (comment.user) {
        userName = comment.user.name || comment.user.username || '匿名用户';
        userId = comment.user.id;
    } else if (comment.user_id) {
        userName = '用户' + comment.user_id;
        userId = comment.user_id;
    }
    
    // 获取点赞数
    const likeCount = comment.like_count || comment.likeCount || comment.LikeCount || 0;
    
    // 获取回复列表
    const replies = comment.replies || comment.Replies || [];
    
    // 检查当前用户是否可以操作
    const canDelete = isLoggedIn() && 
                     STATE.currentUser && 
                     (STATE.currentUser.id === userId || 
                      STATE.currentUser.id === comment.user_id);
    
    let html = `
        <div class="comment-item ${replyStyle}" style="margin-left: ${marginLeft}px">
            <div class="d-flex justify-content-between align-items-start">
                <div>
                    <strong class="comment-author">${escapeHtml(userName)}</strong>
                    <small class="comment-time ms-2">${formatDate(comment.created_at || comment.CreatedAt)}</small>
                </div>
                <div class="btn-group">
                    <button class="btn btn-sm btn-outline-primary" onclick="likeComment(${comment.id})" id="commentLikeBtn${comment.id}">
                        <i class="bi bi-heart"></i> ${likeCount}
                    </button>
                    ${isLoggedIn() ? `
                        <button class="btn btn-sm btn-outline-secondary" onclick="showReplyForm(${comment.id})">
                            <i class="bi bi-reply"></i> 回复
                        </button>
                    ` : ''}
                    ${canDelete ? `
                        <button class="btn btn-sm btn-outline-danger" onclick="deleteComment(${comment.id})">
                            <i class="bi bi-trash"></i>
                        </button>
                    ` : ''}
                </div>
            </div>
            <div class="comment-content mt-2">
                ${escapeHtml(comment.content || '')}
            </div>
            <div id="replyForm${comment.id}" class="mt-3 d-none">
                <form onsubmit="handleReply(${comment.id}, event)">
                    <div class="input-group">
                        <textarea class="form-control" placeholder="回复 ${userName}..." rows="2" required></textarea>
                        <button class="btn btn-primary" type="submit">发送</button>
                    </div>
                </form>
            </div>
    `;
    
    // 递归渲染回复
    if (replies.length > 0) {
        replies.forEach(reply => {
            if (reply && reply.id) {
                html += renderComment(reply, depth + 1);
            }
        });
    }
    
    html += '</div>';
    return html;
}

// 显示回复表单
function showReplyForm(commentId) {
    if (!isLoggedIn()) {
        showLogin();
        return;
    }
    
    const form = document.getElementById(`replyForm${commentId}`);
    form.classList.toggle('d-none');
    
    if (!form.classList.contains('d-none')) {
        form.querySelector('textarea').focus();
    }
}

// 处理评论提交
async function handleCommentSubmit(e) {
    e.preventDefault();
    
    if (!isLoggedIn()) {
        showLogin();
        return;
    }
    
    const content = document.getElementById('commentContent').value.trim();
    if (!content) {
        showGlobalMessage('请输入评论内容', 'warning', 3000);
        return;
    }
    
    const submitBtn = document.querySelector('#commentForm button[type="submit"]');
    const originalText = submitBtn.innerHTML;
    
    try {
        // 显示提交中状态
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 发表中...';
        
        console.log('发表评论:', { postId: STATE.currentPostId, content });
        await apiCall('/comments', 'POST', {
            post_id: STATE.currentPostId,
            content: content
        }, true);
        
        // 清空表单
        document.getElementById('commentContent').value = '';
        
        // 显示成功提示
        showGlobalMessage('评论发表成功！', 'success', 3000);
        
        // 重新加载评论
        setTimeout(() => {
            loadPostComments(STATE.currentPostId);
        }, 500);
        
    } catch (error) {
        console.error('发表评论失败:', error);
        showGlobalMessage(`发表失败: ${error.message}`, 'danger', 3000);
    } finally {
        // 恢复按钮状态
        submitBtn.disabled = false;
        submitBtn.innerHTML = originalText;
    }
}

// 处理回复提交
async function handleReply(parentId, e) {
    e.preventDefault();
    
    if (!isLoggedIn()) {
        showLogin();
        return;
    }
    
    const form = e.target;
    const content = form.querySelector('textarea').value.trim();
    
    if (!content) {
        showGlobalMessage('请输入回复内容', 'warning', 3000);
        return;
    }
    
    const submitBtn = form.querySelector('button[type="submit"]');
    const originalText = submitBtn.innerHTML;
    
    try {
        // 显示提交中状态
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span>';
        
        await apiCall('/comments/reply', 'POST', {
            parent_id: parentId,
            post_id: STATE.currentPostId,
            content: content
        }, true);
        
        // 隐藏回复表单
        form.reset();
        document.getElementById(`replyForm${parentId}`).classList.add('d-none');
        
        // 显示成功提示
        showGlobalMessage('回复发表成功！', 'success', 3000);
        
        // 重新加载评论
        setTimeout(() => {
            loadPostComments(STATE.currentPostId);
        }, 500);
        
    } catch (error) {
        console.error('回复失败:', error);
        showGlobalMessage(`回复失败: ${error.message}`, 'danger', 3000);
    } finally {
        // 恢复按钮状态
        submitBtn.disabled = false;
        submitBtn.innerHTML = originalText;
    }
}

// 点赞评论
async function likeComment(commentId) {
    if (!isLoggedIn()) {
        showLogin();
        return;
    }
    
    const likeBtn = document.getElementById(`commentLikeBtn${commentId}`);
    if (!likeBtn) return;
    
    const originalText = likeBtn.innerHTML;
    const originalClass = likeBtn.className;
    
    try {
        // 显示加载状态
        likeBtn.disabled = true;
        likeBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span>';
        
        await apiCall(`/comments/${commentId}/like`, 'POST', null, true);
        
        // 更新点赞数
        const currentMatch = originalText.match(/\d+/);
        const currentCount = currentMatch ? parseInt(currentMatch[0]) : 0;
        likeBtn.innerHTML = `<i class="bi bi-heart-fill text-danger"></i> ${currentCount + 1}`;
        likeBtn.className = originalClass.replace('btn-outline-primary', 'btn-danger');
        
        // 显示成功提示
        showGlobalMessage('点赞成功！', 'success', 2000);
        
    } catch (error) {
        console.error('点赞评论失败:', error);
        
        // 恢复按钮状态
        likeBtn.disabled = false;
        likeBtn.innerHTML = originalText;
        likeBtn.className = originalClass;
        
        // 检查特定错误
        if (error.message.includes('已经点赞过') || error.message.includes('already liked')) {
            showGlobalMessage('您已经点赞过这条评论了', 'info', 2000);
        } else {
            showGlobalMessage(`点赞失败: ${error.message}`, 'danger', 2000);
        }
    }
}

// 删除评论
async function deleteComment(commentId) {
    if (!confirm('确定要删除这条评论吗？')) {
        return;
    }
    
    try {
        await apiCall(`/comments/${commentId}`, 'DELETE', null, true);
        
        // 显示成功提示
        showGlobalMessage('评论删除成功！', 'success', 3000);
        
        // 重新加载评论
        setTimeout(() => {
            loadPostComments(STATE.currentPostId);
        }, 500);
        
    } catch (error) {
        console.error('删除评论失败:', error);
        showGlobalMessage(`删除失败: ${error.message}`, 'danger', 3000);
    }
}
// 获取评论统计
async function getCommentStats(commentId) {
    try {
        const stats = await apiCall(`/comments/${commentId}/likes`);
        return stats.count || 0;
    } catch (error) {
        console.error('获取评论统计失败:', error);
        return 0;
    }
}