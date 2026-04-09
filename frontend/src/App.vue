<script setup>
import { ref } from 'vue'
import AccountManager from './components/AccountManager.vue'
import MailViewer from './components/MailViewer.vue'
import AuthFilesPanel from './components/AuthFilesPanel.vue'

const activeTab = ref('accounts')
const selectedAccount = ref(null)

function openMailView(account) {
  selectedAccount.value = account
  activeTab.value = 'mail'
}
</script>

<template>
  <div class="app-layout">
    <!-- ── Sidebar ── -->
    <aside class="sidebar">
      <div class="sidebar-logo">
        <span class="logo-mark">MV</span>
      </div>

      <nav class="sidebar-nav">
        <button
          class="nav-item"
          :class="{ active: activeTab === 'accounts' }"
          title="账号管理"
          @click="activeTab = 'accounts'"
        >
          <el-icon size="20"><UserFilled /></el-icon>
        </button>
        <button
          class="nav-item"
          :class="{ active: activeTab === 'mail', 'nav-disabled': !selectedAccount }"
          title="邮件查看"
          @click="selectedAccount && (activeTab = 'mail')"
        >
          <el-icon size="20"><Message /></el-icon>
          <span v-if="selectedAccount" class="nav-badge" />
        </button>
        <button
          class="nav-item"
          :class="{ active: activeTab === 'authfiles' }"
          title="认证文件"
          @click="activeTab = 'authfiles'"
        >
          <el-icon size="20"><Files /></el-icon>
        </button>
      </nav>

      <div class="sidebar-footer">
        <div v-if="selectedAccount && activeTab === 'mail'" class="sidebar-account-hint" :title="selectedAccount.email">
          {{ selectedAccount.email[0].toUpperCase() }}
        </div>
      </div>
    </aside>

    <!-- ── Main Content ── -->
    <main class="app-main">
      <AccountManager
        v-show="activeTab === 'accounts'"
        @view-mail="openMailView"
      />
      <MailViewer
        v-if="selectedAccount"
        v-show="activeTab === 'mail'"
        :account="selectedAccount"
      />
      <div v-if="activeTab === 'mail' && !selectedAccount" class="main-empty">
        <el-empty description="请先在账号管理中选择账号查看邮件" />
      </div>
      <AuthFilesPanel v-show="activeTab === 'authfiles'" />
    </main>
  </div>
</template>

<style>
/* ── Design Tokens ── */
:root {
  --color-primary:       #5b6cf9;
  --color-primary-hover: #4f5ee0;
  --color-primary-light: rgba(91,108,249,0.1);

  --bg-sidebar:  #16181d;
  --bg-page:     #f2f3f7;
  --bg-card:     #ffffff;
  --bg-hover:    #f5f6fa;

  --text-primary:   #1a1d2e;
  --text-secondary: #6b7280;
  --text-muted:     #9ca3af;

  --border-color:       #e8eaee;
  --border-radius-sm:   8px;
  --border-radius-md:   10px;
  --border-radius-lg:   14px;

  --shadow-sm: 0 1px 4px rgba(0,0,0,0.07);
  --shadow-md: 0 4px 16px rgba(0,0,0,0.10);
  --shadow-lg: 0 12px 40px rgba(0,0,0,0.16);
}

/* ── Reset ── */
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
html, body, #app { height: 100%; }
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Inter', 'Segoe UI', sans-serif;
  background: var(--bg-page);
  color: var(--text-primary);
  -webkit-font-smoothing: antialiased;
}

/* ── Element Plus overrides ── */
.el-button--primary {
  --el-button-bg-color:           var(--color-primary);
  --el-button-border-color:       var(--color-primary);
  --el-button-hover-bg-color:     var(--color-primary-hover);
  --el-button-hover-border-color: var(--color-primary-hover);
  --el-button-active-bg-color:    #4253c5;
}
.el-button--success {
  --el-button-bg-color:           #10b981;
  --el-button-border-color:       #10b981;
  --el-button-hover-bg-color:     #0ea570;
  --el-button-hover-border-color: #0ea570;
}
.el-switch.is-checked .el-switch__core {
  background-color: var(--color-primary) !important;
  border-color:     var(--color-primary) !important;
}
.el-pagination.is-background .el-pager li.is-active {
  background-color: var(--color-primary) !important;
}

/* ── Global Dialog polish ── */
.el-dialog {
  border-radius: var(--border-radius-lg) !important;
  overflow: hidden;
  box-shadow: var(--shadow-lg) !important;
}
.el-dialog__header {
  padding: 20px 24px 16px !important;
  margin-right: 0 !important;
  border-bottom: 1px solid var(--border-color) !important;
}
.el-dialog__title {
  font-size: 15px !important;
  font-weight: 600 !important;
  color: var(--text-primary) !important;
  letter-spacing: -0.01em;
}
.el-dialog__body   { padding: 20px 24px !important; }
.el-dialog__footer {
  padding: 14px 24px !important;
  border-top: 1px solid var(--border-color) !important;
  background: #fafbfc;
}
.el-dialog__headerbtn { top: 20px !important; right: 20px !important; }

/* ── Form label polish ── */
.el-form-item__label {
  font-size: 13px !important;
  font-weight: 500 !important;
  color: var(--text-secondary) !important;
}

/* ── Dropdown menu ── */
.el-dropdown-menu__item { font-size: 13px; }

/* ── Scrollbar ── */
::-webkit-scrollbar { width: 5px; height: 5px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: #d1d5db; border-radius: 10px; }
::-webkit-scrollbar-thumb:hover { background: #9ca3af; }
</style>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: var(--bg-page);
}

/* ── Sidebar ── */
.sidebar {
  width: 64px;
  flex-shrink: 0;
  background: var(--bg-sidebar);
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: 2px 0 12px rgba(0,0,0,0.25);
  z-index: 100;
}

.sidebar-logo {
  width: 64px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid rgba(255,255,255,0.05);
}

.logo-mark {
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0.5px;
  color: var(--color-primary);
  background: rgba(91,108,249,0.18);
  padding: 6px 7px;
  border-radius: var(--border-radius-sm);
}

.sidebar-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 0;
  gap: 4px;
}

.nav-item {
  position: relative;
  width: 44px;
  height: 44px;
  border-radius: var(--border-radius-sm);
  border: none;
  background: transparent;
  color: #545870;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
  outline: none;
}
.nav-item:hover         { background: rgba(255,255,255,0.07); color: #b0b5c8; }
.nav-item.active        { background: rgba(91,108,249,0.2); color: var(--color-primary); }
.nav-item.active::before {
  content: '';
  position: absolute;
  left: -10px;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  background: var(--color-primary);
  border-radius: 0 3px 3px 0;
}
.nav-item.nav-disabled  { opacity: 0.3; cursor: not-allowed; }

.nav-badge {
  position: absolute;
  top: 9px;
  right: 9px;
  width: 6px;
  height: 6px;
  background: #10b981;
  border-radius: 50%;
  border: 1.5px solid var(--bg-sidebar);
}

.sidebar-footer {
  padding-bottom: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.sidebar-account-hint {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--color-primary);
  color: #fff;
  font-size: 13px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: default;
}

/* ── Main ── */
.app-main {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.main-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
