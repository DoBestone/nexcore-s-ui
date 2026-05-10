<template>
  <div class="page-container">
    <div class="page-header with-actions">
      <div class="page-header-text">
        <h2 class="page-title">{{ $t('pages.blockRules') }}</h2>
        <p class="page-desc">独立的快捷屏蔽规则模块 — 命中即 reject(高优先级,生效在路由列表之前)。备注以「[NexCore]」开头的规则由主控批量下发,本地禁止编辑;其它备注是节点本地手加,可自由增删改。</p>
      </div>
      <div class="page-header-actions">
        <el-button @click="loadList" :loading="loading">
          <el-icon><Refresh /></el-icon>{{ $t('actions.refresh', '刷新') }}
        </el-button>
        <el-button @click="presetsVisible = true">
          <el-icon><MagicStick /></el-icon>应用预置
        </el-button>
        <el-button type="primary" @click="openEdit(null)">
          <el-icon><Plus /></el-icon>{{ $t('actions.add') }}
        </el-button>
      </div>
    </div>

    <div class="nc-card">
      <el-table :data="rules" stripe>
        <el-table-column label="启用" width="80">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enable"
              :disabled="isManaged(row) || togglingId === row.id"
              @change="(val: boolean) => toggleEnable(row, val)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="110">
          <template #default="{ row }">
            <span class="type-pill">{{ row.type }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="value" label="值" min-width="220" show-overflow-tooltip>
          <template #default="{ row }"><span class="mono">{{ row.value }}</span></template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="180">
          <template #default="{ row }">
            <span v-if="isManaged(row)" class="managed-tag">主控</span>{{ stripNexcorePrefix(row.remark) }}
          </template>
        </el-table-column>
        <el-table-column prop="inboundTag" label="入站 tag" width="140">
          <template #default="{ row }">
            <span v-if="row.inboundTag" class="mono">{{ row.inboundTag }}</span>
            <span v-else class="muted">全局</span>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="170">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" align="center">
          <template #default="{ row }">
            <el-tooltip content="编辑" placement="top">
              <el-button text :disabled="isManaged(row)" @click="openEdit(row)">
                <el-icon><Edit /></el-icon>
              </el-button>
            </el-tooltip>
            <el-popconfirm
              :title="`确定删除该屏蔽规则?`"
              :confirm-button-text="$t('yes')"
              :cancel-button-text="$t('no')"
              @confirm="delRule(row.id!)"
            >
              <template #reference>
                <el-button text :disabled="isManaged(row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="rules.length === 0 && !loading" description="暂无屏蔽规则。点「应用预置」可一键导入,或点「新增」自定义" />
    </div>

    <el-dialog v-model="dialogVisible" :title="editing.id ? '编辑屏蔽规则' : '新增屏蔽规则'" width="540px">
      <el-form :model="editing" label-width="86px">
        <el-form-item label="类型" required>
          <el-select v-model="editing.type">
            <el-option v-for="t in typeOptions" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="值" required>
          <el-input v-model="editing.value" placeholder="多个值用英文逗号分隔" />
          <span class="form-hint">{{ valueHint }}</span>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editing.remark" placeholder="给自己看的说明,空也行" />
        </el-form-item>
        <el-form-item label="入站 tag">
          <el-input v-model="editing.inboundTag" placeholder="留空 = 全部入站生效" />
          <span class="form-hint">填某入站 tag 后,本规则只对该入站流量生效;留空则全局拦截</span>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="editing.enable" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveRule">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="presetsVisible" title="应用预置规则" width="560px">
      <p class="preset-tip">每点一次「导入」会新增对应的规则到列表。已存在同样规则不会自动去重。</p>
      <div class="presets-grid">
        <div v-for="p in presets" :key="p.key" class="preset-card">
          <h4>{{ p.name }}</h4>
          <p>{{ p.description }}</p>
          <el-button type="primary" plain :loading="presetApplying === p.key" @click="applyPreset(p)">
            <el-icon><Plus /></el-icon>导入
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { Plus, Edit, Delete, Refresh, MagicStick } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import HttpUtils from '@/plugins/httputil'

interface BlockRule {
  id?: number
  type: string
  value: string
  remark: string
  inboundTag: string
  enable: boolean
  createdAt?: number
}

interface Preset {
  key: string
  name: string
  description: string
  rules: BlockRule[]
}

// 主控的命名空间 — 跟 service/proxy_host_xui_client.go 的 [NexCore] 前缀对齐。
const MANAGED_PREFIX = '[NexCore]'

const rules = ref<BlockRule[]>([])
const loading = ref(false)
const saving = ref(false)
const togglingId = ref<number | null>(null)
const presetApplying = ref<string | null>(null)
const dialogVisible = ref(false)
const presetsVisible = ref(false)
const editing = ref<BlockRule>(emptyRule())

const typeOptions = [
  { value: 'domain',   label: 'domain — 域名(包含子域)' },
  { value: 'ip',       label: 'ip — IP / CIDR' },
  { value: 'geosite',  label: 'geosite — geosite 数据集 tag' },
  { value: 'geoip',    label: 'geoip — geoip 数据集 tag' },
  { value: 'port',     label: 'port — 端口号' },
  { value: 'protocol', label: 'protocol — 嗅探协议(tls/http/quic)' },
  { value: 'source',   label: 'source — 源 IP / CIDR' },
]

// 预置跟 v1 后端的 listBlockRulePresets 对齐(database/model/block_rule.go 注释)
const presets: Preset[] = [
  {
    key: 'ads',
    name: '屏蔽广告(geosite-category-ads-all)',
    description: '命中所有广告 + 反欺诈域名,推荐启用',
    rules: [{ type: 'geosite', value: 'category-ads-all', remark: '屏蔽广告', inboundTag: '', enable: true }],
  },
  {
    key: 'tracker',
    name: '屏蔽追踪器',
    description: '命中常见 analytics / tracker 域名',
    rules: [{ type: 'geosite', value: 'category-public-tracker', remark: '屏蔽追踪器', inboundTag: '', enable: true }],
  },
  {
    key: 'porn',
    name: '屏蔽成人内容(geosite-category-porn)',
    description: '命中成人内容域名;部分场景(家庭/学校网络)需要',
    rules: [{ type: 'geosite', value: 'category-porn', remark: '屏蔽成人内容', inboundTag: '', enable: true }],
  },
]

const valueHint = computed(() => {
  switch (editing.value.type) {
    case 'domain':   return '示例:example.com,ads.com (sing-box 按 domain_suffix 处理,含子域)'
    case 'ip':       return '示例:1.2.3.4,10.0.0.0/8'
    case 'geosite':  return '示例:category-ads-all,cn (sing-box geosite tag)'
    case 'geoip':    return '示例:cn,private (sing-box geoip tag)'
    case 'port':     return '示例:80,443,8080(整数)'
    case 'protocol': return '示例:tls,http,quic (sing-box sniff 后协议名)'
    case 'source':   return '示例:192.168.1.0/24'
    default:         return ''
  }
})

function emptyRule(): BlockRule {
  return { type: 'domain', value: '', remark: '', inboundTag: '', enable: true }
}

const isManaged = (r: BlockRule) => (r.remark || '').startsWith(MANAGED_PREFIX)

const stripNexcorePrefix = (s: string) => (s || '').replace(/^\[NexCore\]\s*/, '')

const formatTime = (ms?: number) => {
  if (!ms) return '—'
  const d = new Date(ms)
  return d.toLocaleString()
}

const loadList = async () => {
  loading.value = true
  try {
    // sui 内部 API 走 LoadPartialData("block-rules"),响应是 {success, obj:{"block-rules":[...]}}
    const res = await HttpUtils.get('api/block-rules')
    if (res.success && res.obj) {
      rules.value = (res.obj['block-rules'] as BlockRule[]) || []
    }
  } finally {
    loading.value = false
  }
}

const openEdit = (r: BlockRule | null) => {
  editing.value = r ? { ...r } : emptyRule()
  dialogVisible.value = true
}

// saveAction 走 sui 内部 /api/save 路径,跟 inbounds/outbounds 同套
const saveAction = async (action: 'new' | 'edit' | 'del', data: any): Promise<boolean> => {
  const fd = new FormData()
  fd.append('object', 'block-rules')
  fd.append('action', action)
  fd.append('data', JSON.stringify(data))
  const res = await HttpUtils.post('api/save', fd as any)
  return !!res.success
}

const saveRule = async () => {
  if (!editing.value.value.trim()) return ElMessage.error('值不能为空')
  saving.value = true
  try {
    const ok = await saveAction(editing.value.id ? 'edit' : 'new', editing.value)
    if (ok) {
      ElMessage.success(editing.value.id ? '已更新,sing-box 后台 reload' : '已新增,sing-box 后台 reload')
      dialogVisible.value = false
      await loadList()
    }
  } finally {
    saving.value = false
  }
}

const delRule = async (id: number) => {
  if (await saveAction('del', [id])) {
    ElMessage.success('已删除')
    await loadList()
  }
}

const toggleEnable = async (r: BlockRule, val: boolean) => {
  togglingId.value = r.id ?? null
  try {
    const ok = await saveAction('edit', { ...r, enable: val })
    if (ok) {
      r.enable = val
      ElMessage.success(val ? '已启用' : '已禁用')
    }
  } finally {
    togglingId.value = null
  }
}

const applyPreset = async (p: Preset) => {
  presetApplying.value = p.key
  try {
    let okCount = 0
    for (const r of p.rules) {
      if (await saveAction('new', r)) okCount++
    }
    ElMessage.success(`已导入 ${okCount} 条规则`)
    presetsVisible.value = false
    await loadList()
  } finally {
    presetApplying.value = null
  }
}

onMounted(loadList)
</script>

<style scoped>
.muted { color: var(--nc-text-muted); }
.mono { font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, monospace); }
.type-pill {
  display: inline-block;
  padding: 1px 8px;
  background: var(--nc-border-soft);
  border-radius: 4px;
  font-size: 12px;
  font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, monospace);
  color: var(--nc-text-2);
}
.managed-tag {
  display: inline-block;
  padding: 1px 6px;
  background: var(--nc-primary-soft);
  color: var(--nc-primary-deep);
  border-radius: 4px;
  font-size: 11px;
  margin-right: 6px;
  font-weight: 600;
}
.form-hint {
  display: block;
  color: var(--nc-text-muted);
  font-size: 12px;
  margin-top: 4px;
  line-height: 1.4;
}
.preset-tip { margin: 0 0 12px; color: var(--nc-text-muted); font-size: 13px; }
.presets-grid {
  display: grid;
  gap: 12px;
}
.preset-card {
  padding: 14px 16px;
  border: 1px solid var(--nc-border);
  border-radius: var(--radius-md);
}
.preset-card h4 { margin: 0 0 4px; font-size: 14px; }
.preset-card p { margin: 0 0 10px; color: var(--nc-text-muted); font-size: 13px; line-height: 1.5; }
</style>
