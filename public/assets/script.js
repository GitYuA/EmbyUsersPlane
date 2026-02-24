function doLogin() {
    const pass = document.getElementById('loginPass').value;
    const btn = document.querySelector('.btn-submit');
    const originalText = btn.textContent;
    btn.disabled = true; btn.textContent = '验证中...';
    
    fetch('', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
            'X-Requested-With': 'XMLHttpRequest',
            'X-CSRF-Token': document.querySelector('meta[name="csrf-token"]').getAttribute('content')
        },
        body: new URLSearchParams({action: 'login', password: pass})
    })
    .then(r => r.json())
    .then(r => {
        if (r.success) {
            location.reload();
        } else {
            toast(r.message, false);
            btn.disabled = false; btn.textContent = originalText;
        }
    })
    .catch(e => {
        toast('登录请求失败', false);
        btn.disabled = false; btn.textContent = originalText;
    });
}

// 简单的暗黑模式切换逻辑，仅用于登录页
(function() {
    const mode = localStorage.getItem('emby-theme-mode') || 'auto';
    let isDark = false;
    if (mode === 'auto') isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    else isDark = mode === 'dark';
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light');
})();

function setTheme(mode) {
    localStorage.setItem('emby-theme-mode', mode);
    updateTheme();
    updateThemeIcon(mode);
    document.querySelectorAll('input[name="theme"]').forEach(el => {
        el.checked = (el.value === mode);
    });
}

const THEME_ICONS = {
    light: '<path stroke-linecap="round" stroke-linejoin="round" d="M12 3v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591-1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z" />',
    dark: '<path stroke-linecap="round" stroke-linejoin="round" d="M21.752 15.002A9.718 9.718 0 0118 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 003 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 009.002-5.998z" />',
    auto: '<path stroke-linecap="round" stroke-linejoin="round" d="M12 3v18a9 9 0 0 0 0-18z" fill="currentColor" /><path stroke-linecap="round" stroke-linejoin="round" d="M12 3a9 9 0 1 1 0 18V3z" />'
};

function updateThemeIcon(mode) {
    const btn = document.getElementById('themeToggleBtn');
    if (!btn) return;
    const path = THEME_ICONS[mode] || THEME_ICONS.auto;
    btn.innerHTML = `<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="icon-svg">${path}</svg>`;
}

function updateTheme() {
    const mode = localStorage.getItem('emby-theme-mode') || 'auto';
    let isDark = false;
    if (mode === 'auto') {
        isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    } else {
        isDark = mode === 'dark';
    }
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light');
}

function toggleTheme() {
    const current = localStorage.getItem('emby-theme-mode') || 'auto';
    let next = 'light';
    if (current === 'light') next = 'dark';
    else if (current === 'dark') next = 'auto';
    
    setTheme(next);
    const label = next === 'auto' ? '跟随系统' : (next === 'dark' ? '深色模式' : '浅色模式');
    toast('主题已切换: ' + label);
}

(function() {
    if (!localStorage.getItem('emby-theme-mode') && localStorage.getItem('emby-theme')) {
        setTheme(localStorage.getItem('emby-theme'));
    } else {
        setTheme(localStorage.getItem('emby-theme-mode') || 'auto');
    }
    
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (localStorage.getItem('emby-theme-mode') === 'auto') {
            updateTheme();
        }
    });
})();

function toast(m, s) {
    const t = document.createElement('div');
    t.className = 'custom-toast ' + (s ? 'toast-success' : 'toast-error');
    t.textContent = m;
    document.body.appendChild(t);
    setTimeout(() => t.remove(), 3200);
}

function showLoading() { 
    const overlay = document.getElementById('loadingOverlay');
    if(overlay) overlay.style.display = 'flex'; 
}
function hideLoading() { 
    const overlay = document.getElementById('loadingOverlay');
    if(overlay) overlay.style.display = 'none'; 
}

function closeModal(id){ document.getElementById(id).classList.remove('show'); }

const PASS_ICONS = {
    show: '<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="icon-svg"><path stroke-linecap="round" stroke-linejoin="round" d="M3.98 8.223A10.477 10.477 0 001.934 12C3.226 16.338 7.244 19.5 12 19.5c.993 0 1.953-.138 2.863-.395M6.228 6.228A10.45 10.45 0 0112 4.5c4.756 0 8.773 3.162 10.065 7.498a10.523 10.523 0 01-4.293 5.774M6.228 6.228L3 3m3.228 3.228l3.65 3.65m7.894 7.894L21 21m-3.228-3.228l-3.65-3.65m0 0a3 3 0 10-4.243-4.243m4.242 4.242L9.88 9.88" /></svg>',
    hide: '<svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="icon-svg"><path stroke-linecap="round" stroke-linejoin="round" d="M2.036 12.322a1.012 1.012 0 010-.639C3.423 7.51 7.36 4.5 12 4.5c4.638 0 8.573 3.007 9.963 7.178.07.207.07.431 0 .639C20.577 16.49 16.64 19.5 12 19.5c-4.638 0-8.573-3.007-9.963-7.178z" /><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>'
};

function togglePasswordVisibility(btn) {
  const input = btn.previousElementSibling;
  const isPass = input.type === 'password';
  input.type = isPass ? 'text' : 'password';
  btn.innerHTML = isPass ? PASS_ICONS.show : PASS_ICONS.hide;
}

/**
 * 预处理用户数据，生成搜索索引字符串
 * 性能优化：避免在过滤循环中重复调用 toLowerCase()
 */
function preprocessUser(u) {
    if (!u) return;
    if (!u.history && Array.isArray(u.chargeHistory)) u.history = u.chargeHistory; // 兼容旧字段
    const parts = [
        u.name,
        u.group,
        u.openDate,
        formatTime(u.openDate, ''),
        u.lastRecharge,
        formatTime(u.lastRecharge, ''),
        u.lastActivityDate,
        formatTime(u.lastActivityDate),
        u.expireDate,
        u.daysLeft,
        u.status
    ];
    if (u.history && Array.isArray(u.history)) {
        u.history.forEach(h => {
            if (h && h.note) parts.push(h.note);
        });
    }
    u._searchStr = parts.map(p => String(p || '').toLowerCase()).join(' ');
}

const SORT_FIELDS = ['name', 'group', 'openDate', 'lastRecharge', 'lastActivityDate', 'expireDate', 'sortDays', 'status'];
const SORT_FIELD_LABELS = {
    sortDays: '剩余天数',
    expireDate: '到期时间',
    lastRecharge: '最后充值',
    lastActivityDate: '最后登录',
    openDate: '开通日期',
    name: '用户名',
    group: '分组',
    status: '状态'
};

function getSortFieldLabel(field) {
    return SORT_FIELD_LABELS[field] || SORT_FIELD_LABELS.sortDays;
}

function normalizeSortField(field) {
    return SORT_FIELDS.includes(field) ? field : 'sortDays';
}

function normalizeBool(val, defaultVal = true) {
    if (val === undefined || val === null || val === '') return defaultVal;
    if (typeof val === 'boolean') return val;
    if (typeof val === 'number') return val !== 0;
    const v = String(val).toLowerCase();
    if (['1', 'true', 'yes', 'on'].includes(v)) return true;
    if (['0', 'false', 'no', 'off'].includes(v)) return false;
    return defaultVal;
}

const HAS_APP_CONFIG = (typeof appConfig !== 'undefined' && appConfig);
const HAS_STATE = (typeof state !== 'undefined' && state);
const PERMANENT_DATE = (HAS_APP_CONFIG && appConfig.permanentDate) ? appConfig.permanentDate : '2099-12-31';
const STATUS_TEXT = {
    admin: '管理员',
    disabled: '已禁用',
    enabled: '已启用'
};
const COLUMN_OPTIONS = [
    { key: 'group', label: '分组' },
    { key: 'openDate', label: '开通日期' },
    { key: 'lastRecharge', label: '最后充值' },
    { key: 'lastActivityDate', label: '最后登录' },
    { key: 'expireDate', label: '到期时间' },
    { key: 'sortDays', label: '剩余天数' },
    { key: 'status', label: '状态' }
];
const COLUMN_STORAGE_KEY = 'emby-columns';
const COLUMN_ORDER_KEY = 'emby-column-order';
const SORT_STORAGE_KEY = 'emby-last-sort';
const COLUMN_MAP = COLUMN_OPTIONS.reduce((acc, opt) => {
    acc[opt.key] = opt;
    return acc;
}, {});
let columnVisibility = {};
let columnOrder = [];

function isPermanentUser(u) {
    if (!u) return false;
    const expireDate = String(u.expireDate || '');
    const daysLeft = String(u.daysLeft || '');
    return expireDate === '' || expireDate === PERMANENT_DATE || expireDate === '永久' || daysLeft === '永久';
}

function getExpireDays(u) {
    if (!u || isPermanentUser(u)) return null;
    if (u.expireDate) {
        const d = new Date(u.expireDate + 'T23:59:59');
        if (!Number.isNaN(d.getTime())) {
            return Math.floor((d.getTime() - Date.now()) / 86400000);
        }
    }
    if (u.daysLeft !== undefined && u.daysLeft !== null) {
        const parsed = parseInt(u.daysLeft, 10);
        if (!Number.isNaN(parsed)) return parsed;
    }
    return null;
}

function getExpireDisplay(u) {
    if (isPermanentUser(u)) return '永久';
    return (u && u.expireDate) ? u.expireDate : '';
}

function getExpireBadge(days) {
    if (days === null || days === undefined) return '';
    if (days < 0) return '<span class="expiry-badge expired">已过期</span>';
    if (days <= 7) return '<span class="expiry-badge soon">7天内</span>';
    return '';
}

function getSortStorageKey() {
    const sid = HAS_APP_CONFIG ? (appConfig.currentId || 'default') : 'default';
    return `${SORT_STORAGE_KEY}:${sid}`;
}

function loadLastSort() {
    try {
        const raw = localStorage.getItem(getSortStorageKey());
        if (!raw) return null;
        const data = JSON.parse(raw);
        const field = normalizeSortField(data.field || 'sortDays');
        const asc = normalizeBool(data.asc, true);
        return { field, asc };
    } catch (e) {
        return null;
    }
}

function saveLastSort(sort) {
    if (!sort) return;
    try {
        localStorage.setItem(getSortStorageKey(), JSON.stringify({
            field: normalizeSortField(sort.field || 'sortDays'),
            asc: normalizeBool(sort.asc, true)
        }));
    } catch (e) {}
}

function applyDefaultSort(force = false) {
    if (!HAS_APP_CONFIG || !HAS_STATE) return;
    const sid = appConfig.currentId || '';
    if (!force && state.sortServerId === sid) return;
    const last = loadLastSort();
    state.sort = last ? { field: last.field, asc: last.asc } : { field: 'sortDays', asc: true };
    state.sortServerId = sid;
    if (!last) saveLastSort(state.sort);
}

function initTable() {
    if (!HAS_APP_CONFIG || !HAS_STATE) return;
    applyDefaultSort();
    if (state.users) {
        state.users.forEach(preprocessUser);
    }
    updateGroupFilter();
    applyFilter();
}

function updateGroupFilter() {
    const groups = new Set();
    let hasDefault = false;
    
    if (state.users) {
        state.users.forEach(u => {
            if (u.group) groups.add(u.group);
            else hasDefault = true;
        });
    }

    // Update Datalist
    const datalist = document.getElementById('existingGroups');
    if (datalist) {
        datalist.innerHTML = '';
        groups.forEach(g => {
            const opt = document.createElement('option');
            opt.value = g;
            datalist.appendChild(opt);
        });
    }

    // Update Header Filter
    const optionsContainer = document.getElementById('groupFilterOptions');
    const hiddenInput = document.getElementById('groupFilter');
    
    if (optionsContainer && hiddenInput) {
        const currentVal = hiddenInput.value;
        optionsContainer.innerHTML = '';
        
        // 全部选项
        const allOpt = document.createElement('div');
        allOpt.className = 'custom-option' + (currentVal === '' ? ' selected' : '');
        allOpt.dataset.value = '';
        allOpt.textContent = '全部';
        allOpt.onclick = (e) => {
            selectOption('groupFilter', '', '分组 ↕', 'groupFilterOptions', 'applyFilter', e);
        };
        optionsContainer.appendChild(allOpt);
        
        if (hasDefault) {
            const opt = document.createElement('div');
            opt.className = 'custom-option' + (currentVal === '__default__' ? ' selected' : '');
            opt.dataset.value = '__default__';
            opt.textContent = '默认分组';
            opt.onclick = (e) => {
                selectOption('groupFilter', '__default__', '默认分组', 'groupFilterOptions', 'applyFilter', e);
            };
            optionsContainer.appendChild(opt);
        }
        
        Array.from(groups).sort().forEach(g => {
            const opt = document.createElement('div');
            opt.className = 'custom-option' + (currentVal === g ? ' selected' : '');
            opt.dataset.value = g;
            opt.textContent = g;
            opt.onclick = (e) => {
                selectOption('groupFilter', g, g, 'groupFilterOptions', 'applyFilter', e);
            };
            optionsContainer.appendChild(opt);
        });
    }
}

function applyFilter() {
    if (!HAS_STATE) return;
    // 重置选中状态
    state.selectedUserIds = null;
    document.getElementById('selectAll').checked = false;
    
    const term = String(state.search ?? '').toLowerCase();
    const groupFilter = document.getElementById('groupFilter') ? document.getElementById('groupFilter').value : '';
    const statusFilter = document.getElementById('statusFilter') ? document.getElementById('statusFilter').value : '';
    const expireFilter = document.getElementById('expireFilter') ? document.getElementById('expireFilter').value : '';
    const hiddenMap = HAS_APP_CONFIG ? (appConfig.hiddenUsers || {}) : {};
    let hidden = [];
    if (Array.isArray(hiddenMap)) {
        hidden = hiddenMap;
    } else if (HAS_APP_CONFIG) {
        hidden = hiddenMap[appConfig.currentId] || [];
    }
    
    state.filteredUsers = state.users.filter(u => {
        if (!u) return false;
        if (hidden.includes(u.name)) return false;

        // 性能优化：使用预计算的搜索索引
        const matchesSearch = !term || (u._searchStr && u._searchStr.includes(term));
                              
        let matchesGroup = false;
        if (groupFilter === '') matchesGroup = true;
        else if (groupFilter === '__default__') matchesGroup = !u.group;
        else matchesGroup = u.group === groupFilter;
        
        let matchesStatus = true;
        if (statusFilter) {
            const status = u.status || '';
            if (statusFilter === 'admin') matchesStatus = status === STATUS_TEXT.admin;
            else if (statusFilter === 'disabled') matchesStatus = status === STATUS_TEXT.disabled;
            else if (statusFilter === 'enabled') matchesStatus = status === STATUS_TEXT.enabled;
        }

        let matchesExpire = true;
        if (expireFilter) {
            const days = getExpireDays(u);
            if (expireFilter === 'permanent') matchesExpire = isPermanentUser(u);
            else if (expireFilter === 'expired') matchesExpire = days !== null && days < 0;
            else if (expireFilter === 'soon') matchesExpire = days !== null && days >= 0 && days <= 7;
        }

        return matchesSearch && matchesGroup && matchesStatus && matchesExpire;
    });
    state.page = 1;
    applySort();
}

function applySort() {
    const { field, asc } = state.sort;
    state.filteredUsers.sort((a, b) => {
        let va = a[field], vb = b[field];
        if (typeof va === 'string') va = va.toLowerCase();
        if (typeof vb === 'string') vb = vb.toLowerCase();
        if (va < vb) return asc ? -1 : 1;
        if (va > vb) return asc ? 1 : -1;
        return 0;
    });
    updateSortIndicator();
    renderTable();
}

function sortTable(field) {
    if (state.sort.field === field) {
        state.sort.asc = !state.sort.asc;
    } else {
        state.sort.field = field;
        state.sort.asc = true;
    }
    saveLastSort(state.sort);
    applySort();
}

function resetSort() {
    if (!HAS_STATE) return;
    state.sort = { field: 'sortDays', asc: true };
    saveLastSort(state.sort);
    applySort();
}

function updateSortIndicator() {
    const el = document.getElementById('sortIndicator');
    if (el && HAS_STATE) {
        const label = getSortFieldLabel(state.sort.field);
        const dir = state.sort.asc ? '↑' : '↓';
        el.textContent = `排序：${label} ${dir}`;
    }
    document.querySelectorAll('th[data-sort]').forEach(th => {
        const isActive = th.dataset.sort === state.sort.field;
        th.classList.toggle('sorted', isActive);
        th.classList.toggle('asc', isActive && state.sort.asc);
        th.classList.toggle('desc', isActive && !state.sort.asc);
    });
}

function searchUsers() {
    clearTimeout(window.searchTimer);
    window.searchTimer = setTimeout(() => {
        state.search = document.getElementById('searchInput').value;
        applyFilter();
    }, 300);
}

function resetFilters() {
    const searchInput = document.getElementById('searchInput');
    if (searchInput) searchInput.value = '';
    state.search = '';
    if (document.getElementById('groupFilter')) {
        setCustomSelectValue('groupFilter', 'groupFilterOptions', '', '分组 ↕');
    }
    if (document.getElementById('expireFilter')) {
        setCustomSelectValue('expireFilter', 'expireFilterOptions', '', '到期：全部');
    }
    if (document.getElementById('statusFilter')) {
        setCustomSelectValue('statusFilter', 'statusFilterOptions', '', '状态：全部');
    }
    applyFilter();
}

function formatTime(iso, defaultText = '从未登录') {
    if(!iso) return defaultText;
    const d = new Date(iso);
    if(isNaN(d.getTime())) return iso;
    const pad = n => n < 10 ? '0'+n : n;
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function formatLogTime(ts) {
    if (!ts) return '';
    const d = new Date(ts * 1000);
    if (isNaN(d.getTime())) return '';
    const pad = n => n < 10 ? '0' + n : n;
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function updateUserState(updatedUser) {
    // 移除旧版嵌套结构兼容代码，后端现在统一返回扁平结构
    const u = updatedUser;
    const id = u.id;
    const idx = state.users.findIndex(existing => existing.id === id);
    
    // 统一字段提取
    const name = u.name;
    const group = u.group;
    const email = u.email;
    const openDate = u.openDate;
    const lastRecharge = u.lastRecharge;
    const lastActivityDate = u.lastActivityDate;
    const expireDate = u.expireDate;
    const daysLeft = u.daysLeft;
    const status = u.status;
    const history = u.chargeHistory || u.history; // 兼容后端可能返回的字段名差异
    const disabled = u.disabled;

    if (idx !== -1) {
        const u = state.users[idx];
        u.name = name;
        u.group = group;
        u.email = email;
        u.openDate = openDate;
        u.lastRecharge = lastRecharge;
        u.lastActivityDate = lastActivityDate;
        u.expireDate = expireDate;
        u.daysLeft = daysLeft;
        u.status = status;
        u.history = history;
        u.disabled = disabled;
        u.rawData = {
            expire: expireDate,
            open: openDate,
            last: lastRecharge
        };
    } else {
        const newUser = {
            id: id,
            name: name,
            group: group,
            openDate: openDate,
            lastRecharge: lastRecharge || '',
            lastActivityDate: lastActivityDate,
            expireDate: expireDate || '',
            daysLeft: daysLeft || '',
            status: status || '已启用',
            history: history || [],
            disabled: disabled,
            isFirst: true,
            rawData: {
                expire: expireDate,
                open: openDate,
                last: lastRecharge
            }
        };
        state.users.unshift(newUser);
    }
    
    // 更新搜索索引
    const targetUser = idx !== -1 ? state.users[idx] : state.users[0];
    preprocessUser(targetUser);

    updateGroupFilter();
    applyFilter();
}

function removeLocalUser(uid) {
    state.users = state.users.filter(u => u.id !== uid);
    updateGroupFilter();
    applyFilter();
}

function changePageSize() {
    state.pageSize = parseInt(document.getElementById('pageSize').value);
    state.page = 1;
    renderTable();
}

function prevPage() { if(state.page > 1) { state.page--; renderTable(); } }
function nextPage() { 
    const max = Math.ceil(state.filteredUsers.length / state.pageSize);
    if(state.page < max) { state.page++; renderTable(); } 
}

const escapeHtml = (str) => {
    if (!str) return '';
    return String(str)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
};

function updateUserStats() {
    const totalEl = document.getElementById('statTotal');
    if (!totalEl) return;
    const normalEl = document.getElementById('statNormal');
    const disabledEl = document.getElementById('statDisabled');
    const foreverEl = document.getElementById('statForever');
    const adminEl = document.getElementById('statAdmin');
    if (!normalEl || !disabledEl || !foreverEl || !adminEl) return;

        const list = Array.isArray(state.filteredUsers) ? state.filteredUsers : [];
    let normal = 0;
    let disabled = 0;
    let admin = 0;
    let forever = 0;

    list.forEach(u => {
        if (!u) return;
        const status = u.status;
        if (status === STATUS_TEXT.admin) admin++;
        else if (status === STATUS_TEXT.disabled) disabled++;
        else if (status === STATUS_TEXT.enabled) normal++;
        else normal++;

        if (isPermanentUser(u)) {
            forever++;
        }
    });

    totalEl.textContent = list.length;
    normalEl.textContent = normal;
    disabledEl.textContent = disabled;
    foreverEl.textContent = forever;
    adminEl.textContent = admin;
}

function renderTable() {
    const start = (state.page - 1) * state.pageSize;
    const end = start + state.pageSize;
    const pageData = state.filteredUsers.slice(start, end);
    const tbody = document.getElementById('userTableBody');

    updateUserStats();
    
    document.getElementById('totalCount').textContent = state.filteredUsers.length;
    const maxPage = Math.ceil(state.filteredUsers.length / state.pageSize) || 1;
    document.getElementById('pageInfo').textContent = `${state.page} / ${maxPage}`;
    document.getElementById('btnPrev').disabled = state.page === 1;
    document.getElementById('btnNext').disabled = state.page === maxPage;
    
    document.getElementById('selectAll').checked = false;

    if (pageData.length === 0) {
        tbody.innerHTML = '<tr><td colspan="10" class="table-empty">无匹配用户</td></tr>';
        return;
    }

    tbody.innerHTML = pageData.map(u => {
        const safeName = escapeHtml(u.name);
        const uStr = encodeURIComponent(JSON.stringify(u)).replace(/'/g, "%27");
        
        const statusHtml = u.status === '管理员' ? '<span class="status sm">管理员</span>' :
                          (u.status === '已禁用' ? '<span class="status sd">已禁用</span>' : '<span class="status sa">已启用</span>');
        const actionHtml = (u.status === '管理员')
            ? '<button class="btn btn-outline btn-action" disabled title="管理员不可在面板操作">管理员</button>'
            : `<button class="btn btnp btn-action" onclick='openUserActions("${uStr}")'>操作</button>`;
        
                const permanent = isPermanentUser(u);
        const expireDays = getExpireDays(u);
        let daysText = u.daysLeft;
        let daysClass = '';
        
        if (permanent) {
            daysClass = 'forever';
            daysText = '永久';
        } else if (expireDays !== null) {
            if (expireDays < 0) {
                daysClass = 'd0';
                daysText = '已过期' + Math.abs(expireDays) + ' 天';
            } else if (expireDays === 0) {
                daysClass = 'd0';
                daysText = '今天到期';
            } else if (expireDays <= 7) {
                daysClass = 'd7';
                daysText = expireDays + ' 天';
            } else {
                daysText = expireDays + ' 天';
            }
        } else {
            daysText = daysText ? daysText : '-';
        }

        const expireDisplay = getExpireDisplay(u);
        const expireBadge = getExpireBadge(expireDays);
        const expireCell = `<div class="expire-cell"><span>${escapeHtml(expireDisplay || '-')}</span>${expireBadge}</div>`;

        let lastDisplay = escapeHtml(formatTime(u.lastRecharge, '无记录'));
        if (u.isFirst) lastDisplay += ' <small class="first-charge-tip">(首次)</small>';
        
        const isAdmin = u.status === '管理员';
        const isSelected = state.selectedUserIds && state.selectedUserIds.has(u.id);
        const checkboxHtml = isAdmin 
            ? '<input type="checkbox" class="user-checkbox" value="' + u.id + '" disabled title="管理员不可批量操作">'
            : '<input type="checkbox" class="user-checkbox" value="' + u.id + '" onchange="updateBatchBar()" ' + (isSelected ? 'checked' : '') + '>';
        
        return `
          <tr class="user-row">
            <td class="batch-checkbox-cell">${checkboxHtml}</td>
            <td class="col-name" data-label="用户名"><strong class="${u.disabled ? 'disabled-name' : ''}">${safeName}</strong></td>
            <td class="col-group" data-label="分组">${escapeHtml(u.group || '默认')}</td>
            <td class="col-openDate" data-label="开通日期">${escapeHtml(formatTime(u.openDate, '未知'))}</td>
            <td class="col-lastRecharge" data-label="最后充值">${lastDisplay}</td>
            <td class="col-lastActivityDate" data-label="最后登录">${escapeHtml(formatTime(u.lastActivityDate))}</td>
            <td class="col-expireDate" data-label="到期时间">${expireCell}</td>
            <td class="col-sortDays" data-label="剩余天数"><span class="${daysClass}">${daysText}</span></td>
            <td class="col-status" data-label="状态">${statusHtml}</td>
            <td class="col-actions" data-label="操作">\n              <div class="btn-group">
                            ${actionHtml}
              </div>
            </td>
          </tr>
        `;
    }).join('');

    applyColumnOrder();
    updateBatchBar();
}

function post(d, cb, errCb, skipReload = false, showOverlay = true) {
  if (showOverlay) showLoading();
  if (HAS_APP_CONFIG && appConfig.currentId) {
      d.serverId = appConfig.currentId;
  }
  
  const headers = {
      'Content-Type': 'application/x-www-form-urlencoded; charset=UTF-8',
      'X-Requested-With': 'XMLHttpRequest'
  };
  
  const csrfMeta = document.querySelector('meta[name="csrf-token"]');
  if (csrfMeta) {
      headers['X-CSRF-Token'] = csrfMeta.getAttribute('content');
  }

  fetch('', {
    method: 'POST',
    headers: headers,
    body: new URLSearchParams(d)
  })
  .then(r => {
      if (!r.ok) throw new Error(`HTTP ${r.status}: ${r.statusText}`);
      return r.text().then(text => {
          try {
              return JSON.parse(text);
          } catch (e) {
              console.error('Invalid JSON:', text);
              // 截取前100个字符用于显示错误
              const preview = text.length > 100 ? text.substring(0, 100) + '...' : text;
              throw new Error('服务器返回无效数据: ' + preview.replace(/</g, '&lt;'));
          }
      });
  })
  .then(r => {
    if (showOverlay) hideLoading();
    if (r.message) toast(r.message, r.success);
    if (r.success) {
      if (cb) cb(r);
      if (!skipReload) setTimeout(() => location.reload(), 1000);
    } else {
      if (errCb) errCb(r);
    }
  })
  .catch(e => {
    if (showOverlay) hideLoading();
    console.error('Request failed:', e);
    toast('请求失败: ' + e.message, false);
    if (errCb) errCb();
  });
}

let currentUid = '', currentName = '', isCurrentlyDisabled = false;

function renderHistory(container, history) {
  if (!history || history.length === 0) {
    container.innerHTML = '<div class="no-history">暂无充值记录</div>';
    return;
  }
  container.innerHTML = history.slice().reverse().map(h => `
    <div class="timeline-item">
      <div class="timeline-dot"></div>
      <div class="timeline-date">${escapeHtml(h.date || '')} <strong>+${escapeHtml(h.days !== undefined && h.days !== null ? String(h.days) : '0')}天</strong></div>
      <div class="timeline-note">${escapeHtml(h.note || '管理员充值')}</div>
    </div>
  `).join('');
}

function setExpireAction(dateInputId, actionInputId, action) {
  const dateInput = document.getElementById(dateInputId);
  const actionInput = document.getElementById(actionInputId);
  if (!actionInput) return;
  actionInput.value = action;
  if (!dateInput) return;
  if (action === 'permanent') {
    dateInput.value = '';
  } else if (action === 'clear') {
    dateInput.value = '';
  }
}

function bindExpireInputs() {
  const uaExp = document.getElementById('uaExp');
  const uaAction = document.getElementById('uaExpAction');
  if (uaExp && uaAction) {
    uaExp.addEventListener('input', () => {
      uaAction.value = '';
    });
  }
  const bExp = document.getElementById('bExp');
  const bAction = document.getElementById('bExpAction');
  if (bExp && bAction) {
    bExp.addEventListener('input', () => {
      bAction.value = '';
    });
  }
}

let currentUserData = null;

function openUserActions(uStr) {
  try {
    const u = JSON.parse(decodeURIComponent(uStr));
    currentUserData = u;
    currentUid = u.id;
    currentName = u.name;
    
    document.getElementById('actionName').textContent = u.name;
    document.getElementById('uaName').value = ''; 
    document.getElementById('uaGroup').value = u.group || '';
    document.getElementById('uaPass').value = ''; 
    document.getElementById('uaEmail').value = u.email || ''; 
    
    // 强制重置/填充日期字段，防止残留上一个用户的数据
    // 优先使用 rawData，否则尝试从普通字段解析
    const formatDateVal = (v) => {
        if (!v || v === '永久' || v === '未知' || v === PERMANENT_DATE) return '';
        // 提取 YYYY-MM-DD
        if (v.length >= 10) return v.substring(0, 10);
        return v;
    };

    const openVal = (u.rawData && u.rawData.open) || u.openDate;
    const lastVal = (u.rawData && u.rawData.last) || u.lastRecharge;
        let expVal = (u.rawData && u.rawData.expire) || u.expireDate;
    const expActionInput = document.getElementById('uaExpAction');
    if (expActionInput) expActionInput.value = '';
    if (isPermanentUser(u) || expVal === PERMANENT_DATE) expVal = '';

    document.getElementById('uaOpen').value = formatDateVal(openVal);
    document.getElementById('uaLast').value = formatDateVal(lastVal);
    document.getElementById('uaExp').value = formatDateVal(expVal);

    document.getElementById('uaDays').value = '30';

    document.getElementById('uaConfirmDel').value = '';
    document.getElementById('uaNote').value = '';
    
    document.getElementById('userActionType').value = 'charge';
    document.getElementById('uaLabel').textContent = '充值续费';
    document.querySelectorAll('#uaOptions .custom-option').forEach(op => {
        op.classList.remove('selected');
        if (op.textContent === '充值续费') op.classList.add('selected');
    });
    
    const hist = document.getElementById('uaHistory');
    renderHistory(hist, u.history);

    updateUserActionUI();
    document.getElementById('actionModal').classList.add('show');
  } catch(e) { toast('打开操作菜单失败：' + e.message, false); }
}

function updateUserActionUI() {
    const type = document.getElementById('userActionType').value;
    const btn = document.getElementById('uaConfirmBtn');
    
    document.querySelectorAll('.ua-field-group').forEach(el => el.style.display = 'none');
    
    const hist = document.getElementById('uaHistory');
    const hasHistory = currentUserData && currentUserData.history && currentUserData.history.length > 0;
    hist.style.display = (type === 'charge' && hasHistory) ? 'block' : 'none';

    if (type === 'charge') {
        document.getElementById('ua-charge').style.display = 'block';
        btn.textContent = '确认充值';
        btn.className = 'btn btn-lg btn-submit';
    } else if (type === 'edit') {
        document.getElementById('ua-edit').style.display = 'block';
        btn.textContent = '保存修改';
        btn.className = 'btn btn-lg btn-submit';
    } else if (type === 'enable') {
        btn.textContent = '确认启用';
        btn.className = 'btn btn-lg btns';
    } else if (type === 'disable') {
        btn.textContent = '确认禁用';
        btn.className = 'btn btn-lg btn-danger';
    } else if (type === 'delete') {
        document.getElementById('ua-delete').style.display = 'block';
        btn.textContent = '确认删除';
        btn.className = 'btn btn-lg btn-danger';
    }
}

function confirmUserAction() {
    const type = document.getElementById('userActionType').value;
    const note = document.getElementById('uaNote').value;
    const btn = document.getElementById('uaConfirmBtn');
    const originalText = btn.textContent;
    
    let data = { uid: currentUid, note: note };
    
    if (type === 'charge') {
        data.action = 'charge';
        data.charge_uid = currentUid;
        data.charge_days = document.getElementById('uaDays').value;
        data.charge_note = note;
    } else if (type === 'edit') {
        data.action = 'save_edit';
        data.newname = document.getElementById('uaName').value.trim();
        data.newgroup = document.getElementById('uaGroup').value;
        data.newpass = document.getElementById('uaPass').value;
        data.email = document.getElementById('uaEmail').value.trim();
        data.opendate = document.getElementById('uaOpen').value;
        data.lastrecharge = document.getElementById('uaLast').value;
        data.expdate = document.getElementById('uaExp').value;
        const expAction = document.getElementById('uaExpAction') ? document.getElementById('uaExpAction').value : '';
        if (expAction) data.expdate_action = expAction;
        data.disabled = currentUserData.disabled ? 1 : 0;
    } else if (type === 'enable') {
        data.action = 'save_edit';
        data.disabled = 0;
    } else if (type === 'disable') {
        data.action = 'save_edit';
        data.disabled = 1;
    } else if (type === 'delete') {
        const confirm = document.getElementById('uaConfirmDel').value.trim();
        if (confirm !== '确认删除') return toast('请输入“确认删除”', false);
        data.action = 'delete';
    }
    
    btn.disabled = true; btn.textContent = '处理中...';
    
    post(data, (res) => {
        if (res.user) {
            if (type === 'enable') {
                res.user.status = STATUS_TEXT.enabled;
                res.user.disabled = false;
            } else if (type === 'disable') {
                res.user.status = STATUS_TEXT.disabled;
                res.user.disabled = true;
            }
            updateUserState(res.user);
        } else if (type === 'enable' || type === 'disable') {
            refreshUserList();
        }
        if (type === 'delete') removeLocalUser(currentUid);
        closeModal('actionModal');
        btn.disabled = false; btn.textContent = originalText;
    }, () => {
        btn.disabled = false; btn.textContent = originalText;
    }, true, false);
}

function openCreateUser() {
    document.getElementById('newUserName').value = '';
    document.getElementById('newUserPass').value = '';
    document.getElementById('newUserEmail').value = '';
    document.getElementById('newUserDays').value = '30';
    document.getElementById('newUserNote').value = '';
    
    const optionsContainer = document.getElementById('newUserCopyFromOptions');
    const hiddenInput = document.getElementById('newUserCopyFrom');
    const labelSpan = document.getElementById('newUserCopyFromLabel');

    if (optionsContainer && hiddenInput && labelSpan) {
        optionsContainer.innerHTML = '';
        
        // 添加默认选项
        const defaultOption = document.createElement('div');
        defaultOption.className = 'custom-option';
        defaultOption.textContent = '不复制（默认设置）';
        defaultOption.onclick = (e) => selectOption('newUserCopyFrom', '', '不复制（默认设置）', 'newUserCopyFromOptions', '', e);
        optionsContainer.appendChild(defaultOption);

        if (state.users && state.users.length > 0) {
            const sortedUsers = [...state.users].sort((a, b) => (a.name || '').localeCompare(b.name || ''));
            sortedUsers.forEach(u => {
                const opt = document.createElement('div');
                opt.className = 'custom-option';
                const name = u.name || u.id || '未命名用户';
                opt.textContent = name;
                opt.onclick = (e) => selectOption('newUserCopyFrom', u.id, name, 'newUserCopyFromOptions', '', e);
                optionsContainer.appendChild(opt);
            });
        }
        
        // 恢复上次选择 (从服务器配置)
        const sid = appConfig.currentId;
        const sSettings = (appConfig.serverSettings && appConfig.serverSettings[sid]) ? appConfig.serverSettings[sid] : {};
        const lastSelected = sSettings.defaultTemplateUser || '';
        
        let found = false;
        if (lastSelected) {
            // 检查上次选择的用户是否还存在
            const userExists = state.users.find(u => u.id === lastSelected);
            if (userExists) {
                hiddenInput.value = lastSelected;
                labelSpan.textContent = userExists.name || userExists.id || '未命名用户';
                // 标记选中状态
                Array.from(optionsContainer.children).forEach(child => {
                    if (child.textContent === (userExists.name || userExists.id || '未命名用户')) {
                        child.classList.add('selected');
                    }
                });
                found = true;
            }
        }
        
        if (!found) {
            hiddenInput.value = '';
            labelSpan.textContent = '不复制（默认设置）';
            defaultOption.classList.add('selected');
        }
    }

    document.getElementById('createUserModal').classList.add('show');
}

function createUser() {
    const name = document.getElementById('newUserName').value.trim();
    const pass = document.getElementById('newUserPass').value;
    const group = document.getElementById('newUserGroup').value;
    const email = document.getElementById('newUserEmail').value.trim();
    const days = document.getElementById('newUserDays').value;
    const note = document.getElementById('newUserNote').value;
    const copyFrom = document.getElementById('newUserCopyFrom') ? document.getElementById('newUserCopyFrom').value : '';

    if (!name) return toast('请输入用户名', false);
    
    const btn = document.querySelector('#createUserModal .btn-submit');
    const originalText = btn.textContent;
    btn.disabled = true; btn.textContent = '创建中...';
    
    post({action:'create', name: name, pass: pass, group: group, email: email, days: days, note: note, copyFrom: copyFrom}, (res) => {
        if (res.user) updateUserState(res.user);
        // 更新本地配置缓存
        if (appConfig.currentId) {
            if (!appConfig.serverSettings) appConfig.serverSettings = {};
            if (!appConfig.serverSettings[appConfig.currentId]) appConfig.serverSettings[appConfig.currentId] = {};
            appConfig.serverSettings[appConfig.currentId].defaultTemplateUser = copyFrom;
        }
        closeModal('createUserModal');
        btn.disabled = false; btn.textContent = originalText;
    }, () => {
        btn.disabled = false; btn.textContent = originalText;
    }, true);
}

function toggleSelectAll() {
    const selectAll = document.getElementById('selectAll');
    const isChecked = selectAll.checked;
    
    // 获取所有筛选后的非管理员用户ID
    const validUserIds = new Set();
    if (isChecked && state.filteredUsers) {
        state.filteredUsers.forEach(u => {
            if (u.status !== '管理员') {
                validUserIds.add(u.id);
            }
        });
    }
    
    // 更新当前页面上的checkbox
    const checkboxes = document.querySelectorAll('.user-checkbox:not(:disabled)');
    checkboxes.forEach(cb => cb.checked = isChecked);
    
    // 存储选中的用户ID（用于跨页批量操作）
    state.selectedUserIds = isChecked ? validUserIds : new Set();
    
    updateBatchBar();
}

function toggleMobileCheckboxes() {
    document.body.classList.toggle('show-checkboxes');
    toggleFabMenu();
}

function getSelectedUids() {
    // 如果有全选的跨页选择，使用存储的ID
    if (state.selectedUserIds && state.selectedUserIds.size > 0) {
        return Array.from(state.selectedUserIds);
    }
    // 否则只返回当前页面选中的
    const checkboxes = document.querySelectorAll('.user-checkbox:checked');
    const uids = [];
    checkboxes.forEach(cb => uids.push(cb.value));
    return uids;
}

function openBatchModal(type = 'charge') {
    const uids = getSelectedUids();
    if (uids.length === 0) return toast('请先选择用户', false);
    
    document.getElementById('batchCount').textContent = uids.length;
    if (document.getElementById('batchCountSummary')) {
        document.getElementById('batchCountSummary').textContent = uids.length;
    }
    document.getElementById('batchActionType').value = type;
    
    document.getElementById('bDays').value = '30';
    document.getElementById('bOpen').value = '';
    document.getElementById('bLast').value = '';
    document.getElementById('bExp').value = '';
    if (document.getElementById('bExpAction')) document.getElementById('bExpAction').value = '';
    document.getElementById('bConfirmDel').value = '';
    document.getElementById('bNote').value = '';
    
    updateBatchUI();
    document.getElementById('batchModal').classList.add('show');
}

function updateBatchUI() {
    const type = document.getElementById('batchActionType').value;
    const btn = document.getElementById('batchConfirmBtn');
    
    document.querySelectorAll('.batch-field-group').forEach(el => el.style.display = 'none');
    
    if (type === 'charge') document.getElementById('bf-charge').style.display = 'block';
    else if (type === 'edit') document.getElementById('bf-edit').style.display = 'block';
    else if (type === 'delete') document.getElementById('bf-delete').style.display = 'block';
    
    if (type === 'delete' || type === 'disable') {
        btn.className = 'btn btn-lg btn-danger';
        btn.textContent = type === 'delete' ? '确认删除' : '确认禁用';
    } else {
        btn.className = 'btn btn-lg btn-submit';
        btn.textContent = type === 'charge' ? '确认充值' : (type === 'edit' ? '确认修改' : '确认启用');
    }
}

function confirmBatchOp() {
    const uids = getSelectedUids();
    const type = document.getElementById('batchActionType').value;
    const note = document.getElementById('bNote').value;
    const btn = document.getElementById('batchConfirmBtn');
    
    let data = { action: 'batch', uids: uids.join(','), note: note };
    
    if (type === 'charge') {
        data.type = 'charge';
        data.days = document.getElementById('bDays').value;
    } else if (type === 'edit') {
        data.type = 'edit';
        data.newgroup = document.getElementById('bGroup').value;
        data.opendate = document.getElementById('bOpen').value;
        data.lastrecharge = document.getElementById('bLast').value;
        data.expdate = document.getElementById('bExp').value;
        const expAction = document.getElementById('bExpAction') ? document.getElementById('bExpAction').value : '';
        if (expAction) data.expdate_action = expAction;
        if (!data.newgroup && !data.opendate && !data.lastrecharge && !data.expdate && !expAction && !note) {
            return toast('请至少输入一项修改内容', false);
        }
    } else if (type === 'enable') {
        data.type = 'enable';
    } else if (type === 'disable') {
        data.type = 'disable';
    } else if (type === 'delete') {
        const confirmInput = document.getElementById('bConfirmDel').value.trim();
        if(confirmInput !== '确认删除'){ return toast('请输入“确认删除”以确认操作', false); }
        data.type = 'delete';
    }
    
    const originalText = btn.textContent;
    btn.disabled = true; btn.textContent = '处理中...';
    
    post(data, (res) => {
        if (type === 'delete') {
            uids.forEach(uid => removeLocalUser(uid));
        } else if (res.updatedUsers && res.updatedUsers.length > 0) {
            if (type === 'enable' || type === 'disable') {
                res.updatedUsers.forEach(u => {
                    u.status = (type === 'enable') ? STATUS_TEXT.enabled : STATUS_TEXT.disabled;
                    u.disabled = (type === 'disable');
                });
            }
            res.updatedUsers.forEach(u => updateUserState(u));
        } else if (type === 'enable' || type === 'disable') {
            refreshUserList();
        }
        
        // 清除选中状态
        state.selectedUserIds = null;
        document.getElementById('selectAll').checked = false;
        document.querySelectorAll('.user-checkbox:checked').forEach(cb => cb.checked = false);
        updateBatchBar();
        
        closeModal('batchModal');
        btn.disabled = false; btn.textContent = originalText;
    }, () => {
        btn.disabled = false; btn.textContent = originalText;
    }, true, false);
}

function openLogCenter() {
    const modal = document.getElementById('logCenterModal');
    if (!modal) return;

    const sid = appConfig.currentId;
    const sSettings = (appConfig.serverSettings && appConfig.serverSettings[sid]) ? appConfig.serverSettings[sid] : {};
    const defaults = appConfig.globalDefaults || {};
    const logRetention = document.getElementById('logRetentionDays');
    if (logRetention) {
        logRetention.value = sSettings.logRetentionDays || defaults.logRetentionDays || 30;
    }
    if (document.getElementById('logTypeFilter')) {
        setCustomSelectValue('logTypeFilter', 'logTypeOptions', '', '全部类型');
    }
    modal.classList.add('show');
    loadLogList();
}

function openSettings() {
    switchTab('general');
    renderServerList();
    
    // 获取当前服务器的配置，如果不存在则使用全局默认
    const sid = appConfig.currentId;
    const sSettings = (appConfig.serverSettings && appConfig.serverSettings[sid]) ? appConfig.serverSettings[sid] : {};
    const defaults = appConfig.globalDefaults || {};

    if (document.getElementById('checkTime')) {
        const checkTimeInput = document.getElementById('checkTime');
        const checkTimeValue = sSettings.checkTime || defaults.checkTime || '00:00';
        checkTimeInput.value = normalizeTimeValue(checkTimeValue);
        const displayInput = document.getElementById('checkTimeDisplay');
        if (displayInput) displayInput.value = checkTimeInput.value;
    }
    const autoTaskEnabled = sSettings.autoTaskEnabled !== undefined ? sSettings.autoTaskEnabled : (defaults.autoTaskEnabled !== undefined ? defaults.autoTaskEnabled : true);
    const autoEnabled = !(autoTaskEnabled === false || autoTaskEnabled === 0 || autoTaskEnabled === '0');
    const autoRadio = document.querySelector(`input[name="autoTaskEnabled"][value="${autoEnabled ? '1' : '0'}"]`);
    if (autoRadio) autoRadio.checked = true;
    if (document.getElementById('logRetentionDays')) {
        document.getElementById('logRetentionDays').value = sSettings.logRetentionDays || defaults.logRetentionDays || 30;
    }
    if (document.getElementById('panelPass')) {
        document.getElementById('panelPass').value = appConfig.panelPass || '';
    }
    if (document.getElementById('restoreTemplateUser')) {
        document.getElementById('restoreTemplateUser').value = sSettings.restoreTemplateUser || defaults.restoreTemplateUser || '';
    }
    if (document.getElementById('hiddenUsersInput')) {
        const hiddenMap = appConfig.hiddenUsers || {};
        let hidden = [];
        if (Array.isArray(hiddenMap)) {
            hidden = hiddenMap;
        } else {
            hidden = hiddenMap[sid] || [];
        }
        document.getElementById('hiddenUsersInput').value = hidden.join('\n');
    }
    
    // SMTP Settings (Priority: Server Settings > Initial Config > Empty)
    const smtpHost = sSettings.smtp_host !== undefined ? sSettings.smtp_host : (appConfig.smtp ? appConfig.smtp.host : '');
    const smtpPort = sSettings.smtp_port !== undefined ? sSettings.smtp_port : (appConfig.smtp ? appConfig.smtp.port : '');
    const smtpUser = sSettings.smtp_user !== undefined ? sSettings.smtp_user : (appConfig.smtp ? appConfig.smtp.user : '');
    const smtpPass = sSettings.smtp_pass !== undefined ? sSettings.smtp_pass : (appConfig.smtp ? appConfig.smtp.pass : '');
    const smtpFrom = sSettings.smtp_from !== undefined ? sSettings.smtp_from : (appConfig.smtp ? appConfig.smtp.from : '');
    const smtpSecure = sSettings.smtp_secure !== undefined ? sSettings.smtp_secure : (appConfig.smtp ? appConfig.smtp.secure : 'ssl');
    const notifyDays = sSettings.notify_before_days !== undefined ? sSettings.notify_before_days : ((appConfig.notify && appConfig.notify.before_days !== undefined) ? appConfig.notify.before_days : 3);
    const notifyOnOperation = sSettings.notify_on_operation !== undefined ? sSettings.notify_on_operation : (defaults.notify_on_operation !== undefined ? defaults.notify_on_operation : true);
    const notifyOpEnabled = !(notifyOnOperation === false || notifyOnOperation === 0 || notifyOnOperation === '0');
    const expireAction = sSettings.expireAction || 'disable';

    if (document.getElementById('notify_before_days')) document.getElementById('notify_before_days').value = notifyDays;
    const notifyOpRadio = document.querySelector(`input[name="notify_on_operation"][value="${notifyOpEnabled ? '1' : '0'}"]`);
    if (notifyOpRadio) notifyOpRadio.checked = true;
    
    const expireActionRadio = document.querySelector(`input[name="expireAction"][value="${expireAction}"]`);
    if (expireActionRadio) expireActionRadio.checked = true;

    if (document.getElementById('smtp_host')) document.getElementById('smtp_host').value = smtpHost;
    if (document.getElementById('smtp_port')) document.getElementById('smtp_port').value = smtpPort;
    if (document.getElementById('smtp_user')) document.getElementById('smtp_user').value = smtpUser;
    if (document.getElementById('smtp_pass')) document.getElementById('smtp_pass').value = smtpPass;
    if (document.getElementById('smtp_from')) document.getElementById('smtp_from').value = smtpFrom;
    
    const radio = document.querySelector(`input[name="smtp_secure"][value="${smtpSecure}"]`);
    if (radio) radio.checked = true;
    
    document.getElementById('settingsModal').classList.add('show');
}

function switchTab(tabName) {
    document.querySelectorAll('.settings-nav-item').forEach(item => {
        item.classList.remove('active');
        if (item.getAttribute('onclick').includes(tabName)) {
            item.classList.add('active');
        }
    });
    document.querySelectorAll('.tab-content').forEach(el => el.style.display = 'none');
    const target = document.getElementById('tab-' + tabName);
    if(target) target.style.display = 'block';
    if (tabName === 'log') {
        loadLogList();
    }
}

let logEntries = [];

function getLogTags(message) {
    if (!message) return [];
    const tags = new Set();
    if (message.includes('用户充值')) tags.add('recharge');
    if (message.includes('创建用户')) tags.add('create');
    if (message.includes('修改资料')) tags.add('edit');
    if (message.includes('删除用户')) tags.add('delete');
    if (message.includes('禁用用户') || message.includes('状态:禁用')) tags.add('disable');
    if (message.includes('启用用户') || message.includes('状态:启用')) tags.add('enable');
    if (message.includes('自动检查')) tags.add('auto');
    return Array.from(tags);
}

function parseLogEntries(content) {
    if (!content) return [];
    return content.split(/\r?\n/).filter(line => {
        const trimmed = line.trim();
        return trimmed && !trimmed.startsWith('...(仅显示') && !trimmed.startsWith('...(仅显示最近');
    }).map(line => {
        const match = line.match(/^\[(.*?)\]\s+\[(.*?)\]\s+(.*)$/);
        const entry = {
            time: '',
            ip: '',
            message: line,
            raw: line,
            tags: []
        };
        if (match) {
            entry.time = match[1] || '';
            entry.ip = match[2] || '';
            entry.message = match[3] || '';
        }
        entry.tags = getLogTags(entry.message);
        return entry;
    });
}

function highlightText(text, keyword) {
    if (!keyword) return escapeHtml(text);
    const lowerText = text.toLowerCase();
    const lowerKeyword = keyword.toLowerCase();
    let result = '';
    let index = 0;
    while (true) {
        const pos = lowerText.indexOf(lowerKeyword, index);
        if (pos === -1) {
            result += escapeHtml(text.slice(index));
            break;
        }
        result += escapeHtml(text.slice(index, pos));
        result += '<mark>' + escapeHtml(text.slice(pos, pos + lowerKeyword.length)) + '</mark>';
        index = pos + lowerKeyword.length;
    }
    return result;
}

function renderLogHistory() {
    const listEl = document.getElementById('logHistoryList');
    if (!listEl) return;
    const filter = document.getElementById('logTypeFilter') ? document.getElementById('logTypeFilter').value : '';
    const keywordRaw = document.getElementById('logKeyword') ? document.getElementById('logKeyword').value.trim() : '';
    const keyword = keywordRaw.toLowerCase();
    const labelMap = {
        recharge: '充值',
        edit: '修改',
        disable: '禁用',
        enable: '启用',
        delete: '删除',
        create: '创建',
        auto: '自动'
    };
    const items = (logEntries || []).filter(entry => {
        if (filter && (!entry.tags || !entry.tags.includes(filter))) return false;
        if (!keyword) return true;
        const rawMessage = entry.message || entry.raw || '';
        const displayMessage = rawMessage.replace(/\s*\(ID:\s*[^)]+\)/g, '').trim();
        return displayMessage.toLowerCase().includes(keyword);
    });
    if (items.length === 0) {
        listEl.innerHTML = '<div class="log-empty">暂无记录</div>';
        return;
    }
    listEl.innerHTML = items.slice().reverse().map(entry => {
        const tagsHtml = (entry.tags || []).map(tag => {
            const label = labelMap[tag] || tag;
            return `<span class="log-tag ${tag}">${escapeHtml(label)}</span>`;
        }).join('');
        const timeText = entry.time || '';
        const ipText = entry.ip ? ` • ${escapeHtml(entry.ip)}` : '';
        const rawMessage = entry.message || entry.raw || '';
        const displayMessage = rawMessage.replace(/\s*\(ID:\s*[^)]+\)/g, '').trim();
        const messageHtml = highlightText(displayMessage, keyword);
        return `
          <div class="log-item">
            <div class="log-item-head">
              <div class="log-time">${escapeHtml(timeText)}${ipText}</div>
              <div class="log-tags">${tagsHtml}</div>
            </div>
            <div class="log-message">${messageHtml}</div>
          </div>
        `;
    }).join('');
}

function downloadLogFile(mode) {
    const select = document.getElementById('logFileSelect');
    const file = select ? select.value : '';
    if (!file) return toast('请选择日志文件', false);

    const csrfMeta = document.querySelector('meta[name="csrf-token"]');
    const token = csrfMeta ? csrfMeta.getAttribute('content') : '';
    const downloadMode = mode || 'tail';
    let url = '?action=download_log&file=' + encodeURIComponent(file) + '&token=' + encodeURIComponent(token);
    if (downloadMode === 'tail') {
        url += '&tail=1';
    }
    window.location.href = url;
}

function downloadLogWithMode(mode, e) {
    if (e) e.stopPropagation();
    const opts = document.getElementById('logDownloadActionOptions');
    if (opts) {
        opts.classList.remove('show');
        opts.style.display = 'none';
    }
    downloadLogFile(mode);
}

function refreshLogContent() {
    const select = document.getElementById('logFileSelect');
    if (!select) return;
    loadLogList(select.value);
}

function loadLogList(selectedName) {
    const select = document.getElementById('logFileSelect');
    const optionsContainer = document.getElementById('logFileOptions');
    const labelEl = document.getElementById('logFileLabel');
    const historyEl = document.getElementById('logHistoryList');
    const selectContainer = select ? select.closest('.custom-select-container') : null;
    if (!select || !optionsContainer || !labelEl) return;

    optionsContainer.innerHTML = '';
    labelEl.textContent = '加载中...';
    select.value = '';

    post({
        action: 'get_logs',
        file: selectedName || ''
    }, (res) => {
        const files = res.files || [];
        optionsContainer.innerHTML = '';

        if (files.length === 0) {
            labelEl.textContent = '暂无日志';
            select.value = '';
            logEntries = [];
            if (selectContainer) {
                selectContainer.classList.remove('is-single');
                delete selectContainer.dataset.single;
            }
            if (historyEl) historyEl.innerHTML = '<div class="log-empty">暂无记录</div>';
            return;
        }

        if (selectContainer) {
            if (files.length <= 1) {
                selectContainer.classList.add('is-single');
                selectContainer.dataset.single = '1';
            } else {
                selectContainer.classList.remove('is-single');
                delete selectContainer.dataset.single;
            }
        }

        const labelMap = {};
        files.forEach(f => {
            const opt = document.createElement('div');
            opt.className = 'custom-option';
            opt.dataset.value = f.name;
            const timeLabel = f.mtime ? formatLogTime(f.mtime) : '';
            const label = timeLabel ? `${f.name} (${timeLabel})` : f.name;
            labelMap[f.name] = label;
            opt.textContent = label;
            opt.onclick = (e) => {
                selectOption('logFileSelect', f.name, label, 'logFileOptions', 'refreshLogContent', e);
            };
            optionsContainer.appendChild(opt);
        });

        const selected = res.selected || files[0].name;
        setCustomSelectValue('logFileSelect', 'logFileOptions', selected, labelMap[selected] || selected);
        logEntries = parseLogEntries(res.content || '');
        renderLogHistory();
    }, () => {
        labelEl.textContent = '加载失败';
        logEntries = [];
        if (historyEl) historyEl.innerHTML = '<div class="log-empty">暂无记录</div>';
    }, true, false);
}

function runAutoCheckNow(btnElement) {
    let btn = btnElement;
    if (!btn || !(btn instanceof Element)) {
        btn = document.querySelector('#tab-auto .btn-outline');
    }

    const originalText = btn ? btn.textContent : '立即检查';
    if (btn) { btn.disabled = true; btn.textContent = '检查中...'; }

    post({
        action: 'run_auto_check'
    }, () => {
        refreshUserList(() => {
            if (btn) { btn.disabled = false; btn.textContent = originalText; }
        });
    }, () => {
        if (btn) { btn.disabled = false; btn.textContent = originalText; }
    }, true, true);
}

function saveLogSettings(btnElement) {
    const logRetentionDays = document.getElementById('logRetentionDays') ? document.getElementById('logRetentionDays').value : '';
    let btn = btnElement;
    if (!btn || !(btn instanceof Element)) {
        btn = document.querySelector('#logCenterModal .btn-submit');
    }
    const originalText = btn ? btn.textContent : '保存设置';
    if (btn) { btn.disabled = true; btn.textContent = '保存中...'; }

    post({
        action: 'settings_op',
        logRetentionDays: logRetentionDays
    }, (res) => {
        if (res.serverSettings) {
            if (!appConfig.serverSettings) appConfig.serverSettings = {};
            if (!appConfig.serverSettings[appConfig.currentId]) appConfig.serverSettings[appConfig.currentId] = {};
            appConfig.serverSettings[appConfig.currentId] = res.serverSettings;
        }
        if (res.hiddenUsers !== undefined) {
            appConfig.hiddenUsers = res.hiddenUsers;
        }
        if (btn) { btn.disabled = false; btn.textContent = originalText; }
    }, () => {
        if (btn) { btn.disabled = false; btn.textContent = originalText; }
    }, true);
}

function saveSettings(btnElement) {
    const checkTime = document.getElementById('checkTime').value;
    const logRetentionDays = document.getElementById('logRetentionDays') ? document.getElementById('logRetentionDays').value : '';
    const panelPass = document.getElementById('panelPass') ? document.getElementById('panelPass').value : '';
    const restoreTemplateUser = document.getElementById('restoreTemplateUser') ? document.getElementById('restoreTemplateUser').value : '';
    const hiddenUsers = document.getElementById('hiddenUsersInput') ? document.getElementById('hiddenUsersInput').value : '';
    
    const smtp_host = document.getElementById('smtp_host') ? document.getElementById('smtp_host').value : '';
    const smtp_port = document.getElementById('smtp_port') ? document.getElementById('smtp_port').value : '';
    const smtp_user = document.getElementById('smtp_user') ? document.getElementById('smtp_user').value : '';
    const smtp_pass = document.getElementById('smtp_pass') ? document.getElementById('smtp_pass').value : '';
    const smtp_from = document.getElementById('smtp_from') ? document.getElementById('smtp_from').value : '';
    const smtp_secure = document.querySelector('input[name="smtp_secure"]:checked') ? document.querySelector('input[name="smtp_secure"]:checked').value : 'ssl';
    const notify_before_days = document.getElementById('notify_before_days') ? document.getElementById('notify_before_days').value : '';
    const notify_on_operation = document.querySelector('input[name="notify_on_operation"]:checked') ? document.querySelector('input[name="notify_on_operation"]:checked').value : '1';
    const expireAction = document.querySelector('input[name="expireAction"]:checked') ? document.querySelector('input[name="expireAction"]:checked').value : 'disable';
    const autoTaskEnabled = document.querySelector('input[name="autoTaskEnabled"]:checked') ? document.querySelector('input[name="autoTaskEnabled"]:checked').value : '1';
    
    let btn = btnElement;
    if (!btn || !(btn instanceof Element)) {
        btn = document.querySelector('.settings-content .tab-content[style*="block"] .btn-submit');
    }
    
    const originalText = btn ? btn.textContent : '保存';
    if(btn) { btn.disabled = true; btn.textContent = '保存中...'; }
    
    post({
        action: 'settings_op',
        checkTime: checkTime,
        logRetentionDays: logRetentionDays,
        panelPass: panelPass,
        restoreTemplateUser: restoreTemplateUser,
        hiddenUsers: hiddenUsers,
        autoTaskEnabled: autoTaskEnabled,
        smtp_host: smtp_host,
        smtp_port: smtp_port,
        smtp_user: smtp_user,
        smtp_pass: smtp_pass,
        smtp_from: smtp_from,
        smtp_secure: smtp_secure,
        notify_before_days: notify_before_days,
        notify_on_operation: notify_on_operation,
        expireAction: expireAction,
        serverId: appConfig.currentId
    }, (res) => {
        if (res.serverSettings) {
            if (!appConfig.serverSettings) appConfig.serverSettings = {};
            appConfig.serverSettings[appConfig.currentId] = res.serverSettings;
        }
        if (res.hiddenUsers !== undefined) {
            appConfig.hiddenUsers = res.hiddenUsers;
            applyFilter();
        }
        // 更新本地配置中的密码，防止下次打开设置时显示旧值
        appConfig.panelPass = panelPass;
        
        if(btn) { btn.disabled = false; btn.textContent = originalText; }
        toast('设置已保存');
    }, () => {
        if(btn) { btn.disabled = false; btn.textContent = originalText; }
    }, true);
}

function renderServerList() {
    const list = document.getElementById('serverList');
    list.innerHTML = '';
    if (!appConfig.servers || appConfig.servers.length === 0) {
        list.innerHTML = '<div class="empty-state">暂无服务器，请点击下方按钮添加</div>';
        return;
    }
    
    appConfig.servers.forEach(s => {
        const isCurrent = s.id === appConfig.currentId;
        const item = document.createElement('div');
        item.className = `server-item ${isCurrent ? 'active' : ''}`;
        const safeName = escapeHtml(s.name || '未命名');
        const safeUrl = escapeHtml(s.url || '');
        const jsSafeId = String(s.id || '').replace(/\\/g, '\\\\').replace(/'/g, "\\'");
        const jsSafeName = String(s.name || '').replace(/\\/g, '\\\\').replace(/'/g, "\\'");
        
        item.innerHTML = `
            <div class="server-name">
                <svg fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="icon-svg server-icon ${isCurrent ? 'is-current' : ''}"><path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" /></svg>
                ${safeName} 
                ${isCurrent ? '<span class="status sa status-xs">当前使用</span>' : ''}
            </div>
            <div class="server-url">${safeUrl}</div>
            <div class="server-actions">
                ${!isCurrent ? `<button class="btn btns btn-compact" onclick="switchServer('${jsSafeId}', '${jsSafeName}', event)">切换</button>` : ''}
                <button class="btn btn-outline btn-compact" onclick="editServer('${jsSafeId}', event)">编辑</button>
            </div>
        `;
        list.appendChild(item);
    });
}

function openAddServerModal() {
    document.getElementById('serverEditTitle').textContent = '新增服务器';
    document.getElementById('editServerId').value = '';
    document.getElementById('editServerName').value = '';
    document.getElementById('editServerUrl').value = '';
    document.getElementById('editServerKey').value = '';
    document.getElementById('btnDelServerModal').style.display = 'none';
    document.getElementById('serverEditModal').classList.add('show');
}

function editServer(id, e) {
    if(e) e.stopPropagation();
    const s = appConfig.servers.find(x => x.id === id);
    if (!s) return;
    
    document.getElementById('serverEditTitle').textContent = '编辑服务器';
    document.getElementById('editServerId').value = s.id;
    document.getElementById('editServerName').value = s.name;
    document.getElementById('editServerUrl').value = s.url;
    document.getElementById('editServerKey').value = s.key;
    document.getElementById('btnDelServerModal').style.display = 'block';
    document.getElementById('serverEditModal').classList.add('show');
}

function saveServer() {
    const id = document.getElementById('editServerId').value;
    const name = document.getElementById('editServerName').value.trim();
    const url = document.getElementById('editServerUrl').value.trim();
    const key = document.getElementById('editServerKey').value.trim();
    
    if (!url || !key) return toast('请填写完整信息', false);
    
    const btn = document.querySelector('#serverEditModal .btn-submit');
    const originalText = btn.textContent;
    btn.disabled = true; btn.textContent = '保存中...';
    
    post({
        action: 'server_op',
        sub_action: 'save',
        id: id,
        name: name,
        url: url,
        key: key
    }, (res) => {
        if (res.servers) {
            appConfig.servers = res.servers;
            appConfig.currentId = res.currentServerId;
        }
        renderServerList();
        closeModal('serverEditModal');
        
        refreshUserList(() => {
            btn.disabled = false; btn.textContent = originalText;
            toast('服务器保存成功');
        });
    }, () => {
        btn.disabled = false; btn.textContent = originalText;
    }, true);
}

function deleteServer() {
    const id = document.getElementById('editServerId').value;
    if (!id) return;
    if (!confirm('确定要删除此服务器配置吗？')) return;
    
    const btn = document.getElementById('btnDelServerModal');
    const originalText = btn.textContent;
    btn.disabled = true; btn.textContent = '删除中...';

    post({
        action: 'server_op',
        sub_action: 'delete',
        id: id
    }, (res) => {
        if (res.servers) {
            appConfig.servers = res.servers;
            appConfig.currentId = res.currentServerId;
        }
        renderServerList();
        closeModal('serverEditModal');
        
        refreshUserList(() => {
            btn.disabled = false; btn.textContent = originalText;
            toast('服务器已删除');
        });
    }, () => {
        btn.disabled = false; btn.textContent = originalText;
    }, true);
}

function switchServer(id, name, e) {
    if(e) e.stopPropagation();
    document.getElementById('switchServerId').value = id;
    document.getElementById('switchServerName').textContent = name || '未命名';
    document.getElementById('serverSwitchModal').classList.add('show');
}

function confirmSwitchServer() {
    const id = document.getElementById('switchServerId').value;
    const btn = document.querySelector('#serverSwitchModal .btnp');
    const originalText = btn.textContent;
    btn.disabled = true; btn.textContent = '切换中...';

    post({
        action: 'server_op',
        sub_action: 'switch',
        id: id
    }, (res) => {
        if (res.currentServerId) appConfig.currentId = res.currentServerId;
        renderServerList();
        closeModal('serverSwitchModal');
        
        refreshUserList(() => {
            btn.disabled = false; btn.textContent = originalText;
            toast('切换成功');
        });
    }, () => {
        btn.disabled = false; btn.textContent = originalText;
    }, true);
}

function toggleSelect(id) {
    const el = document.getElementById(id);
    if (!el) return;
    const container = el.parentElement;
    if (container && container.dataset.single === '1') return;
    const isShown = el.classList.contains('show');
    
    document.querySelectorAll('.custom-select-options').forEach(d => {
        d.classList.remove('show');
        d.style.display = 'none';
    });

    if (!isShown) {
        const trigger = el.parentElement;
        const rect = trigger.getBoundingClientRect();
        const viewportHeight = window.innerHeight;
        const viewportWidth = window.innerWidth;
        const alignMode = el.dataset.align || 'left';
        const widthMode = el.dataset.width || 'trigger';
        
        el.style.position = 'fixed';
        if (widthMode === 'auto') {
            el.style.width = 'auto';
            el.style.minWidth = '0';
        } else {
            el.style.width = rect.width + 'px';
            el.style.minWidth = '0';
        }
        
        el.style.visibility = 'hidden';
        el.style.display = 'block';
        const dropdownHeight = el.offsetHeight;
        const dropdownWidth = el.offsetWidth;
        el.style.visibility = 'visible';

        let left = rect.left;
        if (alignMode === 'center') {
            left = rect.left + (rect.width - dropdownWidth) / 2;
        } else if (alignMode === 'right') {
            left = rect.right - dropdownWidth;
        }
        const minLeft = 8;
        const maxLeft = Math.max(minLeft, viewportWidth - dropdownWidth - 8);
        left = Math.min(maxLeft, Math.max(minLeft, left));
        el.style.left = left + 'px';
        
        const spaceBelow = viewportHeight - rect.bottom;
        
        if (spaceBelow < dropdownHeight + 10 && rect.top > dropdownHeight + 10) {
            el.style.top = (rect.top - dropdownHeight - 5) + 'px';
            el.style.transformOrigin = 'bottom center';
        } else {
            el.style.top = (rect.bottom + 5) + 'px';
            el.style.transformOrigin = 'top center';
        }
        
        requestAnimationFrame(() => {
            el.classList.add('show');
        });
    }
}

function selectOption(inputId, val, label, optionsId, callbackName, e) {
    e.stopPropagation();
    document.getElementById(inputId).value = val;
    
    const container = document.querySelector(`.custom-select-container [onclick="toggleSelect('${optionsId}')"]`) || document.getElementById(optionsId).parentElement;
    container.querySelector('.custom-select-trigger span').textContent = label;
    
    const opts = document.getElementById(optionsId);
    opts.querySelectorAll('.custom-option').forEach(op => op.classList.remove('selected'));
    e.target.classList.add('selected');
    
    opts.classList.remove('show');
    opts.style.display = 'none';
    
    if (window[callbackName]) window[callbackName]();
}

function setCustomSelectValue(inputId, optionsId, value, label) {
    const input = document.getElementById(inputId);
    if (input) input.value = value;

    const opts = document.getElementById(optionsId);
    let labelText = label || '';
    if (opts) {
        opts.querySelectorAll('.custom-option').forEach(op => {
            const v = op.getAttribute('data-value');
            const match = v !== null && v === String(value);
            if (match && !labelText) labelText = op.textContent;
            op.classList.toggle('selected', match);
        });
    }

    const container = document.querySelector(`.custom-select-container [onclick="toggleSelect('${optionsId}')"]`) || (opts ? opts.parentElement : null);
    if (container) {
        const span = container.querySelector('.custom-select-trigger span');
        if (span) span.textContent = labelText || '';
    }
}

window.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal-backdrop')) {
        e.target.classList.remove('show');
    }
    if (!e.target.closest('.dropdown')) {
        const dropdowns = document.getElementsByClassName("dropdown");
        for (let i = 0; i < dropdowns.length; i++) {
            dropdowns[i].classList.remove('show');
        }
    }
    if (!e.target.closest('.custom-select-container') && !e.target.closest('.custom-select-options') && !e.target.closest('.input-suggestions-wrapper')) {
        document.querySelectorAll('.custom-select-options').forEach(d => {
            d.classList.remove('show');
            d.style.display = 'none';
        });
    }
});

window.addEventListener('resize', () => {
    document.querySelectorAll('.custom-select-options').forEach(d => {
        d.classList.remove('show');
        d.style.display = 'none';
    });
});
window.addEventListener('scroll', (e) => {
    // 如果是下拉框内部滚动，不关闭
    if (e.target.classList && e.target.classList.contains('custom-select-options')) {
        return;
    }
    document.querySelectorAll('.custom-select-options').forEach(d => {
        d.classList.remove('show');
        d.style.display = 'none';
    });
}, true);

if (HAS_APP_CONFIG && (!appConfig.servers || appConfig.servers.length === 0)) {
    window.addEventListener('DOMContentLoaded', () => {
        toast('请先配置 Emby 服务器信息', false);
        openSettings();
    });
}

function normalizeTimeValue(value) {
    const raw = String(value || '').trim();
    const match = raw.match(/^(\d{1,2}):(\d{1,2})$/);
    if (!match) return '00:00';
    let h = parseInt(match[1], 10);
    let m = parseInt(match[2], 10);
    if (Number.isNaN(h) || Number.isNaN(m)) return '00:00';
    h = Math.max(0, Math.min(23, h));
    m = Math.max(0, Math.min(59, m));
    return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`;
}

function initTimePickers() {
    const pickers = document.querySelectorAll('.time-picker');
    if (!pickers.length) return;

    const pad2 = (n) => String(n).padStart(2, '0');
    const nearestStep = (minute, step) => {
        const max = Math.floor(59 / step) * step;
        const rounded = Math.round(minute / step) * step;
        return Math.max(0, Math.min(max, rounded));
    };
    const setSelected = (container, value) => {
        if (!container) return;
        container.querySelectorAll('.time-option').forEach(btn => {
            btn.classList.toggle('is-selected', parseInt(btn.dataset.value, 10) === value);
        });
    };
    const scrollToSelected = (container) => {
        if (!container) return;
        const selected = container.querySelector('.time-option.is-selected');
        if (selected) selected.scrollIntoView({ block: 'nearest' });
    };

    pickers.forEach((picker) => {
        if (picker._timePicker) return;
        const display = picker.querySelector('.time-input-display');
        const hidden = picker.querySelector('input[type="hidden"]');
        const popover = picker.querySelector('.time-popover');
        const hourList = picker.querySelector('.time-options-hour');
        const minuteList = picker.querySelector('.time-options-minute');
        const stepButtons = Array.from(picker.querySelectorAll('.time-step'));
        const applyBtn = picker.querySelector('.time-apply');
        const cancelBtn = picker.querySelector('.time-cancel');
        const trigger = picker.querySelector('.time-trigger');

        if (!display || !hidden || !popover || !hourList || !minuteList) return;

        let step = parseInt(picker.getAttribute('data-minute-step') || '1', 10);
        if (![1, 5, 10].includes(step)) step = 1;
        let draftHour = 0;
        let draftMinute = 0;

        const syncDisplay = (value) => {
            const normalized = normalizeTimeValue(value);
            hidden.value = normalized;
            display.value = normalized;
        };

        const setDraftFromValue = (value) => {
            const normalized = normalizeTimeValue(value);
            const parts = normalized.split(':');
            draftHour = parseInt(parts[0], 10);
            draftMinute = parseInt(parts[1], 10);
        };

        const updatePreview = () => {
            display.value = `${pad2(draftHour)}:${pad2(draftMinute)}`;
        };

        const buildHours = () => {
            hourList.innerHTML = '';
            for (let h = 0; h < 24; h++) {
                const btn = document.createElement('button');
                btn.type = 'button';
                btn.className = 'time-option';
                btn.dataset.value = String(h);
                btn.textContent = pad2(h);
                btn.addEventListener('click', () => {
                    draftHour = h;
                    setSelected(hourList, draftHour);
                    updatePreview();
                });
                hourList.appendChild(btn);
            }
        };

        const buildMinutes = () => {
            minuteList.innerHTML = '';
            for (let m = 0; m < 60; m += step) {
                const btn = document.createElement('button');
                btn.type = 'button';
                btn.className = 'time-option';
                btn.dataset.value = String(m);
                btn.textContent = pad2(m);
                btn.addEventListener('click', () => {
                    draftMinute = m;
                    setSelected(minuteList, draftMinute);
                    updatePreview();
                });
                minuteList.appendChild(btn);
            }
        };

        const setActiveStep = () => {
            stepButtons.forEach(btn => {
                btn.classList.toggle('is-active', parseInt(btn.dataset.step, 10) === step);
            });
        };

        const openPopover = () => {
            if (picker.classList.contains('is-open')) return;
            const baseValue = hidden.value || display.value || '00:00';
            setDraftFromValue(baseValue);
            draftMinute = nearestStep(draftMinute, step);
            updatePreview();
            setSelected(hourList, draftHour);
            buildMinutes();
            setSelected(minuteList, draftMinute);
            setActiveStep();
            picker.classList.add('is-open');
            popover.setAttribute('aria-hidden', 'false');
            scrollToSelected(hourList);
            scrollToSelected(minuteList);
        };

        const closePopover = (restore) => {
            picker.classList.remove('is-open');
            popover.setAttribute('aria-hidden', 'true');
            if (restore) {
                display.value = hidden.value;
            }
        };

        const applyValue = () => {
            const value = `${pad2(draftHour)}:${pad2(draftMinute)}`;
            hidden.value = value;
            display.value = value;
            closePopover(false);
        };

        stepButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                const next = parseInt(btn.dataset.step, 10);
                step = [1, 5, 10].includes(next) ? next : 1;
                draftMinute = nearestStep(draftMinute, step);
                buildMinutes();
                setSelected(minuteList, draftMinute);
                setActiveStep();
                updatePreview();
                scrollToSelected(minuteList);
            });
        });

        if (trigger) {
            trigger.addEventListener('click', () => openPopover());
        }
        display.addEventListener('click', () => openPopover());
        display.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                openPopover();
            }
        });
        if (applyBtn) applyBtn.addEventListener('click', applyValue);
        if (cancelBtn) cancelBtn.addEventListener('click', () => closePopover(true));

        buildHours();
        buildMinutes();
        syncDisplay(hidden.value || display.value || '00:00');

        picker._timePicker = { close: closePopover };
    });

    if (!initTimePickers._bound) {
        document.addEventListener('click', (e) => {
            document.querySelectorAll('.time-picker.is-open').forEach(picker => {
                if (!picker.contains(e.target)) {
                    const api = picker._timePicker;
                    if (api && typeof api.close === 'function') api.close(true);
                }
            });
        });
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                document.querySelectorAll('.time-picker.is-open').forEach(picker => {
                    const api = picker._timePicker;
                    if (api && typeof api.close === 'function') api.close(true);
                });
            }
        });
        initTimePickers._bound = true;
    }
}
window.addEventListener('DOMContentLoaded', initTable);
window.addEventListener('DOMContentLoaded', initTimePickers);
window.addEventListener('DOMContentLoaded', initColumnControls);
window.addEventListener('DOMContentLoaded', bindExpireInputs);

function toggleFabMenu() {
    const menu = document.getElementById('fabMenu');
    if (!menu) return;
    menu.classList.toggle('show');
    const btn = document.querySelector('.fab-btn');
    if (menu.classList.contains('show')) {
        btn.style.transform = 'rotate(90deg)';
    } else {
        btn.style.transform = 'rotate(0)';
    }
}

window.addEventListener('click', (e) => {
    if (!e.target.closest('.fab-container')) {
        const menu = document.getElementById('fabMenu');
        if (menu) menu.classList.remove('show');
        const btn = document.querySelector('.fab-btn');
        if(btn) btn.style.transform = 'rotate(0)';
    }
});

function refreshUserList(callback) {
    post({action: 'get_users'}, (res) => {
        if (res.users) {
            state.users = res.users.map(u => {
                if (u && !u.history && Array.isArray(u.chargeHistory)) {
                    u.history = u.chargeHistory; // 兼容后端字段名
                }
                return u;
            });
            initTable();
            if (callback) callback();
        }
    }, null, true);
}

function refreshCache(btn) {
    if (!btn) return;
    const originalHtml = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '<div class="spinner spinner-sm"></div>';
    
    post({action: 'refresh_cache'}, () => {
        refreshUserList(() => {
            btn.disabled = false;
            btn.innerHTML = originalHtml;
            toast('刷新成功');
        });
    }, () => {
        btn.disabled = false;
        btn.innerHTML = originalHtml;
    }, true);
}

function updateBatchBar() {
    const selectAllCheckbox = document.getElementById('selectAll');
    const pageCheckboxes = document.querySelectorAll('.user-checkbox:not(:disabled)');
    
    // 确保 selectedUserIds 已初始化
    if (!state.selectedUserIds) {
        state.selectedUserIds = new Set();
    }
    
    // 同步当前页面的选中状态到 state
    pageCheckboxes.forEach(cb => {
        if (cb.checked) {
            state.selectedUserIds.add(cb.value);
        } else {
            state.selectedUserIds.delete(cb.value);
        }
    });
    
    // 计算总选中数量
    const count = state.selectedUserIds.size;
    
    // 更新全选checkbox状态
    const checkedPageCheckboxes = document.querySelectorAll('.user-checkbox:checked');
    if (pageCheckboxes.length > 0) {
        selectAllCheckbox.checked = checkedPageCheckboxes.length === pageCheckboxes.length;
    } else {
        selectAllCheckbox.checked = false;
    }
    
    document.getElementById('selectedCountDisplay').textContent = count;
    const bar = document.getElementById('batchActionBar');
    if (count > 0) {
        bar.classList.add('show');
    } else {
        bar.classList.remove('show');
    }
}

function toggleDropdown(e) {
    if (e && e.stopPropagation) e.stopPropagation();
    const trigger = e && e.currentTarget ? e.currentTarget : null;
    const dropdown = trigger ? trigger.closest('.dropdown') : null;
    if (!dropdown) return;
    const isOpen = dropdown.classList.contains('show');
    document.querySelectorAll('.dropdown').forEach(d => d.classList.remove('show'));
    if (!isOpen) dropdown.classList.add('show');
}

function getDefaultColumnOrder() {
    return COLUMN_OPTIONS.map(opt => opt.key);
}

function loadColumnOrder() {
    let stored = [];
    try {
        stored = JSON.parse(localStorage.getItem(COLUMN_ORDER_KEY) || '[]');
    } catch (e) {
        stored = [];
    }
    const defaults = getDefaultColumnOrder();
    if (!Array.isArray(stored) || stored.length === 0) return defaults;
    const filtered = stored.filter(k => COLUMN_MAP[k]);
    defaults.forEach(k => {
        if (!filtered.includes(k)) filtered.push(k);
    });
    return filtered;
}

function saveColumnOrder() {
    try {
        localStorage.setItem(COLUMN_ORDER_KEY, JSON.stringify(columnOrder));
    } catch (e) {}
}

function applyColumnOrder() {
    const table = document.querySelector('table');
    if (!table) return;
    const order = columnOrder && columnOrder.length ? columnOrder : getDefaultColumnOrder();
    const rows = table.querySelectorAll('thead tr, tbody tr');
    rows.forEach(row => {
        const cells = Array.from(row.children);
        if (cells.length < 3) return;
        const firstCell = cells[0];
        const nameCell = row.querySelector('.col-name');
        const actionsCell = row.querySelector('.col-actions') || cells[cells.length - 1];
        const frag = document.createDocumentFragment();
        if (firstCell) frag.appendChild(firstCell);
        if (nameCell && nameCell !== firstCell) frag.appendChild(nameCell);
        order.forEach(key => {
            const cell = row.querySelector(`.col-${key}`);
            if (cell && cell !== firstCell && cell !== nameCell && cell !== actionsCell) {
                frag.appendChild(cell);
            }
        });
        if (actionsCell && actionsCell !== firstCell && actionsCell !== nameCell) {
            frag.appendChild(actionsCell);
        }
        row.innerHTML = '';
        row.appendChild(frag);
    });
}

function applyColumnVisibility() {
    COLUMN_OPTIONS.forEach(opt => {
        const hidden = columnVisibility[opt.key] === false;
        document.body.classList.toggle(`hide-col-${opt.key}`, hidden);
    });
    applyColumnOrder();
}

function initColumnControls() {
    const menu = document.getElementById('columnToggleMenu');
    if (!menu || initColumnControls._ready) return;
    let stored = {};
    try {
        stored = JSON.parse(localStorage.getItem(COLUMN_STORAGE_KEY) || '{}');
    } catch (e) {
        stored = {};
    }
    columnVisibility = {};
    COLUMN_OPTIONS.forEach(opt => {
        columnVisibility[opt.key] = stored[opt.key] !== undefined ? stored[opt.key] : true;
    });
    columnOrder = loadColumnOrder();

    const renderColumnMenu = () => {
        menu.innerHTML = '';
        let draggingKey = null;
        columnOrder.forEach(key => {
            const opt = COLUMN_MAP[key];
            if (!opt) return;
            const label = document.createElement('label');
            label.className = 'dropdown-item';
            label.dataset.col = key;
            label.draggable = true;
            label.addEventListener('dragstart', (e) => {
                draggingKey = key;
                label.classList.add('dragging');
                if (e.dataTransfer) {
                    e.dataTransfer.effectAllowed = 'move';
                    e.dataTransfer.setData('text/plain', key);
                }
            });
            label.addEventListener('dragend', () => {
                draggingKey = null;
                label.classList.remove('dragging');
                menu.querySelectorAll('.drag-over').forEach(el => el.classList.remove('drag-over'));
            });
            label.addEventListener('dragover', (e) => {
                e.preventDefault();
                if (draggingKey && draggingKey !== key) {
                    label.classList.add('drag-over');
                }
            });
            label.addEventListener('dragleave', () => {
                label.classList.remove('drag-over');
            });
            label.addEventListener('drop', (e) => {
                e.preventDefault();
                label.classList.remove('drag-over');
                if (!draggingKey || draggingKey === key) return;
                const from = columnOrder.indexOf(draggingKey);
                const to = columnOrder.indexOf(key);
                if (from < 0 || to < 0) return;
                columnOrder.splice(from, 1);
                columnOrder.splice(to, 0, draggingKey);
                saveColumnOrder();
                renderColumnMenu();
                applyColumnOrder();
            });

            const input = document.createElement('input');
            input.type = 'checkbox';
            input.checked = columnVisibility[key];
            input.dataset.col = key;
            input.addEventListener('change', () => {
                columnVisibility[key] = input.checked;
                localStorage.setItem(COLUMN_STORAGE_KEY, JSON.stringify(columnVisibility));
                applyColumnVisibility();
            });
            const text = document.createElement('span');
            text.textContent = opt.label;
            const handle = document.createElement('span');
            handle.className = 'drag-handle';
            handle.textContent = '::';
            label.appendChild(input);
            label.appendChild(text);
            label.appendChild(handle);
            menu.appendChild(label);
        });

        const resetBtn = document.createElement('button');
        resetBtn.type = 'button';
        resetBtn.className = 'btn btn-outline btn-xs column-reset-btn';
        resetBtn.textContent = '恢复默认列';
        resetBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            columnVisibility = {};
            COLUMN_OPTIONS.forEach(opt => { columnVisibility[opt.key] = true; });
            columnOrder = getDefaultColumnOrder();
            localStorage.setItem(COLUMN_STORAGE_KEY, JSON.stringify(columnVisibility));
            saveColumnOrder();
            renderColumnMenu();
            applyColumnVisibility();
        });
        menu.appendChild(resetBtn);
    };

    renderColumnMenu();
    applyColumnVisibility();
    initColumnControls._ready = true;
}

function showGroupSuggestions(input) {
    const container = input.parentElement;
    if(!container) return;

    let optionsDiv = container.querySelector('.custom-select-options');
    if (!optionsDiv) {
        optionsDiv = document.createElement('div');
        optionsDiv.className = 'custom-select-options';
        container.appendChild(optionsDiv);
    }

    const groups = new Set();
    if(state.users) state.users.forEach(u => { if (u.group) groups.add(u.group); });
    
    const filterVal = input.value.trim().toLowerCase();
    let matchedGroups = Array.from(groups).sort();
    
    if (filterVal) {
        matchedGroups = matchedGroups.filter(g => g.toLowerCase().includes(filterVal));
    }

    if (matchedGroups.length === 0) {
        optionsDiv.classList.remove('show');
        optionsDiv.style.display = 'none';
        return;
    }

    optionsDiv.innerHTML = '';
    matchedGroups.forEach(g => {
        const div = document.createElement('div');
        div.className = 'custom-option';
        div.textContent = g;
        div.onclick = (e) => {
            e.stopPropagation();
            input.value = g;
            optionsDiv.classList.remove('show');
            optionsDiv.style.display = 'none';
        };
        optionsDiv.appendChild(div);
    });

    // Positioning logic
    const rect = container.getBoundingClientRect();
    optionsDiv.style.position = 'fixed';
    optionsDiv.style.left = rect.left + 'px';
    optionsDiv.style.width = rect.width + 'px';
    optionsDiv.style.top = (rect.bottom + 5) + 'px';
    optionsDiv.style.marginTop = '0';

    optionsDiv.classList.add('show');
    optionsDiv.style.display = 'block';
}

function downloadBackup() {
    const csrfMeta = document.querySelector('meta[name="csrf-token"]');
    const token = csrfMeta ? csrfMeta.getAttribute('content') : '';
    window.location.href = '?action=backup&token=' + token;
}

function triggerRestore() {
    document.getElementById('restoreFile').click();
}

function uploadBackup(input) {
    if (!input.files || input.files.length === 0) return;
    
    const file = input.files[0];
    if (!confirm('确定要恢复备份吗？这将覆盖当前所有数据！')) {
        input.value = '';
        return;
    }
    
    const formData = new FormData();
    formData.append('action', 'restore');
    formData.append('backup_file', file);
    
    const btn = document.querySelector('button[onclick="triggerRestore()"]');
    const originalText = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = '恢复中...';
    
    const csrfMeta = document.querySelector('meta[name="csrf-token"]');
    const headers = {
        'X-Requested-With': 'XMLHttpRequest'
    };
    if (csrfMeta) headers['X-CSRF-Token'] = csrfMeta.getAttribute('content');

    fetch('', {
        method: 'POST',
        headers: headers,
        body: formData
    })
    .then(r => r.json())
    .then(r => {
        if (r.success) {
            toast(r.message, true);
            setTimeout(() => location.reload(), 1000);
        } else {
            toast(r.message, false);
            btn.disabled = false;
            btn.innerHTML = originalText;
        }
    })
    .catch(e => {
        toast('恢复失败: ' + e.message, false);
        btn.disabled = false;
        btn.innerHTML = originalText;
    });
    
    input.value = '';
}

function sendTestEmail(btn) {
    const to = document.getElementById('test_email_to').value.trim();
    if (!to) return toast('请输入接收邮箱', false);
    
    // 获取当前填写的 SMTP 信息
    const host = document.getElementById('smtp_host').value.trim();
    const port = document.getElementById('smtp_port').value.trim();
    const user = document.getElementById('smtp_user').value.trim();
    const pass = document.getElementById('smtp_pass').value.trim();
    const from = document.getElementById('smtp_from').value.trim();
    const secure = document.querySelector('input[name="smtp_secure"]:checked').value;
    
    if (!host || !user || !pass) return toast('请先填写完整的 SMTP 信息', false);

    const originalText = btn.textContent;
    btn.disabled = true; btn.textContent = '发送中...';
    
    post({
        action: 'test_email',
        test_to: to,
        smtp_host: host,
        smtp_port: port,
        smtp_user: user,
        smtp_pass: pass,
        smtp_from: from,
        smtp_secure: secure
    }, (res) => {
        btn.disabled = false; btn.textContent = originalText;
    }, () => {
        btn.disabled = false; btn.textContent = originalText;
    }, true, false);
}
