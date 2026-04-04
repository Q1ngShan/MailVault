<script setup>
import { ref, reactive, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { GetMails, GetMailDetail } from '../../bindings/mailvault/mailservice'

const props = defineProps({
  account: { type: Object, required: true }
})

const folder = ref('inbox')
const mails = ref([])
const loading = ref(false)
const detailLoading = ref(false)
const selectedMail = ref(null)
const mailDetail = ref(null)
const total = ref(0)
const page = ref(1)
const pageSize = 20

async function loadMails() {
  loading.value = true
  selectedMail.value = null
  mailDetail.value = null
  try {
    const res = await GetMails(props.account.id, folder.value, page.value, pageSize)
    mails.value = res.items || []
    total.value = res.total || 0
  } catch (e) {
    ElMessage.error('加载邮件失败: ' + e)
  } finally {
    loading.value = false
  }
}

async function selectMail(mail) {
  selectedMail.value = mail
  mailDetail.value = null
  detailLoading.value = true
  try {
    const res = await GetMailDetail(props.account.id, folder.value, mail.uid)
    mailDetail.value = res.detail
  } catch (e) {
    ElMessage.error('加载邮件详情失败: ' + e)
  } finally {
    detailLoading.value = false
  }
}

function handleFolderChange() {
  page.value = 1
  loadMails()
}

function handlePageChange(p) {
  page.value = p
  loadMails()
}

onMounted(loadMails)
watch(() => props.account, () => {
  page.value = 1
  folder.value = 'inbox'
  loadMails()
})
</script>

<template>
  <div class="mail-viewer">
    <!-- Left: mail list -->
    <div class="mail-list-panel">
      <div class="mail-list-header">
        <div class="account-info">
          <el-icon><User /></el-icon>
          <span class="account-email">{{ account.email }}</span>
        </div>
        <div class="folder-tabs">
          <el-radio-group v-model="folder" size="small" @change="handleFolderChange">
            <el-radio-button value="inbox">收件箱</el-radio-button>
            <el-radio-button value="junk">垃圾邮件</el-radio-button>
          </el-radio-group>
          <el-button size="small" @click="loadMails" :loading="loading" circle>
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </div>

      <div class="mail-list" v-loading="loading">
        <div
          v-for="mail in mails"
          :key="mail.uid"
          class="mail-item"
          :class="{ active: selectedMail?.uid === mail.uid }"
          @click="selectMail(mail)"
        >
          <div class="mail-subject">{{ mail.subject || '(无主题)' }}</div>
          <div class="mail-from">{{ mail.from }}</div>
          <div class="mail-date">{{ mail.date }}</div>
        </div>
        <el-empty v-if="!loading && mails.length === 0" description="暂无邮件" :image-size="60" />
      </div>

      <div class="mail-pagination">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          @current-change="handlePageChange"
          small
        />
      </div>
    </div>

    <!-- Right: mail detail -->
    <div class="mail-detail-panel">
      <div v-if="!selectedMail" class="detail-empty">
        <el-empty description="选择邮件查看内容" />
      </div>

      <div v-else class="detail-content" v-loading="detailLoading">
        <div v-if="mailDetail" class="detail-inner">
          <div class="detail-header">
            <h2 class="detail-subject">{{ mailDetail.subject || '(无主题)' }}</h2>
            <div class="detail-meta">
              <div><span class="meta-label">发件人：</span>{{ mailDetail.from }}</div>
              <div><span class="meta-label">收件人：</span>{{ mailDetail.to }}</div>
              <div><span class="meta-label">时间：</span>{{ mailDetail.date }}</div>
            </div>
          </div>
          <el-divider />
          <div class="detail-body">
            <div
              v-if="mailDetail.body_html"
              class="html-body"
              v-html="mailDetail.body_html"
            />
            <pre v-else-if="mailDetail.body_text" class="text-body">{{ mailDetail.body_text }}</pre>
            <el-empty v-else description="邮件内容为空" :image-size="60" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mail-viewer {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.mail-list-panel {
  width: 320px;
  min-width: 260px;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
}

.mail-list-header {
  padding: 12px;
  border-bottom: 1px solid #e4e7ed;
  background: #fafafa;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.account-info {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.account-email {
  font-weight: 500;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.folder-tabs {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mail-list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.mail-item {
  padding: 12px 14px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background 0.15s;
}

.mail-item:hover { background: #f5f7fa; }
.mail-item.active { background: #ecf5ff; border-left: 3px solid #409eff; }

.mail-subject {
  font-size: 13px;
  font-weight: 500;
  color: #303133;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 4px;
}

.mail-from {
  font-size: 12px;
  color: #606266;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 2px;
}

.mail-date {
  font-size: 11px;
  color: #909399;
}

.mail-pagination {
  padding: 8px;
  border-top: 1px solid #e4e7ed;
  display: flex;
  justify-content: center;
}

.mail-detail-panel {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.detail-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.detail-content {
  height: 100%;
  overflow-y: auto;
}

.detail-inner {
  padding: 24px 28px;
}

.detail-subject {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
  line-height: 1.4;
}

.detail-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  color: #606266;
}

.meta-label {
  color: #909399;
}

.detail-body {
  margin-top: 8px;
}

.html-body {
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
}

.text-body {
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.6;
  color: #303133;
  font-family: inherit;
}
</style>
