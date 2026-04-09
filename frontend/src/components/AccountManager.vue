<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  GetAccounts, CreateAccount, UpdateAccount, DeleteAccount,
  ImportAccounts, ExportAccounts, RefreshToken, RefreshAllTokens,
  GetAccountTypes, CreateAccountType, UpdateAccountType, DeleteAccountType,
  ArchiveAccount, ArchiveAllAccounts,
  CheckAllAccounts, DeleteDeadAccounts,
  GetCodexOAuth, GetCodexConfig, SaveCodexConfig, SaveCodexToken,
  GetCLIProxyConfig, SaveCLIProxyConfig, SyncCodexToken
} from '../../bindings/mailvault/mailservice'
import { Dialogs } from '@wailsio/runtime'

const emit = defineEmits(['view-mail'])

const accounts    = ref([])
const accountTypes = ref([])
const loading     = ref(false)
const total       = ref(0)

const query = reactive({
  search:      '',
  account_type: '',
  page:        1,
  page_size:   20,
  active_only: false,
  sort_by:     '',
  sort_order:  'desc'
})

const showAccountDialog = ref(false)
const showImportDialog  = ref(false)
const showTypeDialog    = ref(false)
const showCheckDialog   = ref(false)
const editingAccount    = ref(null)
const editingType       = ref(null)

const accountForm = reactive({
  email: '', password: '', codex_password: '', client_id: '',
  refresh_token: '', account_type: '', remark: '', is_active: true
})
const typeForm    = reactive({ code: '', label: '', color: '#409EFF' })
const importText  = ref('')
const refreshingAll = ref(false)

const checking      = ref(false)
const checkResult   = ref(null)
const checkStatusMap = reactive({})

const codexLoadingMap    = reactive({})
const showCodexDialog    = ref(false)
const codexResult        = ref('')
const codexEmail         = ref('')

const showCodexConfigDialog = ref(false)
const codexConfigForm = reactive({ proxy: '', oauth_client_id: '', oauth_redirect_uri: '' })

const showCLIProxyConfigDialog = ref(false)
const cliProxyConfigForm = reactive({ url: '', api_key: '' })
const syncingMap = reactive({})


// ── Helpers ──────────────────────────────────────────────────────────────────

function emailInitial(email) {
  return email ? email[0].toUpperCase() : '?'
}

function emailAvatarColor(email) {
  const palette = ['#5b6cf9','#06b6d4','#10b981','#f59e0b','#8b5cf6','#ec4899','#ef4444','#f97316']
  let h = 0
  for (const c of (email || '')) h = (h * 31 + c.charCodeAt(0)) >>> 0
  return palette[h % palette.length]
}

// ── Data loading ─────────────────────────────────────────────────────────────

async function loadAccounts() {
  loading.value = true
  try {
    const res = await GetAccounts({ ...query })
    accounts.value = res.items || []
    total.value    = res.total || 0
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
  } catch (e) { console.error(e) }
}

onMounted(() => { loadAccounts(); loadTypes() })

function getTypeInfo(code) {
  return accountTypes.value.find(t => t.code === code) || null
}

// ── Account CRUD ─────────────────────────────────────────────────────────────

function openCreate() {
  editingAccount.value = null
  Object.assign(accountForm, { email:'', password:'', codex_password:'', client_id:'', refresh_token:'', account_type:'', remark:'', is_active: true })
  showAccountDialog.value = true
}

function openEdit(row) {
  editingAccount.value = row
  Object.assign(accountForm, {
    email: row.email, password: row.password, codex_password: row.codex_password || '',
    client_id: row.client_id, refresh_token: row.refresh_token,
    account_type: row.account_type || '', remark: row.remark || '', is_active: row.is_active
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
  } catch (e) { ElMessage.error('保存失败: ' + e) }
}

async function deleteAccount(row) {
  await ElMessageBox.confirm(`确认删除账号 ${row.email}？`, '确认删除', { type: 'warning' })
  try {
    await DeleteAccount(row.id)
    ElMessage.success('删除成功')
    loadAccounts()
  } catch (e) { ElMessage.error('删除失败: ' + e) }
}

async function archiveAccount(row) {
  try {
    await ArchiveAccount(row.id)
    ElMessage.success('已归档')
    loadAccounts()
  } catch (e) { ElMessage.error('归档失败: ' + e) }
}

// ── Token refresh ─────────────────────────────────────────────────────────────

async function refreshSingle(row) {
  row._refreshing = true
  try {
    await RefreshToken(row.id)
    ElMessage.success(`${row.email} Token 刷新成功`)
    loadAccounts()
  } catch (e) {
    ElMessage.error(`刷新失败: ${e}`)
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

// ── Check ─────────────────────────────────────────────────────────────────────

async function checkAll() {
  checking.value = true
  checkResult.value = null
  for (const acc of accounts.value) checkStatusMap[acc.id] = 'checking'
  showCheckDialog.value = true
  try {
    const result = await CheckAllAccounts()
    checkResult.value = result
    for (const r of result.results || []) checkStatusMap[r.id] = r.alive ? 'alive' : 'dead'
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
    `确认删除 ${deadIds.value.length} 个失效账号？`,
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
  } catch (e) { ElMessage.error('删除失败: ' + e) }
}

function rowCheckStatus(id) { return checkStatusMap[id] || '' }

// ── Codex OAuth ───────────────────────────────────────────────────────────────

async function getCodexOAuth(row) {
  codexLoadingMap[row.id] = true
  codexEmail.value = row.email
  try {
    const result = await GetCodexOAuth(row.id)
    if (result.success) {
      codexResult.value = result.json
      showCodexDialog.value = true
    } else {
      ElMessage.error('获取失败: ' + result.error)
    }
  } catch (e) {
    ElMessage.error('获取失败: ' + e)
  } finally {
    codexLoadingMap[row.id] = false
  }
}

function copyCodexJson() {
  navigator.clipboard.writeText(codexResult.value)
    .then(() => ElMessage.success('已复制到剪贴板'))
    .catch(() => ElMessage.error('复制失败'))
}

function downloadCodexJson() {
  const blob = new Blob([codexResult.value], { type: 'application/json' })
  const url  = URL.createObjectURL(blob)
  const a    = document.createElement('a')
  a.href     = url
  a.download = codexEmail.value + '.json'
  a.click()
  URL.revokeObjectURL(url)
}

async function saveCodexToFile() {
  const path = await Dialogs.SaveFile({
    Title:    '保存 Codex Token',
    Filename: codexEmail.value + '.json',
    Filters:  [{ DisplayName: 'JSON 文件', Pattern: '*.json' }]
  })
  if (!path) return
  try {
    await SaveCodexToken(path, codexResult.value)
    ElMessage.success('已保存到 ' + path)
  } catch (e) {
    ElMessage.error('保存失败: ' + e)
  }
}

async function syncToProxy() {
  syncingMap['current'] = true
  try {
    const result = await SyncCodexToken(codexResult.value)
    if (result.success) ElMessage.success('同步成功')
    else ElMessage.error('同步失败: ' + result.error)
  } catch (e) {
    ElMessage.error('同步失败: ' + e)
  } finally {
    syncingMap['current'] = false
  }
}

// ── Codex Config ─────────────────────────────────────────────────────────────

async function openCodexConfig() {
  try {
    const cfg = await GetCodexConfig()
    Object.assign(codexConfigForm, {
      proxy:            cfg.proxy || '',
      oauth_client_id:  cfg.oauth_client_id || '',
      oauth_redirect_uri: cfg.oauth_redirect_uri || ''
    })
  } catch (e) {}
  showCodexConfigDialog.value = true
}

async function saveCodexConfig() {
  try {
    await SaveCodexConfig({ ...codexConfigForm })
    ElMessage.success('配置已保存')
    showCodexConfigDialog.value = false
  } catch (e) { ElMessage.error('保存失败: ' + e) }
}

// ── CLIProxy Config ───────────────────────────────────────────────────────────

async function openCLIProxyConfig() {
  try {
    const cfg = await GetCLIProxyConfig()
    Object.assign(cliProxyConfigForm, { url: cfg.url || '', api_key: cfg.api_key || '' })
  } catch (e) {}
  showCLIProxyConfigDialog.value = true
}

async function saveCLIProxyConfig() {
  try {
    await SaveCLIProxyConfig({ ...cliProxyConfigForm })
    ElMessage.success('配置已保存')
    showCLIProxyConfigDialog.value = false
  } catch (e) { ElMessage.error('保存失败: ' + e) }
}

// ── Import / Export ───────────────────────────────────────────────────────────

async function doImport() {
  if (!importText.value.trim()) return ElMessage.warning('请输入账号数据')
  try {
    const result = await ImportAccounts(importText.value)
    ElMessage.success(`导入完成: 成功 ${result.success}/${result.total}`)
    showImportDialog.value = false
    importText.value = ''
    loadAccounts()
  } catch (e) { ElMessage.error('导入失败: ' + e) }
}

async function doExport() {
  try {
    const text = await ExportAccounts()
    const blob = new Blob([text], { type: 'text/plain' })
    const url  = URL.createObjectURL(blob)
    const a    = document.createElement('a')
    a.href     = url
    a.download = 'accounts.txt'
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) { ElMessage.error('导出失败: ' + e) }
}

// ── Account Types ─────────────────────────────────────────────────────────────

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
  } catch (e) { ElMessage.error('保存失败: ' + e) }
}

async function deleteType(t) {
  await ElMessageBox.confirm(`确认删除分类 "${t.label}"？`, '确认删除', { type: 'warning' })
  try {
    await DeleteAccountType(t.id)
    ElMessage.success('删除成功')
    loadTypes()
  } catch (e) { ElMessage.error('删除失败: ' + e) }
}

// ── Misc ──────────────────────────────────────────────────────────────────────

function handleSearch()            { query.page = 1; loadAccounts() }
function handlePageChange(page)    { query.page = page; loadAccounts() }
function handlePageSizeChange(size){ query.page_size = size; query.page = 1; loadAccounts() }
function handleSortChange()        { query.page = 1; loadAccounts() }

function handleToolCommand(cmd) {
  if      (cmd === 'codex-config')    openCodexConfig()
  else if (cmd === 'cliproxy-config') openCLIProxyConfig()
else if (cmd === 'type-manage')     showTypeDialog.value = true
}

function refreshAge(days) {
  if (days < 0)  return { text: '未刷新', cls: 'age-none' }
  if (days === 0) return { text: '今天',   cls: 'age-ok' }
  if (days <= 7) return { text: days + '天前', cls: 'age-ok' }
  if (days <= 14) return { text: days + '天前', cls: 'age-warn' }
  return { text: days + '天前', cls: 'age-danger' }
}
</script>

<template>
  <div class="account-manager">

    <!-- ── Toolbar ── -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-input
          v-model="query.search"
          placeholder="搜索邮箱 / 备注"
          style="width: 200px"
          clearable
          @input="handleSearch"
          @clear="handleSearch"
        >
          <template #prefix><el-icon style="color:var(--text-muted)"><Search /></el-icon></template>
        </el-input>

        <el-select v-model="query.account_type" placeholder="全部分类" clearable style="width:108px" @change="handleSearch">
          <el-option v-for="t in accountTypes" :key="t.code" :label="t.label" :value="t.code" />
        </el-select>

        <div class="sort-group">
          <el-select v-model="query.sort_by" placeholder="默认排序" clearable style="width:110px" @change="handleSortChange">
            <el-option label="按 ID"     value="" />
            <el-option label="按分类"   value="account_type" />
            <el-option label="按刷新时间" value="last_refresh" />
            <el-option label="按邮箱"   value="email" />
          </el-select>
          <el-select v-model="query.sort_order" style="width:76px" @change="handleSortChange">
            <el-option label="降序" value="desc" />
            <el-option label="升序" value="asc" />
          </el-select>
        </div>

        <el-switch v-model="query.active_only" active-text="仅活跃" @change="handleSearch" />
      </div>

      <div class="toolbar-right">
        <el-button-group>
          <el-button @click="showImportDialog = true" size="small">
            <el-icon><Upload /></el-icon>导入
          </el-button>
          <el-button @click="doExport" size="small">
            <el-icon><Download /></el-icon>导出
          </el-button>
        </el-button-group>

        <el-button @click="refreshAll" :loading="refreshingAll" size="small" type="warning">
          <el-icon><Refresh /></el-icon>批量刷新
        </el-button>

        <el-button @click="checkAll" :loading="checking" size="small" type="success">
          <el-icon><CircleCheck /></el-icon>检测
        </el-button>

        <el-button
          @click="deleteDeadAccounts"
          size="small" type="danger"
          :disabled="!checkResult || deadIds.length === 0"
        >
          <el-icon><Delete /></el-icon>删除失效<span v-if="deadIds.length > 0">({{ deadIds.length }})</span>
        </el-button>

        <el-divider direction="vertical" />

        <el-dropdown trigger="click" @command="handleToolCommand">
          <el-button size="small">
            <el-icon><Setting /></el-icon>
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="codex-config">   <el-icon><Connection /></el-icon>Codex 配置</el-dropdown-item>
              <el-dropdown-item command="cliproxy-config"><el-icon><Share /></el-icon>CLIProxy 配置</el-dropdown-item>
<el-dropdown-item command="type-manage">    <el-icon><Collection /></el-icon>分类管理</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <el-button @click="openCreate" type="primary" size="small">
          <el-icon><Plus /></el-icon>添加
        </el-button>
      </div>
    </div>

    <!-- ── Account List ── -->
    <div class="account-list" v-loading="loading">
      <transition-group name="card-list" tag="div" class="card-list-inner">
        <div
          v-for="row in accounts"
          :key="row.id"
          class="account-card"
          :class="{
            'card-dead':     rowCheckStatus(row.id) === 'dead',
            'card-alive':    rowCheckStatus(row.id) === 'alive',
            'card-inactive': !row.is_active
          }"
        >
          <!-- Avatar -->
          <div
            class="card-avatar"
            :style="{ background: getTypeInfo(row.account_type)?.color || emailAvatarColor(row.email) }"
          >
            <template v-if="rowCheckStatus(row.id) === 'checking'">
              <el-icon class="spin" size="14"><Loading /></el-icon>
            </template>
            <template v-else-if="rowCheckStatus(row.id) === 'alive'">
              <el-icon size="14" style="color:#fff"><CircleCheckFilled /></el-icon>
            </template>
            <template v-else-if="rowCheckStatus(row.id) === 'dead'">
              <el-icon size="14" style="color:#fff"><CircleCloseFilled /></el-icon>
            </template>
            <template v-else>
              {{ emailInitial(row.email) }}
            </template>
          </div>

          <!-- Body -->
          <div class="card-body">
            <!-- Row 1 -->
            <div class="card-top">
              <span class="email-text" :class="{ 'dead-text': rowCheckStatus(row.id) === 'dead' }">
                {{ row.email }}
              </span>
              <div class="card-meta">
                <span
                  v-if="row.account_type"
                  class="type-chip"
                  :style="{
                    color:      getTypeInfo(row.account_type)?.color || '#6b7280',
                    background: (getTypeInfo(row.account_type)?.color || '#6b7280') + '18',
                    borderColor:(getTypeInfo(row.account_type)?.color || '#6b7280') + '40'
                  }"
                >{{ getTypeInfo(row.account_type)?.label || row.account_type }}</span>
                <span v-if="!row.is_active" class="type-chip archived">归档</span>
                <span class="age-dot" :class="refreshAge(row.days_since_refresh).cls">
                  · {{ refreshAge(row.days_since_refresh).text }}
                </span>
              </div>
            </div>

            <!-- Row 2 -->
            <div class="card-bottom">
              <span class="card-remark">{{ row.remark || '' }}</span>
              <div class="card-actions">
                <el-tooltip content="查看邮件" placement="top" :show-after="500">
                  <el-button size="small" type="primary" link @click="emit('view-mail', row)">
                    <el-icon><Message /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="刷新 Token" placement="top" :show-after="500">
                  <el-button size="small" link @click="refreshSingle(row)" :loading="row._refreshing">
                    <el-icon><Refresh /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="获取 Codex Token" placement="top" :show-after="500">
                  <el-button size="small" type="warning" link @click="getCodexOAuth(row)" :loading="codexLoadingMap[row.id]">
                    <el-icon><Key /></el-icon>
                    <span class="btn-label">Codex</span>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="编辑" placement="top" :show-after="500">
                  <el-button size="small" link @click="openEdit(row)">
                    <el-icon><Edit /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip v-if="row.is_active" content="归档" placement="top" :show-after="500">
                  <el-button size="small" link @click="archiveAccount(row)">
                    <el-icon><Box /></el-icon>
                  </el-button>
                </el-tooltip>
                <el-tooltip content="删除" placement="top" :show-after="500">
                  <el-button size="small" type="danger" link @click="deleteAccount(row)">
                    <el-icon><Delete /></el-icon>
                  </el-button>
                </el-tooltip>
              </div>
            </div>
          </div>
        </div>
      </transition-group>

      <div v-if="!loading && accounts.length === 0" class="empty-state">
        <el-icon size="44" style="color:var(--text-muted)"><Inbox /></el-icon>
        <p>暂无账号</p>
      </div>
    </div>

    <!-- ── Pagination ── -->
    <div class="pagination-bar">
      <span class="total-label">共 <strong>{{ total }}</strong> 个账号</span>
      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.page_size"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="sizes, prev, pager, next"
        @current-change="handlePageChange"
        @size-change="handlePageSizeChange"
        small
        background
      />
    </div>

    <!-- ── Check Dialog ── -->
    <el-dialog v-model="showCheckDialog" title="账号存活检测" width="540px" :close-on-click-modal="!checking">
      <div v-if="checking" class="check-progress">
        <el-icon class="spin" size="22"><Loading /></el-icon>
        <span>正在检测所有账号，请稍候...</span>
      </div>
      <div v-else-if="checkResult">
        <div class="check-summary">
          <div class="stat-item"><div class="stat-val">{{ checkResult.total }}</div><div class="stat-lbl">总计</div></div>
          <div class="stat-item ok"><div class="stat-val">{{ checkResult.alive }}</div><div class="stat-lbl">存活</div></div>
          <div class="stat-item danger"><div class="stat-val">{{ checkResult.dead }}</div><div class="stat-lbl">失效</div></div>
        </div>
        <el-table :data="checkResult.results" max-height="280" size="small" style="margin-top:14px; border-radius:8px; overflow:hidden;">
          <el-table-column prop="email" label="邮箱" show-overflow-tooltip />
          <el-table-column label="状态" width="70">
            <template #default="{ row }">
              <el-tag :type="row.alive ? 'success' : 'danger'" size="small" round>{{ row.alive ? '存活' : '失效' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="error" label="原因" show-overflow-tooltip>
            <template #default="{ row }">
              <span style="font-size:11px;color:#f56c6c">{{ row.error }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <el-button @click="showCheckDialog = false">关闭</el-button>
        <el-button type="danger" :disabled="!checkResult || deadIds.length === 0" @click="deleteDeadAccounts">
          删除失效 ({{ deadIds.length }})
        </el-button>
      </template>
    </el-dialog>

    <!-- ── Account Form Dialog ── -->
    <el-dialog v-model="showAccountDialog" :title="editingAccount ? '编辑账号' : '添加账号'" width="480px">
      <el-form :model="accountForm" label-width="90px" label-position="left">
        <el-form-item label="邮箱" required>
          <el-input v-model="accountForm.email" placeholder="user@example.com" />
        </el-form-item>
        <el-form-item label="邮箱密码">
          <el-input v-model="accountForm.password" type="password" show-password placeholder="Outlook 登录密码" />
        </el-form-item>
        <el-form-item label="Codex 密码">
          <el-input v-model="accountForm.codex_password" type="password" show-password placeholder="留空则使用邮箱验证码登录" />
        </el-form-item>
        <el-form-item label="Client ID">
          <el-input v-model="accountForm.client_id" placeholder="Azure 应用 Client ID" />
        </el-form-item>
        <el-form-item label="Refresh Token">
          <el-input v-model="accountForm.refresh_token" type="textarea" :rows="3" placeholder="OAuth Refresh Token" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="accountForm.account_type" clearable placeholder="选择分类" style="width:100%">
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

    <!-- ── Import Dialog ── -->
    <el-dialog v-model="showImportDialog" title="批量导入" width="540px">
      <p class="import-hint">每行格式：<code>邮箱----密码----ClientID----RefreshToken</code></p>
      <el-input v-model="importText" type="textarea" :rows="10"
        placeholder="user@outlook.com----password----client_id----refresh_token" />
      <template #footer>
        <el-button @click="showImportDialog = false">取消</el-button>
        <el-button type="primary" @click="doImport">导入</el-button>
      </template>
    </el-dialog>

    <!-- ── Codex Result Dialog ── -->
    <el-dialog v-model="showCodexDialog" :title="'Codex Token — ' + codexEmail" width="640px">
      <div class="codex-result">
        <pre class="codex-json">{{ codexResult }}</pre>
      </div>
      <template #footer>
        <el-button @click="showCodexDialog = false">关闭</el-button>
        <el-button @click="copyCodexJson"><el-icon><CopyDocument /></el-icon>复制</el-button>
        <el-button @click="saveCodexToFile"><el-icon><FolderOpened /></el-icon>保存到文件</el-button>
        <el-button type="success" :loading="syncingMap['current']" @click="syncToProxy">
          <el-icon><Share /></el-icon>同步到 CLIProxy
        </el-button>
        <el-button type="primary" @click="downloadCodexJson">
          <el-icon><Download /></el-icon>下载 JSON
        </el-button>
      </template>
    </el-dialog>

    <!-- ── CLIProxy Config Dialog ── -->
    <el-dialog v-model="showCLIProxyConfigDialog" title="CLIProxy 配置" width="460px">
      <el-form :model="cliProxyConfigForm" label-width="110px">
        <el-form-item label="Base URL">
          <el-input v-model="cliProxyConfigForm.url" placeholder="http://localhost:8317" />
        </el-form-item>
        <el-form-item label="Management Key">
          <el-input v-model="cliProxyConfigForm.api_key" type="password" show-password placeholder="MANAGEMENT_KEY" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCLIProxyConfigDialog = false">取消</el-button>
        <el-button type="primary" @click="saveCLIProxyConfig">保存</el-button>
      </template>
    </el-dialog>

    <!-- ── Codex Config Dialog ── -->
    <el-dialog v-model="showCodexConfigDialog" title="Codex OAuth 配置" width="460px">
      <el-form :model="codexConfigForm" label-width="100px">
        <el-form-item label="HTTP 代理">
          <el-input v-model="codexConfigForm.proxy" placeholder="http://127.0.0.1:7897 (可选)" />
        </el-form-item>
        <el-form-item label="Client ID">
          <el-input v-model="codexConfigForm.oauth_client_id" placeholder="默认: app_EMoamEEZ73f0CkXaXp7hrann" />
        </el-form-item>
        <el-form-item label="Redirect URI">
          <el-input v-model="codexConfigForm.oauth_redirect_uri" placeholder="默认: http://localhost:1455/auth/callback" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCodexConfigDialog = false">取消</el-button>
        <el-button type="primary" @click="saveCodexConfig">保存</el-button>
      </template>
    </el-dialog>

    <!-- ── Type Manager Dialog ── -->
    <el-dialog v-model="showTypeDialog" title="账号分类管理" width="500px">
      <el-table :data="accountTypes" size="small" style="border-radius:8px;overflow:hidden;">
        <el-table-column prop="label" label="名称" />
        <el-table-column prop="code"  label="代码" />
        <el-table-column label="颜色" width="70">
          <template #default="{ row }">
            <div class="color-dot" :style="{ background: row.color }" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="110">
          <template #default="{ row }">
            <el-button size="small" link @click="openEditType(row)">编辑</el-button>
            <el-button size="small" link type="danger" @click="deleteType(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div style="margin-top:12px">
        <el-button type="primary" size="small" @click="openCreateType">
          <el-icon><Plus /></el-icon>添加分类
        </el-button>
      </div>
      <el-form
        v-if="editingType !== undefined"
        :model="typeForm"
        label-width="70px"
        style="margin-top:16px;padding-top:16px;border-top:1px solid var(--border-color)"
      >
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="代码"><el-input v-model="typeForm.code" placeholder="team" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="名称"><el-input v-model="typeForm.label" placeholder="团队" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="颜色"><el-color-picker v-model="typeForm.color" /></el-form-item></el-col>
        </el-row>
        <el-button type="primary" size="small" @click="saveType">保存</el-button>
        <el-button size="small" @click="editingType = undefined">取消</el-button>
      </el-form>
    </el-dialog>

  </div>
</template>

<style scoped>
/* ── Layout ── */
.account-manager {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-page);
  overflow: hidden;
}

/* ── Toolbar ── */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 16px;
  background: rgba(242,243,247,0.9);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--border-color);
  position: sticky;
  top: 0;
  z-index: 10;
}

.toolbar-left, .toolbar-right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.sort-group {
  display: flex;
  gap: 0;
}
.sort-group .el-select:first-child :deep(.el-input__wrapper) {
  border-radius: 6px 0 0 6px;
}
.sort-group .el-select:last-child :deep(.el-input__wrapper) {
  border-radius: 0 6px 6px 0;
  border-left: none;
}

/* ── Account list ── */
.account-list {
  flex: 1;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 12px 14px;
}

.card-list-inner {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

/* ── Account card ── */
.account-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 14px;
  background: var(--bg-card);
  border-radius: var(--border-radius-md);
  border: 1px solid var(--border-color);
  cursor: default;
  transition: box-shadow 0.18s, transform 0.18s, border-color 0.18s;
  position: relative;
}
.account-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-1px);
  border-color: #d5d8e0;
}
.account-card:hover .card-actions { opacity: 1; pointer-events: auto; }

.account-card.card-dead    { opacity: 0.52; }
.account-card.card-alive   { border-color: #86efac; background: #f0fdf4; }
.account-card.card-inactive{ opacity: 0.55; }

/* ── Avatar ── */
.card-avatar {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  font-weight: 700;
  color: #fff;
  flex-shrink: 0;
  letter-spacing: -0.5px;
  box-shadow: 0 2px 6px rgba(0,0,0,0.18);
}

/* ── Card body ── */
.card-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.card-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}

.email-text {
  font-size: 13.5px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
  flex: 1;
}
.dead-text {
  text-decoration: line-through;
  color: var(--text-muted);
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-shrink: 0;
}

.type-chip {
  font-size: 11px;
  font-weight: 500;
  padding: 2px 7px;
  border-radius: 20px;
  border: 1px solid;
  white-space: nowrap;
  line-height: 1.4;
}
.type-chip.archived { color: #9ca3af; background: #f3f4f6; border-color: #e5e7eb; }

.age-dot {
  font-size: 11.5px;
  white-space: nowrap;
}
.age-dot.age-ok     { color: #10b981; }
.age-dot.age-warn   { color: #f59e0b; }
.age-dot.age-danger { color: #ef4444; }
.age-dot.age-none   { color: var(--text-muted); }

.card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.card-remark {
  font-size: 12px;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  min-width: 0;
}

.card-actions {
  display: flex;
  align-items: center;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s;
}

.btn-label { font-size: 11.5px; margin-left: 2px; }

/* ── Empty state ── */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 56px 0;
  color: var(--text-muted);
  font-size: 13px;
}

/* ── Pagination ── */
.pagination-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-shrink: 0;
  padding: 8px 16px;
  border-top: 1px solid var(--border-color);
  background: var(--bg-card);
}
.total-label {
  font-size: 12px;
  color: var(--text-secondary);
}
.total-label strong { color: var(--text-primary); }

/* ── Card list transition ── */
.card-list-enter-active,
.card-list-leave-active { transition: all 0.2s ease; }
.card-list-enter-from   { opacity: 0; transform: translateY(-6px); }
.card-list-leave-to     { opacity: 0; transform: translateY(6px); }

/* ── Check dialog ── */
.check-progress {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 24px 0;
  justify-content: center;
  font-size: 14px;
  color: var(--text-secondary);
}

.check-summary {
  display: flex;
  gap: 0;
  border: 1px solid var(--border-color);
  border-radius: 10px;
  overflow: hidden;
}
.stat-item {
  flex: 1;
  padding: 14px 0;
  text-align: center;
  border-right: 1px solid var(--border-color);
}
.stat-item:last-child { border-right: none; }
.stat-val { font-size: 24px; font-weight: 700; color: var(--text-primary); }
.stat-lbl { font-size: 12px; color: var(--text-muted); margin-top: 2px; }
.stat-item.ok     .stat-val { color: #10b981; }
.stat-item.danger .stat-val { color: #ef4444; }

/* ── Import hint ── */
.import-hint { margin-bottom: 10px; font-size: 13px; color: var(--text-secondary); }
.import-hint code {
  background: var(--bg-page);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 12px;
  border: 1px solid var(--border-color);
}

/* ── Codex result ── */
.codex-result {
  background: #13141a;
  border-radius: 10px;
  padding: 16px;
  max-height: 400px;
  overflow: auto;
  border: 1px solid #2a2d3a;
}
.codex-json {
  margin: 0;
  font-family: 'SF Mono', 'Fira Code', Menlo, Monaco, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-all;
  color: #c9d1d9;
}

/* ── Misc ── */
.color-dot { width: 18px; height: 18px; border-radius: 50%; }
.spin { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
