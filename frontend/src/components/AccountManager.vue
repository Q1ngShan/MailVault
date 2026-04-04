<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  GetAccounts, CreateAccount, UpdateAccount, DeleteAccount,
  ImportAccounts, ExportAccounts, RefreshToken, RefreshAllTokens,
  GetAccountTypes, CreateAccountType, UpdateAccountType, DeleteAccountType,
  ArchiveAccount, ArchiveAllAccounts,
  CheckAllAccounts, DeleteDeadAccounts
} from '../../bindings/mailstore/mailservice'

const emit = defineEmits(['view-mail'])

// State
const accounts = ref([])
const accountTypes = ref([])
const loading = ref(false)
const total = ref(0)

const query = reactive({
  search: '',
  account_type: '',
  page: 1,
  page_size: 20,
  active_only: false
})

// Dialogs
const showAccountDialog = ref(false)
const showImportDialog = ref(false)
const showTypeDialog = ref(false)
const showCheckDialog = ref(false)
const editingAccount = ref(null)
const editingType = ref(null)

const accountForm = reactive({
  email: '',
  password: '',
  client_id: '',
  refresh_token: '',
  account_type: '',
  remark: '',
  is_active: true
})

const typeForm = reactive({ code: '', label: '', color: '#409EFF' })
const importText = ref('')
const refreshingAll = ref(false)

// Check alive state
const checking = ref(false)
const checkResult = ref(null)          // CheckAllResult
// Map: account id -> 'alive' | 'dead' | 'checking'
const checkStatusMap = reactive({})

async function loadAccounts() {
  loading.value = true
  try {
    const res = await GetAccounts({ ...query })
    accounts.value = res.items || []
    total.value = res.total || 0
  } catch (e) {
    ElMessage.error('加载账号失败: ' + e)
  } finally {
    loading.value = false
  }
}

async function loadTypes() {
  try {
    const res = await GetAccountTypes()
    accountTypes.value = res || []
  } catch (e) {
    console.error(e)
  }
}

onMounted(() => {
  loadAccounts()
  loadTypes()
})

function getTypeLabel(code) {
  const t = accountTypes.value.find(t => t.code === code)
  return t ? t.label : code
}

function getTypeColor(code) {
  const t = accountTypes.value.find(t => t.code === code)
  return t ? t.color : '#909399'
}

function openCreate() {
  editingAccount.value = null
  Object.assign(accountForm, { email: '', password: '', client_id: '', refresh_token: '', account_type: '', remark: '', is_active: true })
  showAccountDialog.value = true
}

function openEdit(row) {
  editingAccount.value = row
  Object.assign(accountForm, {
    email: row.email,
    password: row.password,
    client_id: row.client_id,
    refresh_token: row.refresh_token,
    account_type: row.account_type || '',
    remark: row.remark || '',
    is_active: row.is_active
  })
  showAccountDialog.value = true
}

async function saveAccount() {
  if (!accountForm.email) return ElMessage.warning('邮箱不能为空')
  try {
    if (editingAccount.value) {
      await UpdateAccount(editingAccount.value.id, { ...accountForm })
      ElMessage.success('更新成功')
    } else {
      await CreateAccount({ ...accountForm })
      ElMessage.success('创建成功')
    }
    showAccountDialog.value = false
    loadAccounts()
  } catch (e) {
    ElMessage.error('保存失败: ' + e)
  }
}

async function deleteAccount(row) {
  await ElMessageBox.confirm(`确认删除账号 ${row.email}？`, '确认删除', { type: 'warning' })
  try {
    await DeleteAccount(row.id)
    ElMessage.success('删除成功')
    loadAccounts()
  } catch (e) {
    ElMessage.error('删除失败: ' + e)
  }
}

async function archiveAccount(row) {
  try {
    await ArchiveAccount(row.id)
    ElMessage.success('已归档')
    loadAccounts()
  } catch (e) {
    ElMessage.error('归档失败: ' + e)
  }
}

async function refreshSingle(row) {
  row._refreshing = true
  try {
    await RefreshToken(row.id)
    ElMessage.success(`${row.email} Token 刷新成功`)
    loadAccounts()
  } catch (e) {
    ElMessage.error(`Token 刷新失败: ${e}`)
  } finally {
    row._refreshing = false
  }
}

async function refreshAll() {
  refreshingAll.value = true
  try {
    const result = await RefreshAllTokens()
    ElMessage({
      message: `刷新完成: 成功 ${result.success}/${result.total}${result.failed > 0 ? '，失败 ' + result.failed : ''}`,
      type: result.failed > 0 ? 'warning' : 'success'
    })
    loadAccounts()
  } catch (e) {
    ElMessage.error('批量刷新失败: ' + e)
  } finally {
    refreshingAll.value = false
  }
}

// ── 存活检测 ────────────────────────────────────────────────────────────────

async function checkAll() {
  checking.value = true
  checkResult.value = null
  // Mark all as checking
  for (const acc of accounts.value) {
    checkStatusMap[acc.id] = 'checking'
  }
  showCheckDialog.value = true

  try {
    const result = await CheckAllAccounts()
    checkResult.value = result
    // Update status map
    for (const r of result.results || []) {
      checkStatusMap[r.id] = r.alive ? 'alive' : 'dead'
    }
    loadAccounts()
  } catch (e) {
    ElMessage.error('检测失败: ' + e)
  } finally {
    checking.value = false
  }
}

const deadIds = computed(() => {
  if (!checkResult.value) return []
  return (checkResult.value.results || []).filter(r => !r.alive).map(r => r.id)
})

async function deleteDeadAccounts() {
  if (deadIds.value.length === 0) return ElMessage.info('没有失效账号')
  await ElMessageBox.confirm(
    `确认删除 ${deadIds.value.length} 个失效账号？此操作不可撤销。`,
    '删除失效账号',
    { type: 'error', confirmButtonText: '确认删除', confirmButtonClass: 'el-button--danger' }
  )
  try {
    const n = await DeleteDeadAccounts(deadIds.value)
    ElMessage.success(`已删除 ${n} 个失效账号`)
    showCheckDialog.value = false
    checkResult.value = null
    Object.keys(checkStatusMap).forEach(k => delete checkStatusMap[k])
    loadAccounts()
  } catch (e) {
    ElMessage.error('删除失败: ' + e)
  }
}

function rowCheckStatus(id) {
  return checkStatusMap[id] || ''
}

// ── Import / Export ─────────────────────────────────────────────────────────

async function doImport() {
  if (!importText.value.trim()) return ElMessage.warning('请输入账号数据')
  try {
    const result = await ImportAccounts(importText.value)
    ElMessage.success(`导入完成: 成功 ${result.success}/${result.total}`)
    showImportDialog.value = false
    importText.value = ''
    loadAccounts()
  } catch (e) {
    ElMessage.error('导入失败: ' + e)
  }
}

async function doExport() {
  try {
    const text = await ExportAccounts()
    const blob = new Blob([text], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'accounts.txt'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    ElMessage.error('导出失败: ' + e)
  }
}

// ── Account types ───────────────────────────────────────────────────────────

function openCreateType() {
  editingType.value = null
  Object.assign(typeForm, { code: '', label: '', color: '#409EFF' })
  showTypeDialog.value = true
}

function openEditType(t) {
  editingType.value = t
  Object.assign(typeForm, { code: t.code, label: t.label, color: t.color })
  showTypeDialog.value = true
}

async function saveType() {
  if (!typeForm.code || !typeForm.label) return ElMessage.warning('请填写分类代码和名称')
  try {
    if (editingType.value) {
      await UpdateAccountType(editingType.value.id, { ...typeForm })
    } else {
      await CreateAccountType({ ...typeForm })
    }
    ElMessage.success('保存成功')
    showTypeDialog.value = false
    loadTypes()
  } catch (e) {
    ElMessage.error('保存失败: ' + e)
  }
}

async function deleteType(t) {
  await ElMessageBox.confirm(`确认删除分类 "${t.label}"？`, '确认删除', { type: 'warning' })
  try {
    await DeleteAccountType(t.id)
    ElMessage.success('删除成功')
    loadTypes()
  } catch (e) {
    ElMessage.error('删除失败: ' + e)
  }
}

function handleSearch() {
  query.page = 1
  loadAccounts()
}

function handlePageChange(page) {
  query.page = page
  loadAccounts()
}
</script>

<template>
  <div class="account-manager">
    <!-- Toolbar -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-input
          v-model="query.search"
          placeholder="搜索邮箱/备注"
          style="width: 220px"
          clearable
          @input="handleSearch"
          @clear="handleSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-select
          v-model="query.account_type"
          placeholder="全部分类"
          clearable
          style="width: 130px"
          @change="handleSearch"
        >
          <el-option v-for="t in accountTypes" :key="t.code" :label="t.label" :value="t.code" />
        </el-select>
        <el-switch v-model="query.active_only" active-text="仅活跃" @change="handleSearch" />
      </div>
      <div class="toolbar-right">
        <el-button @click="showTypeDialog = true" size="small">
          <el-icon><Setting /></el-icon>分类
        </el-button>
        <el-button @click="showImportDialog = true" size="small">
          <el-icon><Upload /></el-icon>导入
        </el-button>
        <el-button @click="doExport" size="small">
          <el-icon><Download /></el-icon>导出
        </el-button>
        <el-button @click="refreshAll" :loading="refreshingAll" size="small" type="warning">
          <el-icon><Refresh /></el-icon>批量刷新
        </el-button>
        <el-button @click="checkAll" :loading="checking" size="small" type="success">
          <el-icon><CircleCheck /></el-icon>一键检测
        </el-button>
        <el-button
          @click="deleteDeadAccounts"
          size="small"
          type="danger"
          :disabled="!checkResult || deadIds.length === 0"
        >
          <el-icon><Delete /></el-icon>删除失效 <span v-if="deadIds.length > 0">({{ deadIds.length }})</span>
        </el-button>
        <el-button @click="openCreate" type="primary" size="small">
          <el-icon><Plus /></el-icon>添加
        </el-button>
      </div>
    </div>

    <!-- Table -->
    <el-table :data="accounts" v-loading="loading" style="width: 100%; flex: 1;" height="100%">
      <el-table-column prop="email" label="邮箱" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          <div class="email-cell">
            <el-icon v-if="rowCheckStatus(row.id) === 'checking'" class="spin text-muted"><Loading /></el-icon>
            <el-icon v-else-if="rowCheckStatus(row.id) === 'alive'" class="text-success"><CircleCheckFilled /></el-icon>
            <el-icon v-else-if="rowCheckStatus(row.id) === 'dead'" class="text-danger"><CircleCloseFilled /></el-icon>
            <span :class="{ 'text-danger': rowCheckStatus(row.id) === 'dead' }">{{ row.email }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="分类" width="90">
        <template #default="{ row }">
          <el-tag v-if="row.account_type" :color="getTypeColor(row.account_type)" effect="dark" size="small">
            {{ getTypeLabel(row.account_type) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="remark" label="备注" min-width="100" show-overflow-tooltip />
      <el-table-column label="Token刷新" width="110">
        <template #default="{ row }">
          <span v-if="row.last_refresh_time" :class="{ 'text-warning': row.days_since_refresh > 7, 'text-danger': row.days_since_refresh > 14 }">
            {{ row.days_since_refresh >= 0 ? row.days_since_refresh + '天前' : '-' }}
          </span>
          <span v-else class="text-muted">未刷新</span>
        </template>
      </el-table-column>
      <el-table-column label="存活" width="70">
        <template #default="{ row }">
          <template v-if="rowCheckStatus(row.id)">
            <el-tag v-if="rowCheckStatus(row.id) === 'alive'" type="success" size="small">存活</el-tag>
            <el-tag v-else-if="rowCheckStatus(row.id) === 'dead'" type="danger" size="small">失效</el-tag>
            <el-icon v-else class="spin text-muted"><Loading /></el-icon>
          </template>
          <span v-else class="text-muted">-</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="250" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" link @click="emit('view-mail', row)">
            <el-icon><Message /></el-icon>邮件
          </el-button>
          <el-button size="small" link @click="refreshSingle(row)" :loading="row._refreshing">
            <el-icon><Refresh /></el-icon>刷新
          </el-button>
          <el-button size="small" link @click="openEdit(row)">
            <el-icon><Edit /></el-icon>编辑
          </el-button>
          <el-button size="small" link @click="archiveAccount(row)" v-if="row.is_active">
            <el-icon><Box /></el-icon>归档
          </el-button>
          <el-button size="small" type="danger" link @click="deleteAccount(row)">
            <el-icon><Delete /></el-icon>删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Pagination -->
    <div class="pagination">
      <el-pagination
        v-model:current-page="query.page"
        :page-size="query.page_size"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="handlePageChange"
        small
      />
    </div>

    <!-- Check Result Dialog -->
    <el-dialog v-model="showCheckDialog" title="存活检测结果" width="560px" :close-on-click-modal="!checking">
      <div v-if="checking" class="check-progress">
        <el-icon class="spin" size="24"><Loading /></el-icon>
        <span>正在检测所有账号，请稍候…</span>
      </div>
      <div v-else-if="checkResult">
        <div class="check-summary">
          <el-statistic title="总计" :value="checkResult.total" />
          <el-statistic title="存活" :value="checkResult.alive" class="stat-success" />
          <el-statistic title="失效" :value="checkResult.dead" class="stat-danger" />
        </div>
        <el-table :data="checkResult.results" max-height="320" size="small" style="margin-top: 12px;">
          <el-table-column prop="email" label="邮箱" show-overflow-tooltip />
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="row.alive ? 'success' : 'danger'" size="small">
                {{ row.alive ? '存活' : '失效' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="error" label="失败原因" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="text-danger" style="font-size: 11px;">{{ row.error }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <el-button @click="showCheckDialog = false">关闭</el-button>
        <el-button
          type="danger"
          :disabled="!checkResult || deadIds.length === 0"
          @click="deleteDeadAccounts"
        >
          <el-icon><Delete /></el-icon>
          删除全部失效账号 ({{ deadIds.length }})
        </el-button>
      </template>
    </el-dialog>

    <!-- Account Dialog -->
    <el-dialog v-model="showAccountDialog" :title="editingAccount ? '编辑账号' : '添加账号'" width="480px">
      <el-form :model="accountForm" label-width="90px">
        <el-form-item label="邮箱" required>
          <el-input v-model="accountForm.email" placeholder="user@outlook.com" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="accountForm.password" type="password" show-password placeholder="账号密码" />
        </el-form-item>
        <el-form-item label="Client ID">
          <el-input v-model="accountForm.client_id" placeholder="Azure应用Client ID" />
        </el-form-item>
        <el-form-item label="Refresh Token">
          <el-input v-model="accountForm.refresh_token" type="textarea" :rows="3" placeholder="OAuth Refresh Token" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="accountForm.account_type" clearable placeholder="选择分类" style="width: 100%">
            <el-option v-for="t in accountTypes" :key="t.code" :label="t.label" :value="t.code" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="accountForm.remark" placeholder="备注信息" />
        </el-form-item>
        <el-form-item label="状态" v-if="editingAccount">
          <el-switch v-model="accountForm.is_active" active-text="活跃" inactive-text="归档" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAccountDialog = false">取消</el-button>
        <el-button type="primary" @click="saveAccount">保存</el-button>
      </template>
    </el-dialog>

    <!-- Import Dialog -->
    <el-dialog v-model="showImportDialog" title="批量导入账号" width="540px">
      <p class="import-hint">每行格式: <code>邮箱----密码----ClientID----RefreshToken</code></p>
      <el-input v-model="importText" type="textarea" :rows="10"
        placeholder="user@outlook.com----password----client_id----refresh_token" />
      <template #footer>
        <el-button @click="showImportDialog = false">取消</el-button>
        <el-button type="primary" @click="doImport">导入</el-button>
      </template>
    </el-dialog>

    <!-- Type Dialog -->
    <el-dialog v-model="showTypeDialog" title="账号分类管理" width="500px">
      <el-table :data="accountTypes" size="small">
        <el-table-column prop="label" label="名称" />
        <el-table-column prop="code" label="代码" />
        <el-table-column label="颜色" width="80">
          <template #default="{ row }">
            <div class="color-dot" :style="{ background: row.color }"></div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" link @click="openEditType(row)">编辑</el-button>
            <el-button size="small" link type="danger" @click="deleteType(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div style="margin-top: 12px;">
        <el-button type="primary" size="small" @click="openCreateType">
          <el-icon><Plus /></el-icon>添加分类
        </el-button>
      </div>
      <el-form v-if="editingType !== undefined" :model="typeForm" label-width="70px"
        style="margin-top: 16px; padding-top: 16px; border-top: 1px solid #eee;">
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="代码"><el-input v-model="typeForm.code" placeholder="team" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="名称"><el-input v-model="typeForm.label" placeholder="团队" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="颜色"><el-color-picker v-model="typeForm.color" /></el-form-item></el-col>
        </el-row>
        <el-button type="primary" size="small" @click="saveType">保存分类</el-button>
        <el-button size="small" @click="editingType = undefined">取消</el-button>
      </el-form>
    </el-dialog>
  </div>
</template>

<style scoped>
.account-manager {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px;
  gap: 12px;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.toolbar-left, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
}

.email-cell {
  display: flex;
  align-items: center;
  gap: 5px;
}

.check-progress {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 0;
  justify-content: center;
  font-size: 14px;
  color: #606266;
}

.check-summary {
  display: flex;
  gap: 32px;
  justify-content: center;
  padding: 8px 0;
}

.stat-success :deep(.el-statistic__content) { color: #67c23a; }
.stat-danger  :deep(.el-statistic__content) { color: #f56c6c; }

.import-hint { margin-bottom: 8px; font-size: 13px; color: #909399; }
.import-hint code { background: #f5f7fa; padding: 2px 6px; border-radius: 3px; font-size: 12px; }
.color-dot { width: 20px; height: 20px; border-radius: 50%; }

.text-success { color: #67c23a; }
.text-warning { color: #e6a23c; }
.text-danger  { color: #f56c6c; }
.text-muted   { color: #c0c4cc; }

.spin { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
