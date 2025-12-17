/**
 * 分类模块 - 处理分类的增删改查
 */

// 加载分类列表
async function loadCategories() {
    const container = document.getElementById('categoriesTable');
    
    // 显示加载中
    container.innerHTML = `
        <tr>
            <td colspan="6" class="text-center py-4">
                <div class="spinner-border spinner-border-sm text-primary" role="status">
                    <span class="visually-hidden">加载中...</span>
                </div>
                加载中...
            </td>
        </tr>
    `;
    
    try {
        // 修改这里：使用 /api/categories/all 而不是 /api/categories
        const categories = await apiCall('/categories/all');
        
        if (!categories || categories.length === 0) {
            container.innerHTML = `
                <tr>
                    <td colspan="6" class="text-center py-4 text-muted">
                        <i class="bi bi-tag display-6"></i>
                        <p class="mt-2">暂无分类</p>
                        <button class="btn btn-sm btn-success" onclick="showCreateCategory()">
                            <i class="bi bi-plus-circle"></i> 创建第一个分类
                        </button>
                    </td>
                </tr>
            `;
            return;
        }
        
        // 渲染分类表格
        let html = '';
        categories.forEach(category => {
            html += `
                <tr>
                    <td>${category.id}</td>
                    <td>
                        <span class="badge bg-primary">${escapeHtml(category.name)}</span>
                    </td>
                    <td>
                        <code class="text-muted">${category.slug || '-'}</code>
                    </td>
                    <td>
                        <span class="badge bg-secondary">${category.post_count || 0}</span>
                    </td>
                    <td>${formatDate(category.created_at)}</td>
                    <td>
                        <div class="btn-group btn-group-sm">
                            <button class="btn btn-outline-primary" onclick="showEditCategory(${category.id})">
                                <i class="bi bi-pencil"></i>
                            </button>
                            <button class="btn btn-outline-danger" onclick="deleteCategory(${category.id})">
                                <i class="bi bi-trash"></i>
                            </button>
                        </div>
                    </td>
                </tr>
            `;
        });
        
        container.innerHTML = html;
        
    } catch (error) {
        container.innerHTML = `
            <tr>
                <td colspan="6" class="text-center py-4 text-danger">
                    <i class="bi bi-exclamation-triangle"></i> 加载分类失败: ${error.message}
                </td>
            </tr>
        `;
    }
}

// 处理分类提交
async function handleCategorySubmit(e) {
    e.preventDefault();
    
    const categoryId = document.getElementById('categoryId').value;
    const name = document.getElementById('categoryName').value.trim();
    const slug = document.getElementById('categorySlug').value.trim();
    
    if (!name) {
        showGlobalMessage('请输入分类名称', 'warning', 3000);
        return;
    }
    
    const submitBtn = document.getElementById('categorySubmitBtn');
    const originalText = submitBtn.innerHTML;
    
    const categoryData = { name: name };
    if (slug) {
        categoryData.slug = slug.toLowerCase().replace(/\s+/g, '-');
    }
    
    const isUpdate = !!categoryId;
    
    try {
        // 显示加载状态
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm"></span> 处理中...';
        
        let data;
        if (isUpdate) {
            console.log(`更新分类 ${categoryId}:`, categoryData);
            data = await apiCall(`/categories/${categoryId}`, 'PUT', categoryData, true);
        } else {
            console.log('创建分类:', categoryData);
            data = await apiCall('/categories', 'POST', categoryData, true);
        }
        
        // 显示成功提示
        showGlobalMessage(`${isUpdate ? '更新' : '创建'}分类成功！`, 'success', 3000);
        
        // 清空表单
        document.getElementById('categoryForm').reset();
        
        setTimeout(() => {
            showCategories();
        }, 1500);
        
    } catch (error) {
        console.error(`${isUpdate ? '更新' : '创建'}分类失败:`, error);
        showGlobalMessage(`${isUpdate ? '更新' : '创建'}失败: ${error.message}`, 'danger', 3000);
    } finally {
        // 恢复按钮状态
        submitBtn.disabled = false;
        submitBtn.innerHTML = originalText;
    }
}

// 删除分类
async function deleteCategory(categoryId) {
    if (!confirm('确定要删除这个分类吗？分类下的文章将不会被删除，但会失去分类信息。')) {
        return;
    }
    
    try {
        console.log(`删除分类 ${categoryId}...`);
        await apiCall(`/categories/${categoryId}`, 'DELETE', null, true);
        
        showGlobalMessage('分类删除成功！', 'success', 3000);
        
        // 重新加载分类列表
        setTimeout(() => {
            loadCategories();
        }, 500);
        
    } catch (error) {
        console.error('删除分类失败:', error);
        showGlobalMessage(`删除失败: ${error.message}`, 'danger', 3000);
    }
}
// 获取所有分类 - 使用新的 /api/categories/all 接口
async function getCategories() {
    return await apiCall('/categories/all');
}

// 获取单个分类
async function getCategory(categoryId) {
    return await apiCall(`/categories/${categoryId}`);
}

// 新增：搜索分类
async function searchCategories(keyword) {
    if (!keyword || keyword.trim() === '') {
        return await getCategories();
    }
    return await apiCall(`/categories/search?keyword=${encodeURIComponent(keyword)}`);
}