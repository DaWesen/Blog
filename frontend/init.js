/**
 * 初始化脚本 - 用于绑定事件监听器
 */

// 初始化所有事件监听器
function initEventListeners() {
    console.log('初始化事件监听器...');
    
    // 绑定导航栏事件
    document.querySelectorAll('a[onclick]').forEach(link => {
        const originalOnClick = link.getAttribute('onclick');
        link.removeAttribute('onclick');
        link.addEventListener('click', function(e) {
            e.preventDefault();
            eval(originalOnClick);
        });
    });
    
    // 绑定按钮事件
    document.querySelectorAll('button[onclick]').forEach(button => {
        const originalOnClick = button.getAttribute('onclick');
        button.removeAttribute('onclick');
        button.addEventListener('click', function(e) {
            eval(originalOnClick);
        });
    });
    
    // 绑定表单提交事件
    const forms = ['loginForm', 'registerForm', 'postForm', 'categoryForm', 'commentForm', 'profileForm'];
    forms.forEach(formId => {
        const form = document.getElementById(formId);
        if (form) {
            form.addEventListener('submit', function(e) {
                e.preventDefault();
                console.log(`表单提交: ${formId}`);
            });
        }
    });
}

// 初始化头像事件
function initAvatarEvents() {
    // 头像上传预览
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
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', function() {
    console.log('DOM加载完成，初始化事件监听器');
    setTimeout(() => {
        initEventListeners();
        initAvatarEvents();
    }, 100);
});

// 暴露初始化函数
window.initEventListeners = initEventListeners;
window.initAvatarEvents = initAvatarEvents;