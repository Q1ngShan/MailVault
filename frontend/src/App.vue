<script setup>
import { ref, onMounted } from 'vue'
import AccountManager from './components/AccountManager.vue'
import MailViewer from './components/MailViewer.vue'

const activeTab = ref('accounts')
const selectedAccount = ref(null)

function openMailView(account) {
  selectedAccount.value = account
  activeTab.value = 'mail'
}
</script>

<template>
  <div class="app-container">
    <el-container style="height: 100vh;">
      <el-header class="app-header">
        <div class="header-title">
          <el-icon size="20"><Message /></el-icon>
          <span>MailVault</span>
        </div>
        <div class="header-nav">
          <el-button
            :type="activeTab === 'accounts' ? 'primary' : 'default'"
            @click="activeTab = 'accounts'"
            size="small"
          >
            <el-icon><UserFilled /></el-icon>
            账号管理
          </el-button>
          <el-button
            :type="activeTab === 'mail' ? 'primary' : 'default'"
            @click="activeTab = 'mail'"
            size="small"
            :disabled="!selectedAccount"
          >
            <el-icon><Message /></el-icon>
            邮件查看
          </el-button>
        </div>
      </el-header>
      <el-main style="padding: 0; overflow: hidden;">
        <AccountManager
          v-if="activeTab === 'accounts'"
          @view-mail="openMailView"
        />
        <MailViewer
          v-if="activeTab === 'mail' && selectedAccount"
          :account="selectedAccount"
        />
        <div v-if="activeTab === 'mail' && !selectedAccount" class="empty-state">
          <el-empty description="请先选择一个账号查看邮件" />
        </div>
      </el-main>
    </el-container>
  </div>
</template>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.app-container {
  height: 100vh;
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  height: 52px !important;
  padding: 0 16px;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.header-nav {
  display: flex;
  gap: 8px;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}
</style>
