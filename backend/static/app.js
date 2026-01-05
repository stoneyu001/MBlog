// --- 图标路径库 (Heroicons Outline) ---
const iconPaths = {
    // 侧边栏 & 导航
    collection: "M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10",
    chart: "M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z",
    menu: "M4 6h16M4 12h16M4 18h16",
    search: "M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z",

    // 操作
    download: "M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12", // 用于导入
    plus: "M12 4v16m8-8H4",
    pencil: "M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z",
    trash: "M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16",
    check: "M10 .5a9.5 9.5 0 1 0 9.5 9.5A9.51 9.51 0 0 0 10 .5Zm3.707 8.207-4 4a1 1 0 0 1-1.414 0l-2-2a1 1 0 0 1 1.414-1.414L9 10.586l3.293-3.293a1 1 0 0 1 1.414 1.414Z",
    spinner: "M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z",

    // 文件类型
    tech: "M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4", // </>
    life: "M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z", // Sun
    document: "M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z", // Generic Doc
    paperclip: "M8 4a3 3 0 00-3 3v4a5 5 0 0010 0V7a1 1 0 112 0v4a7 7 0 11-14 0V7a5 5 0 0110 0v4a3 3 0 11-6 0V7a1 1 0 012 0v4a1 1 0 102 0V7a3 3 0 00-3-3z", // File list item in modal
    upload: "M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" // Upload dashed box icon
};

function adminApp() {
    return {
        // --- 核心数据 ---
        icons: iconPaths, // 暴露给 HTML 使用
        currentView: 'files',
        files: [],
        searchQuery: '',
        isBuilding: false,
        toast: { show: false, message: '' },

        importModal: {
            show: false,
            category: 'tech',
            files: []
        },

        modal: {
            show: false,
            type: 'new',
            directory: 'tech',
            filename: '',
            content: '',
            originalFilename: ''
        },

        // --- 初始化 ---
        init() {
            this.loadFileList();
        },

        get filteredFiles() {
            if (!this.searchQuery) return this.files;
            return this.files.filter(f => f.toLowerCase().includes(this.searchQuery.toLowerCase()));
        },

        showToastMsg(msg) {
            this.toast.message = msg;
            this.toast.show = true;
            setTimeout(() => { this.toast.show = false; }, 3000);
        },

        // --- API 模拟与交互 ---
        loadFileList() {
            fetch('/api/files', { credentials: 'include' })
                .then(res => {
                    if (res.status === 401) {
                        window.location.href = '/login';
                        return [];
                    }
                    return res.json();
                })
                .then(files => { this.files = files.sort(); })
                .catch(err => {
                    console.error('Failed to load files:', err);
                    this.showToastMsg('加载文件列表失败');
                });
        },

        rebuildSite() {
            this.isBuilding = true;
            fetch('/api/build', { method: 'POST', credentials: 'include' })
                .then(res => {
                    setTimeout(() => {
                        this.isBuilding = false;
                        if (res.ok) this.showToastMsg('站点构建成功');
                        else this.showToastMsg('站点构建失败');
                    }, 1000);
                })
                .catch(err => {
                    this.isBuilding = false;
                    this.showToastMsg('请求失败');
                });
        },

        // --- 模态框逻辑 ---
        openModal(type, file = null) {
            this.modal.type = type;
            this.modal.show = true;
            if (type === 'edit' && file) {
                this.modal.originalFilename = file;
                if (file.startsWith('tech/')) {
                    this.modal.directory = 'tech';
                    this.modal.filename = file.replace('tech/', '');
                } else if (file.startsWith('life/')) {
                    this.modal.directory = 'life';
                    this.modal.filename = file.replace('life/', '');
                } else {
                    this.modal.directory = 'tech';
                    this.modal.filename = file;
                }

                fetch(`/api/files/${file}`, { credentials: 'include' })
                    .then(res => res.text())
                    .then(content => { this.modal.content = content; })
                    .catch(() => this.showToastMsg('加载文件内容失败'));
            } else {
                this.modal.directory = 'tech';
                this.modal.filename = '';
                this.modal.content = '';
                this.modal.originalFilename = '';
            }
        },

        closeModal() { this.modal.show = false; },

        saveFile() {
            if (!this.modal.filename) { alert('请输入文件名'); return; }
            let fullFilename = this.modal.filename;
            if (!fullFilename.startsWith('tech/') && !fullFilename.startsWith('life/')) {
                fullFilename = `${this.modal.directory}/${this.modal.filename}`;
            }

            fetch('/api/files', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                credentials: 'include',
                body: JSON.stringify({
                    filename: fullFilename,
                    content: this.modal.content,
                    originalFilename: this.modal.originalFilename
                })
            }).then(res => {
                if (res.ok) {
                    this.showToastMsg('文件保存成功');
                    this.closeModal();
                    this.loadFileList();
                } else this.showToastMsg('保存失败');
            });
        },

        deleteFile(file) {
            if (!confirm(`确定要删除文件 ${file} 吗？`)) return;
            fetch(`/api/files/${file}`, { method: 'DELETE', credentials: 'include' })
                .then(res => {
                    if (res.ok) {
                        this.showToastMsg('文件删除成功');
                        this.loadFileList();
                    } else this.showToastMsg('删除失败');
                });
        },

        // --- 导入逻辑 ---
        openImportModal() { this.importModal.show = true; },
        closeImportModal() { this.importModal.show = false; this.importModal.files = []; },

        handleImportFileSelect(e) {
            const files = e.target.files || e.dataTransfer.files;
            if (!files.length) return;
            this.importModal.files = [
                ...this.importModal.files,
                ...Array.from(files).filter(f => f.name.endsWith('.md'))
            ];
            if (e.target.tagName === 'INPUT') e.target.value = '';
        },

        removeImportFile(index) { this.importModal.files.splice(index, 1); },

        submitImport() {
            if (this.importModal.files.length === 0) return;
            const formData = new FormData();

            this.importModal.files.forEach(file => {
                const newName = `${this.importModal.category}/${file.name}`;
                const newFile = new File([file], newName, { type: file.type });
                formData.append('files', newFile);
            });

            this.showToastMsg('正在上传...');

            fetch('/api/upload', { method: 'POST', body: formData, credentials: 'include' })
                .then(res => res.json())
                .then(data => {
                    if (data.failed > 0) {
                        this.showToastMsg(`导入完成: ${data.success} 成功, ${data.failed} 失败`);
                    } else {
                        this.showToastMsg(`成功导入 ${data.success} 个文件`);
                    }
                    this.closeImportModal();
                    this.loadFileList();
                })
                .catch(err => {
                    console.error('Upload failed:', err);
                    this.showToastMsg('导入失败');
                });
        }
    };
}