<template>
  <div class="page-container">
    <div class="page-header with-actions">
      <div class="page-header-text">
        <h2 class="page-title">{{ $t('pages.outbounds') }}</h2>
        <p class="page-desc">{{ $t('outbounds.desc', '配置出站协议与连通性测试') }}</p>
      </div>
      <div class="page-header-actions">
        <el-button type="primary" @click="showModal(0)">
          <el-icon><Plus /></el-icon>{{ $t('actions.add') }}
        </el-button>
        <el-button :loading="testingAll" :disabled="outbounds.length === 0" @click="checkAllOutbounds">
          <el-icon><Stopwatch /></el-icon>{{ $t('actions.testAll', '测试全部') }}
        </el-button>
        <el-button :loading="refreshing" @click="refresh">
          <el-icon><RefreshRight /></el-icon>{{ $t('actions.refresh', '刷新') }}
        </el-button>
      </div>
    </div>

    <div class="ob-toolbar nc-card">
      <el-input
        v-model="filter"
        :placeholder="$t('actions.search', '搜索 tag / 类型 / 地址')"
        clearable
        class="ob-toolbar__search"
      >
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
      <div class="ob-toolbar__stats">
        <span class="ob-stat"><span class="ob-stat__num">{{ outbounds.length }}</span>{{ $t('outbounds.total', '出站') }}</span>
        <span v-if="onlineCount > 0" class="ob-stat ob-stat--ok">
          <span class="status-dot online"></span>{{ onlineCount }} {{ $t('online') }}
        </span>
        <span v-if="passCount > 0" class="ob-stat ob-stat--ok">
          <el-icon><CircleCheck /></el-icon>{{ passCount }} {{ $t('outbounds.pass', '通') }}
        </span>
        <span v-if="failCount > 0" class="ob-stat ob-stat--err">
          <el-icon><CircleClose /></el-icon>{{ failCount }} {{ $t('outbounds.fail', '不通') }}
        </span>
      </div>
    </div>

    <div v-if="outbounds.length === 0" class="empty-state nc-card">
      <el-empty :description="$t('noData', '暂无数据')">
        <el-button type="primary" @click="showModal(0)">
          <el-icon><Plus /></el-icon>{{ $t('actions.add') }}
        </el-button>
      </el-empty>
    </div>

    <el-table
      v-else
      :data="filteredOutbounds"
      stripe
      size="small"
      class="nc-table ob-table"
      :empty-text="$t('noData')"
    >
      <el-table-column :label="$t('type')" width="120">
        <template #default="{ row }">
          <span class="proto-pill" :class="`proto-${row.type}`">{{ row.type }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('objects.tag')" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="mono ob-tag">{{ row.tag }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('out.addr')" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.server" class="mono">
            {{ row.server }}<span class="ob-port">:{{ row.server_port ?? '?' }}</span>
          </span>
          <span v-else class="ob-muted">—</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('objects.tls')" width="80" align="center">
        <template #default="{ row }">
          <el-tag
            v-if="Object.hasOwn(row, 'tls')"
            :type="row.tls?.enabled ? 'success' : 'info'"
            size="small"
            effect="plain"
          >
            {{ row.tls?.enabled ? 'ON' : 'OFF' }}
          </el-tag>
          <span v-else class="ob-muted">—</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('online')" width="80" align="center">
        <template #default="{ row }">
          <span v-if="onlines.includes(row.tag)" class="status-dot online" :title="$t('online')"></span>
          <span v-else class="ob-muted">—</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('out.delay')" width="120" align="center">
        <template #default="{ row }">
          <el-icon v-if="checkResults[row.tag]?.loading" class="is-loading"><Loading /></el-icon>
          <template v-else-if="checkResults[row.tag]">
            <el-tag
              v-if="checkResults[row.tag].success"
              :type="latencyType(checkResults[row.tag].data?.Delay ?? 0)"
              size="small"
              effect="plain"
              class="mono"
            >
              {{ checkResults[row.tag].data?.Delay }} ms
            </el-tag>
            <el-tooltip
              v-else
              :content="checkResults[row.tag].errorMessage || $t('failed')"
              placement="top"
            >
              <el-icon style="color: var(--nc-danger)"><CircleClose /></el-icon>
            </el-tooltip>
          </template>
          <span v-else class="ob-muted">—</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('actions.action')" width="180" align="center">
        <template #default="{ row }">
          <el-tooltip :content="$t('actions.test', '测试')" placement="top">
            <el-button text @click="checkOutbound(row.tag)">
              <el-icon><Stopwatch /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip v-if="Data().enableTraffic" :content="$t('stats.graphTitle', '流量图')" placement="top">
            <el-button text @click="showStats(row.tag)">
              <el-icon><DataLine /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip :content="$t('actions.edit')" placement="top">
            <el-button text @click="showModal(row.id)">
              <el-icon style="color: var(--nc-primary)"><Edit /></el-icon>
            </el-button>
          </el-tooltip>
          <el-tooltip :content="$t('actions.del')" placement="top">
            <el-popconfirm
              :title="$t('confirm')"
              :confirm-button-text="$t('yes')"
              :cancel-button-text="$t('no')"
              @confirm="delOutbound(row.tag)"
            >
              <template #reference>
                <el-button text>
                  <el-icon style="color: var(--nc-danger)"><Delete /></el-icon>
                </el-button>
              </template>
            </el-popconfirm>
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>

    <OutboundVue
      v-model="modal.visible"
      :visible="modal.visible"
      :id="modal.id"
      :data="modal.data"
      :tags="outboundTags"
      @close="closeModal"
    />
    <Stats
      v-model="stats.visible"
      :visible="stats.visible"
      :resource="stats.resource"
      :tag="stats.tag"
      @close="closeStats"
    />
  </div>
</template>

<script lang="ts" setup>
import Data from '@/store/modules/data'
import HttpUtils from '@/plugins/httputil'
import { Outbound } from '@/types/outbounds'
import { computed, defineAsyncComponent, ref } from 'vue'

const OutboundVue = defineAsyncComponent(() => import('@/layouts/modals/Outbound.vue'))
const Stats = defineAsyncComponent(() => import('@/layouts/modals/Stats.vue'))
import {
  Plus, Edit, Delete, DataLine, Stopwatch, Loading, CircleClose, CircleCheck,
  RefreshRight, Search,
} from '@element-plus/icons-vue'

interface CheckResult {
  loading?: boolean
  success: boolean
  data?: { OK?: boolean; Delay?: number; Error?: string } | null
  errorMessage?: string
}

const checkResults = ref<Record<string, CheckResult>>({})

const checkOutbound = async (tag: string) => {
  checkResults.value = { ...checkResults.value, [tag]: { loading: true, success: false } }
  const msg = await HttpUtils.get('api/checkOutbound', { tag })
  const success = msg.success && msg.obj?.OK
  const errorMessage = success ? undefined : (msg.obj?.Error ?? msg.msg ?? '')
  checkResults.value = {
    ...checkResults.value,
    [tag]: { loading: false, success, data: msg.obj ?? null, errorMessage },
  }
}

const testingAll = ref(false)
const checkAllOutbounds = async () => {
  const list = outbounds.value
  if (list.length === 0) return
  testingAll.value = true
  try {
    await Promise.all(list.map((o) => checkOutbound(o.tag)))
  } finally {
    testingAll.value = false
  }
}

const refreshing = ref(false)
const refresh = async () => {
  refreshing.value = true
  try {
    Data().lastLoad = 0
    await Data().loadData()
  } finally {
    refreshing.value = false
  }
}

const filter = ref('')
const outbounds = computed((): Outbound[] => <Outbound[]>(Data().outbounds ?? []))
const filteredOutbounds = computed(() => {
  const q = filter.value.trim().toLowerCase()
  if (!q) return outbounds.value
  return outbounds.value.filter((o: any) =>
    (o.tag || '').toLowerCase().includes(q) ||
    (o.type || '').toLowerCase().includes(q) ||
    (o.server || '').toLowerCase().includes(q),
  )
})
const outboundTags = computed((): string[] => [
  ...(Data().outbounds?.map((o: Outbound) => o.tag) ?? []),
  ...(Data().endpoints?.map((e: any) => e.tag) ?? []),
])
const onlines = computed(() => Data().onlines.outbound ?? [])

const onlineCount = computed(() => outbounds.value.filter((o: any) => onlines.value.includes(o.tag)).length)
const passCount = computed(() =>
  Object.values(checkResults.value).filter((r) => !r.loading && r.success).length,
)
const failCount = computed(() =>
  Object.values(checkResults.value).filter((r) => !r.loading && !r.success && r.data !== undefined).length,
)

const latencyType = (ms: number): 'success' | 'warning' | 'danger' => {
  if (ms < 200) return 'success'
  if (ms < 600) return 'warning'
  return 'danger'
}

const modal = ref({ visible: false, id: 0, data: '' })
const showModal = (id: number) => {
  modal.value.id = id
  modal.value.data = id == 0 ? '' : JSON.stringify(outbounds.value.findLast((o) => o.id == id))
  modal.value.visible = true
}
const closeModal = () => { modal.value.visible = false }

const stats = ref({ visible: false, resource: 'outbound', tag: '' })
const delOutbound = async (tag: string) => {
  await Data().save('outbounds', 'del', tag)
}
const showStats = (tag: string) => {
  stats.value.tag = tag
  stats.value.visible = true
}
const closeStats = () => { stats.value.visible = false }
</script>

<style scoped>
.ob-toolbar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 14px;
  margin-bottom: 12px;
}
.ob-toolbar__search { max-width: 320px; }
.ob-toolbar__stats {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-left: auto;
  font-size: 12.5px;
  color: var(--nc-text-muted);
}
.ob-stat {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.ob-stat__num {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: var(--nc-text-1);
  margin-right: 4px;
}
.ob-stat--ok { color: var(--nc-success); }
.ob-stat--err { color: var(--nc-danger); }

.ob-table { background: var(--nc-surface); }
.ob-table :deep(.el-table__row) td { vertical-align: middle; }

.proto-pill {
  display: inline-block;
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-pill);
  letter-spacing: 0.04em;
  text-transform: uppercase;
  background: var(--nc-primary-soft);
  color: var(--nc-primary);
}
.proto-pill.proto-direct { color: #475569; background: #e2e8f0; }
.proto-pill.proto-block  { color: #dc2626; background: #fee2e2; }
.proto-pill.proto-dns    { color: #7c3aed; background: #ede9fe; }
.proto-pill.proto-vless,
.proto-pill.proto-vmess  { color: #2563eb; background: #dbeafe; }
.proto-pill.proto-trojan { color: #d97706; background: #fef3c7; }
.proto-pill.proto-shadowsocks { color: #16a34a; background: #dcfce7; }
.proto-pill.proto-socks,
.proto-pill.proto-http   { color: #0d9488; background: #ccfbf1; }

.ob-tag { color: var(--nc-text-1); font-weight: 500; }
.ob-port { color: var(--nc-text-muted); font-weight: 400; }
.ob-muted { color: var(--nc-text-faint); }

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--nc-text-faint);
}
.status-dot.online {
  background: var(--nc-success);
  box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.18);
}

.empty-state {
  padding: 40px 16px;
}
</style>
