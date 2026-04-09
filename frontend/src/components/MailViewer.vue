<script setup>
import { ref, reactive, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { GetMails, GetMailDetail } from '../../bindings/mailvault/mailservice'

const props = defineProps({
  account: { type: Object, required: true }
})

const folder       = ref('inbox')
const mails        = ref([])
const loading      = ref(false)
const detailLoading = ref(false)
const selectedMail = ref(null)
const mailDetail   = ref(null)
const total        = ref(0)
const page         = ref(1)
const pageSize     = 20

async function loadMails() {
  loading.value = true
  selectedMail.value = null
  mailDetail.value   = null
  try {
    const res   = await GetMails(props.account.id, folder.value, page.value, pageSize)
    mails.value = res.items || []
    total.value = res.total || 0
  } catch (e) {
    ElMessage.error('加载邮件失败: ' + e)
  } finally {
    loading.value = false
  }
}

async function selectMail(mail) {
  selectedMail.value  = mail
  mailDetail.value    = null
  detailLoading.value = true
  try {
    const res      = await GetMailDetail(props.account.id, folder.value, mail.uid)
    mailDetail.value = res.detail
  } catch (e) {
    ElMessage.error('加载邮件详情失败: ' + e)
  } finally {
    detailLoading.value = false
  }
}

function handleFolderChange() { page.value = 1; loadMails() }
function handlePageChange(p)  { page.value = p; loadMails() }

function formatDate(dateStr) {
  if (!dateStr) return ''
  try {
    const d = new Date(dateStr)
    const now = new Date()
    const diff = now - d
    if (diff < 86400000 && d.getDate() === now.getDate()) {
      return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
    }
    if (diff < 7 * 86400000) {
      return ['周日','周一','周二','周三','周四','周五','周六'][d.getDay()]
    }
    return (d.getMonth() + 1) + '/' + d.getDate()
  } catch { return dateStr }
}

onMounted(loadMails)
watch(() => props.account, () => { page.value = 1; folder.value = 'inbox'; loadMails() })
</script>

<template>
  <div class="mail-viewer">

    <!-- ── Left: mail list ── -->
    <div class="mail-list-panel">

      <!-- Panel header -->
      <div class="panel-header">
        <div class="account-chip">
          <div class="account-avatar">{{ account.email[0].toUpperCase() }}</div>
          <span class="account-email" :title="account.email">{{ account.email }}</span>
        </div>
        <div class="panel-toolbar">
          <el-radio-group v-model="folder" size="small" @change="handleFolderChange">
            <el-radio-button value="inbox">收件箱</el-radio-button>
            <el-radio-button value="junk">垃圾邮件</el-radio-button>
          </el-radio-group>
          <el-button size="small" :loading="loading" circle @click="loadMails" title="刷新">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- Mail items -->
      <div class="mail-list" v-loading="loading">
        <div
          v-for="mail in mails"
          :key="mail.uid"
          class="mail-item"
          :class="{ active: selectedMail?.uid === mail.uid }"
          @click="selectMail(mail)"
        >
          <div class="mail-item-header">
            <span class="mail-subject">{{ mail.subject || '(无主题)' }}</span>
            <span class="mail-date">{{ formatDate(mail.date) }}</span>
          </div>
          <div class="mail-from">{{ mail.from }}</div>
        </div>

        <el-empty
          v-if="!loading && mails.length === 0"
          description="暂无邮件"
          :image-size="56"
          style="padding: 40px 0"
        />
      </div>

      <!-- Pagination -->
      <div class="list-pagination">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="prev, pager, next"
          @current-change="handlePageChange"
          small
          background
        />
      </div>
    </div>

    <!-- ── Right: mail detail ── -->
    <div class="mail-detail-panel">

      <!-- Empty state -->
      <div v-if="!selectedMail" class="detail-empty">
        <el-icon size="48" style="color: var(--text-muted)"><Message /></el-icon>
        <p>选择邮件查看内容</p>
      </div>

      <!-- Detail content -->
      <div v-else class="detail-scroll" v-loading="detailLoading">
        <div v-if="mailDetail" class="detail-inner">

          <!-- Subject -->
          <h2 class="detail-subject">{{ mailDetail.subject || '(无主题)' }}</h2>

          <!-- Meta chips -->
          <div class="detail-meta">
            <div class="meta-chip">
              <span class="meta-key">发件人</span>
              <span class="meta-val">{{ mailDetail.from }}</span>
            </div>
            <div class="meta-chip">
              <span class="meta-key">收件人</span>
              <span class="meta-val">{{ mailDetail.to }}</span>
            </div>
            <div class="meta-chip">
              <span class="meta-key">时间</span>
              <span class="meta-val">{{ mailDetail.date }}</span>
            </div>
          </div>

          <div class="detail-divider" />

          <!-- Body -->
          <div class="detail-body">
            <div v-if="mailDetail.body_html" class="html-body" v-html="mailDetail.body_html" />
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
  background: var(--bg-page);
}

/* ── Left panel ── */
.mail-list-panel {
  width: 300px;
  min-width: 240px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-right: 1px solid var(--border-color);
  box-shadow: 2px 0 8px rgba(0,0,0,0.04);
}

.panel-header {
  padding: 12px 14px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: #fafbfc;
}

.account-chip {
  display: flex;
  align-items: center;
  gap: 8px;
}

.account-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-primary);
  color: #fff;
  font-size: 12px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.account-email {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* ── Mail list ── */
.mail-list {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.mail-item {
  padding: 11px 14px;
  border-bottom: 1px solid #f3f4f6;
  cursor: pointer;
  transition: background 0.12s;
}
.mail-item:hover  { background: var(--bg-hover); }
.mail-item.active {
  background: var(--color-primary-light);
  border-left: 3px solid var(--color-primary);
  padding-left: 11px;
}

.mail-item-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 3px;
}

.mail-subject {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}
.mail-item.active .mail-subject { color: var(--color-primary); }

.mail-date {
  font-size: 11px;
  color: var(--text-muted);
  white-space: nowrap;
  flex-shrink: 0;
}

.mail-from {
  font-size: 12px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ── Pagination ── */
.list-pagination {
  padding: 8px 10px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: center;
  background: #fafbfc;
}

/* ── Right panel ── */
.mail-detail-panel {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: var(--bg-page);
}

.detail-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-muted);
  font-size: 13px;
}

.detail-scroll {
  height: 100%;
  overflow-y: auto;
}

.detail-inner {
  max-width: 740px;
  margin: 0 auto;
  padding: 32px 28px 48px;
}

.detail-subject {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.4;
  letter-spacing: -0.02em;
  margin-bottom: 16px;
}

/* ── Meta chips ── */
.detail-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 20px;
}

.meta-chip {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 13px;
}

.meta-key {
  font-size: 11.5px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  min-width: 48px;
}

.meta-val {
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-divider {
  height: 1px;
  background: var(--border-color);
  margin-bottom: 24px;
}

/* ── Mail body ── */
.detail-body { min-height: 120px; }

.html-body {
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-primary);
  word-break: break-word;
}
.html-body :deep(a) { color: var(--color-primary); }
.html-body :deep(img) { max-width: 100%; height: auto; border-radius: 6px; }

.text-body {
  font-size: 13.5px;
  line-height: 1.75;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--text-primary);
  font-family: inherit;
}
</style>
