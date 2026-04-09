<script setup>
import { ref, computed, onMounted } from 'vue'
import { GetCLIProxyAuthFiles, GetCLIProxyUsage, GetCLIProxyStatus } from '../../bindings/mailvault/mailservice'

// ── Connection Status ─────────────────────────────────────────────────────────
const status        = ref(null)   // CLIProxyStatus | null
const statusLoading = ref(false)

async function checkStatus() {
  statusLoading.value = true
  try {
    status.value = await GetCLIProxyStatus()
  } catch (e) {
    status.value = { connected: false, stats_enabled: false, error: String(e) }
  } finally {
    statusLoading.value = false
  }
}

// ── Auth Files ────────────────────────────────────────────────────────────────
const files        = ref([])
const filesLoading = ref(false)
const filesError   = ref('')

async function loadFiles() {
  filesLoading.value = true
  filesError.value   = ''
  try {
    const res = await GetCLIProxyAuthFiles()
    files.value = res.files || []
  } catch (e) {
    filesError.value = String(e)
  } finally {
    filesLoading.value = false
  }
}

const fileStats = computed(() => {
  const f = files.value
  return {
    total:      f.length,
    active:     f.filter(x => x.status === 'active').length,
    pending:    f.filter(x => x.status === 'pending').length,
    refreshing: f.filter(x => x.status === 'refreshing').length,
    error:      f.filter(x => x.status === 'error').length,
    disabled:   f.filter(x => x.disabled || x.status === 'disabled').length,
  }
})

// ── Usage Stats ───────────────────────────────────────────────────────────────
const usage        = ref(null)
const usageLoading = ref(false)
const usageError   = ref('')

async function loadUsage() {
  usageLoading.value = true
  usageError.value   = ''
  try {
    usage.value = await GetCLIProxyUsage()
  } catch (e) {
    usageError.value = String(e)
  } finally {
    usageLoading.value = false
  }
}

// Top models sorted by requests
const topModels = computed(() => {
  if (!usage.value?.apis) return []
  const list = []
  for (const [api, apiStats] of Object.entries(usage.value.apis)) {
    if (!apiStats.models) continue
    for (const [model, ms] of Object.entries(apiStats.models)) {
      list.push({ api, model, requests: ms.total_requests, tokens: ms.total_tokens })
    }
  }
  return list.sort((a, b) => b.requests - a.requests).slice(0, 8)
})

// Last 7 days chart data
const chartDays = computed(() => {
  if (!usage.value?.requests_by_day) return []
  const entries = Object.entries(usage.value.requests_by_day)
    .sort((a, b) => a[0].localeCompare(b[0]))
    .slice(-7)
  if (entries.length === 0) return []
  const max = Math.max(...entries.map(([, v]) => v), 1)
  return entries.map(([day, val]) => ({
    label: day.slice(5),   // MM-DD
    val,
    pct: Math.round((val / max) * 100)
  }))
})

// ── Combined reload ────────────────────────────────────────────────────────────
function loadAll() { checkStatus(); loadFiles(); loadUsage() }
onMounted(loadAll)

// ── Helpers ───────────────────────────────────────────────────────────────────
function tagType(status) {
  const m = { active: 'success', error: 'danger', pending: 'warning', refreshing: 'primary', disabled: 'info' }
  return m[status] || 'info'
}

function fmtSize(bytes) {
  if (!bytes) return '—'
  if (bytes < 1024) return bytes + ' B'
  return (bytes / 1024).toFixed(1) + ' KB'
}

function fmtNum(n) {
  if (!n) return '0'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000)     return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

function successRate(u) {
  if (!u || !u.total_requests) return '—'
  return ((u.success_count / u.total_requests) * 100).toFixed(1) + '%'
}
</script>

<template>
  <div class="af-panel">

    <!-- ── Header ── -->
    <div class="af-header">
      <div class="af-title">
        <el-icon size="15"><Files /></el-icon>
        <span>CLIProxy 面板</span>
      </div>
      <el-button size="small" :loading="filesLoading || usageLoading" link style="color:var(--text-muted)" @click="loadAll">
        <el-icon><Refresh /></el-icon>
      </el-button>
    </div>

    <!-- ── Connection Status Bar ── -->
    <div
      v-if="status || statusLoading"
      class="conn-bar"
      :class="{
        'conn-loading':  statusLoading,
        'conn-ok':       !statusLoading && status?.connected,
        'conn-err':      !statusLoading && status && !status.connected
      }"
    >
      <template v-if="statusLoading">
        <el-icon class="spin" size="12"><Loading /></el-icon>
        <span>验证连接中...</span>
      </template>
      <template v-else-if="status?.connected">
        <el-icon size="12"><CircleCheckFilled /></el-icon>
        <span>已连接</span>
        <span v-if="!status.stats_enabled" class="conn-warn">· 使用统计未开启</span>
      </template>
      <template v-else>
        <el-icon size="12"><CircleCloseFilled /></el-icon>
        <span>{{ status?.error || '连接失败' }}</span>
        <el-button size="small" link style="margin-left:auto;font-size:11px" @click="loadAll">重试</el-button>
      </template>
    </div>

    <div class="af-body">

      <!-- ══ Usage Statistics Section ══ -->
      <div class="section-label">使用统计</div>

      <div v-if="usageLoading" class="af-center" style="padding:24px 0">
        <el-icon class="spin" size="18"><Loading /></el-icon>
        <span>加载中...</span>
      </div>
      <div v-else-if="usageError" class="af-center af-error" style="padding:16px 0">
        <el-icon size="16"><CircleCloseFilled /></el-icon>
        <span>{{ usageError }}</span>
      </div>
      <template v-else-if="usage">
        <!-- Big stats row -->
        <div class="usage-stats-row">
          <div class="usage-stat">
            <div class="usage-stat-val">{{ fmtNum(usage.total_requests) }}</div>
            <div class="usage-stat-lbl">总请求</div>
          </div>
          <div class="usage-stat ok">
            <div class="usage-stat-val">{{ fmtNum(usage.success_count) }}</div>
            <div class="usage-stat-lbl">成功</div>
          </div>
          <div class="usage-stat danger">
            <div class="usage-stat-val">{{ fmtNum(usage.failure_count) }}</div>
            <div class="usage-stat-lbl">失败</div>
          </div>
          <div class="usage-stat info">
            <div class="usage-stat-val">{{ fmtNum(usage.total_tokens) }}</div>
            <div class="usage-stat-lbl">总 Token</div>
          </div>
          <div class="usage-stat">
            <div class="usage-stat-val">{{ successRate(usage) }}</div>
            <div class="usage-stat-lbl">成功率</div>
          </div>
        </div>

        <!-- 7-day bar chart -->
        <div v-if="chartDays.length > 0" class="chart-wrap">
          <div class="chart-title">近 7 天请求量</div>
          <div class="chart-bars">
            <div v-for="d in chartDays" :key="d.label" class="chart-col">
              <div class="chart-bar-wrap">
                <div class="chart-bar" :style="{ height: d.pct + '%' }" />
              </div>
              <div class="chart-label">{{ d.label }}</div>
              <div class="chart-val">{{ fmtNum(d.val) }}</div>
            </div>
          </div>
        </div>

        <!-- Top models -->
        <div v-if="topModels.length > 0" class="models-wrap">
          <div class="chart-title" style="margin-bottom:6px">模型用量 Top{{ topModels.length }}</div>
          <div class="model-list">
            <div v-for="m in topModels" :key="m.api + m.model" class="model-row">
              <div class="model-name" :title="m.api + ' › ' + m.model">{{ m.model }}</div>
              <div class="model-bar-bg">
                <div
                  class="model-bar-fill"
                  :style="{ width: ((m.requests / topModels[0].requests) * 100) + '%' }"
                />
              </div>
              <div class="model-nums">
                <span class="model-req">{{ fmtNum(m.requests) }}</span>
                <span class="model-tok">{{ fmtNum(m.tokens) }} tok</span>
              </div>
            </div>
          </div>
        </div>
      </template>
      <div v-else class="af-center af-muted" style="padding:20px 0">
        <el-icon size="24"><DataAnalysis /></el-icon>
        <span v-if="status && !status.stats_enabled">CLIProxy 使用统计未开启</span>
        <span v-else>暂无统计数据</span>
      </div>

      <!-- ══ Auth Files Section ══ -->
      <div class="section-label" style="margin-top:12px">
        认证文件
        <span v-if="files.length > 0" class="section-count">{{ files.length }}</span>
      </div>

      <!-- File stats row -->
      <div v-if="!filesLoading && !filesError && files.length > 0" class="af-stats">
        <div class="af-stat">
          <div class="af-stat-val">{{ fileStats.active }}</div>
          <div class="af-stat-lbl">活跃</div>
        </div>
        <div class="af-stat warn">
          <div class="af-stat-val">{{ fileStats.pending }}</div>
          <div class="af-stat-lbl">待处理</div>
        </div>
        <div class="af-stat info">
          <div class="af-stat-val">{{ fileStats.refreshing }}</div>
          <div class="af-stat-lbl">刷新中</div>
        </div>
        <div class="af-stat danger">
          <div class="af-stat-val">{{ fileStats.error }}</div>
          <div class="af-stat-lbl">错误</div>
        </div>
        <div class="af-stat muted">
          <div class="af-stat-val">{{ fileStats.disabled }}</div>
          <div class="af-stat-lbl">禁用</div>
        </div>
      </div>

      <div v-if="filesLoading" class="af-center" style="padding:20px 0">
        <el-icon class="spin" size="18"><Loading /></el-icon>
        <span>加载中...</span>
      </div>
      <div v-else-if="filesError" class="af-center af-error" style="padding:16px 0">
        <el-icon size="16"><CircleCloseFilled /></el-icon>
        <span>{{ filesError }}</span>
      </div>
      <div v-else-if="files.length === 0" class="af-center af-muted" style="padding:20px 0">
        <el-icon size="28"><Files /></el-icon>
        <span>暂无认证文件</span>
      </div>
      <div v-else class="af-list">
        <div
          v-for="f in files"
          :key="f.id || f.name"
          class="af-item"
          :class="{ 'af-item-error': f.status === 'error', 'af-item-disabled': f.disabled || f.status === 'disabled' }"
        >
          <div class="af-indicator" :class="'ind-' + (f.status || 'unknown')" />
          <div class="af-info">
            <div class="af-name">{{ f.name || '—' }}</div>
            <div class="af-meta">
              <span v-if="f.label" class="af-chip">{{ f.label }}</span>
              <span v-if="f.provider" class="af-chip af-chip-muted">{{ f.provider }}</span>
              <span v-if="f.type" class="af-chip af-chip-muted">{{ f.type }}</span>
            </div>
            <div v-if="f.status_message" class="af-msg">{{ f.status_message }}</div>
          </div>
          <div class="af-right">
            <el-tag :type="tagType(f.status)" size="small" round style="font-size:10px">{{ f.status }}</el-tag>
            <span class="af-size">{{ fmtSize(f.size) }}</span>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<style scoped>
.af-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--bg-page);
  overflow: hidden;
}

/* ── Connection status bar ── */
.conn-bar {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 5px 14px;
  font-size: 11.5px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-color);
}
.conn-loading { background: #f8f9fa; color: var(--text-muted); }
.conn-ok      { background: #f0fdf4; color: #15803d; }
.conn-err     { background: #fff5f5; color: #dc2626; }
.conn-warn    { color: #d97706; }

/* ── Header ── */
.af-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 13px 16px 11px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-color);
  background: rgba(242,243,247,0.9);
  backdrop-filter: blur(10px);
}
.af-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text-primary);
}

/* ── Scrollable body ── */
.af-body {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0 0 16px;
}

/* ── Section label ── */
.section-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-muted);
  padding: 14px 14px 6px;
}
.section-count {
  background: var(--color-primary);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 20px;
  letter-spacing: 0;
  text-transform: none;
}

/* ── Usage stats row ── */
.usage-stats-row {
  display: flex;
  margin: 0 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius-md);
  overflow: hidden;
  background: var(--bg-card);
}
.usage-stat {
  flex: 1;
  padding: 10px 4px 8px;
  text-align: center;
  border-right: 1px solid var(--border-color);
}
.usage-stat:last-child { border-right: none; }
.usage-stat-val {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
}
.usage-stat-lbl {
  font-size: 10px;
  color: var(--text-muted);
  margin-top: 3px;
}
.usage-stat.ok     .usage-stat-val { color: #10b981; }
.usage-stat.danger .usage-stat-val { color: #ef4444; }
.usage-stat.info   .usage-stat-val { color: #5b6cf9; }

/* ── Bar chart ── */
.chart-wrap {
  margin: 10px 12px 0;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius-md);
  padding: 10px 12px 8px;
}
.chart-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 8px;
}
.chart-bars {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 56px;
}
.chart-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  height: 100%;
}
.chart-bar-wrap {
  flex: 1;
  width: 100%;
  display: flex;
  align-items: flex-end;
}
.chart-bar {
  width: 100%;
  background: var(--color-primary);
  border-radius: 3px 3px 0 0;
  min-height: 3px;
  opacity: 0.75;
  transition: opacity 0.15s;
}
.chart-col:hover .chart-bar { opacity: 1; }
.chart-label {
  font-size: 9px;
  color: var(--text-muted);
  white-space: nowrap;
}
.chart-val {
  font-size: 9px;
  color: var(--text-secondary);
  font-weight: 600;
}

/* ── Model list ── */
.models-wrap {
  margin: 10px 12px 0;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius-md);
  padding: 10px 12px 8px;
}
.model-list {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.model-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.model-name {
  font-size: 11px;
  color: var(--text-primary);
  font-weight: 500;
  width: 100px;
  flex-shrink: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.model-bar-bg {
  flex: 1;
  height: 6px;
  background: var(--bg-page);
  border-radius: 3px;
  overflow: hidden;
}
.model-bar-fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: 3px;
  opacity: 0.7;
  transition: width 0.3s ease;
}
.model-nums {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0;
  flex-shrink: 0;
  min-width: 60px;
}
.model-req {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1;
}
.model-tok {
  font-size: 9.5px;
  color: var(--text-muted);
  line-height: 1.3;
}

/* ── Auth file stats row ── */
.af-stats {
  display: flex;
  margin: 0 12px 8px;
  border: 1px solid var(--border-color);
  border-radius: var(--border-radius-md);
  overflow: hidden;
  background: var(--bg-card);
}
.af-stat {
  flex: 1;
  padding: 8px 0 6px;
  text-align: center;
  border-right: 1px solid var(--border-color);
}
.af-stat:last-child { border-right: none; }
.af-stat-val {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1;
}
.af-stat-lbl {
  font-size: 10px;
  color: var(--text-muted);
  margin-top: 2px;
}
.af-stat.ok      .af-stat-val { color: #10b981; }
.af-stat.warn    .af-stat-val { color: #f59e0b; }
.af-stat.info    .af-stat-val { color: #5b6cf9; }
.af-stat.danger  .af-stat-val { color: #ef4444; }
.af-stat.muted   .af-stat-val { color: #9ca3af; }

/* ── Center placeholder ── */
.af-center {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.af-error { color: #ef4444; }
.af-muted { color: var(--text-muted); }

/* ── File list ── */
.af-list {
  padding: 0 12px;
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.af-item {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  padding: 9px 10px;
  background: var(--bg-card);
  border-radius: var(--border-radius-md);
  border: 1px solid var(--border-color);
  transition: box-shadow 0.15s;
}
.af-item:hover { box-shadow: var(--shadow-sm); }
.af-item-error    { border-color: #fca5a5; background: #fff5f5; }
.af-item-disabled { opacity: 0.5; }

.af-indicator {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  margin-top: 5px;
  flex-shrink: 0;
  background: #9ca3af;
}
.ind-active     { background: #10b981; box-shadow: 0 0 0 3px rgba(16,185,129,0.2); }
.ind-error      { background: #ef4444; box-shadow: 0 0 0 3px rgba(239,68,68,0.2); }
.ind-pending    { background: #f59e0b; }
.ind-refreshing { background: #5b6cf9; }
.ind-disabled   { background: #9ca3af; }

.af-info { flex: 1; min-width: 0; }
.af-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.af-meta { display: flex; flex-wrap: wrap; gap: 3px; margin-top: 3px; }
.af-chip {
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 20px;
  background: rgba(91,108,249,0.1);
  color: #5b6cf9;
  border: 1px solid rgba(91,108,249,0.2);
  white-space: nowrap;
}
.af-chip-muted { background: var(--bg-page); color: var(--text-muted); border-color: var(--border-color); }
.af-msg { margin-top: 3px; font-size: 10.5px; color: #ef4444; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.af-right { display: flex; flex-direction: column; align-items: flex-end; gap: 3px; flex-shrink: 0; }
.af-size  { font-size: 10px; color: var(--text-muted); }

.spin { animation: spin 1s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
