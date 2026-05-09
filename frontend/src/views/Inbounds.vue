<template>
  <div class="page-container">
    <div class="page-header with-actions">
      <div class="page-header-text">
        <h2 class="page-title">{{ $t('pages.inbounds') }}</h2>
        <p class="page-desc">{{ $t('inbounds.desc', '配置 Sing-Box 入站协议、监听端口与关联用户') }}</p>
      </div>
      <div class="page-header-actions">
        <el-button type="primary" @click="showModal(0)">
          <el-icon><Plus /></el-icon>{{ $t('actions.add') }}
        </el-button>
        <el-button :loading="refreshing" @click="refresh">
          <el-icon><RefreshRight /></el-icon>{{ $t('actions.refresh', '刷新') }}
        </el-button>
        <el-button v-if="inbounds.length + totalClients > 0" type="danger" plain @click="purgeAll">
          <el-icon><Delete /></el-icon>清除全部数据
        </el-button>
      </div>
    </div>

    <!-- 防火墙告警 -->
    <el-alert
      v-if="firewall?.active && blockedPorts.length"
      type="warning"
      :closable="false"
      show-icon
      class="firewall-alert"
    >
      <template #title>
        系统防火墙({{ firewall.tool.toUpperCase() }})阻挡入站端口:{{ blockedPorts.join(', ') }}
      </template>
      <template #default>
        <div class="firewall-alert-body">
          客户端从外网连不进来。在服务器上执行:
          <code class="firewall-alert-cmd">{{ firewallFix }}</code>
        </div>
      </template>
    </el-alert>

    <!-- 概览卡 -->
    <div class="overview nc-card">
      <div class="ov-stat">
        <span class="ov-stat__label">{{ $t('pages.inbounds') }}</span>
        <span class="ov-stat__value">{{ inbounds.length }}</span>
      </div>
      <div class="ov-stat">
        <span class="ov-stat__label">{{ $t('pages.clients') }}</span>
        <span class="ov-stat__value">{{ totalClients }}</span>
      </div>
      <div v-if="onlineCount > 0" class="ov-stat ov-stat--ok">
        <span class="ov-stat__label">
          <span class="status-dot online"></span>{{ $t('online') }}
        </span>
        <span class="ov-stat__value">{{ onlineCount }}</span>
      </div>
      <div class="ov-stat">
        <span class="ov-stat__label">{{ $t('home.topTraffic.up', '上行') }} / {{ $t('home.topTraffic.down', '下行') }}</span>
        <span class="ov-stat__value ov-stat__value--small">
          {{ HumanReadable.sizeFormat(totalUp) }}
          <span class="muted">/</span>
          {{ HumanReadable.sizeFormat(totalDown) }}
        </span>
      </div>
      <div v-if="totalUp + totalDown > 0" class="ov-stat">
        <span class="ov-stat__label">{{ $t('stats.totalUsage', '总流量') }}</span>
        <span class="ov-stat__value">{{ HumanReadable.sizeFormat(totalUp + totalDown) }}</span>
      </div>
      <div class="ov-toolbar">
        <el-input
          v-model="filter"
          :placeholder="$t('actions.search', '搜索 tag / 类型 / 端口')"
          clearable
          class="ov-search"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
      </div>
    </div>

    <div v-if="inbounds.length === 0" class="empty-state nc-card">
      <el-empty :description="$t('noData', '暂无入站')">
        <el-button type="primary" @click="showModal(0)">
          <el-icon><Plus /></el-icon>{{ $t('actions.add') }}
        </el-button>
      </el-empty>
    </div>

    <!-- 多用户入站:VLESS / VMess / Trojan / Hysteria2 / Tuic / SS-2022 / AnyTLS … -->
    <el-card v-if="multiUser.length > 0" class="cat-card">
      <template #header>
        <div class="cat-head">
          <el-icon class="cat-head__icon"><User /></el-icon>
          <span class="cat-head__title">多用户入站</span>
          <span class="cat-head__sub">VLESS / VMess / Trojan / SS-2022 等 — 一个端口多个客户</span>
          <span class="cat-head__count">{{ multiUser.length }}</span>
        </div>
      </template>
      <el-table :data="filtered(multiUser)" stripe size="small" class="nc-table ib-table">
        <el-table-column :label="$t('enable', '启用')" width="68" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enable !== false"
              :loading="toggling[row.id]"
              @change="(v: boolean) => toggleEnable(row, v)"
            />
          </template>
        </el-table-column>
        <el-table-column :label="$t('type')" width="120">
          <template #default="{ row }">
            <span class="proto-pill" :class="`proto-${row.type}`">{{ row.type }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('objects.tag')" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mono">{{ row.tag }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('in.port', '端口')" width="100" align="center">
          <template #default="{ row }">
            <span class="mono">{{ row.listen_port }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('inbounds.clientCount', '客户数')" width="100" align="center">
          <template #default="{ row }">
            <button v-if="row.users?.length" class="client-pill" @click="openClients(row)">
              <el-icon><User /></el-icon>{{ row.users.length }}
            </button>
            <span v-else class="muted">0</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('stats.totalUsage', '总流量')" min-width="170">
          <template #default="{ row }">
            <span v-if="trafficOf(row.tag)" class="mono traffic-cell">
              {{ HumanReadable.sizeFormat(trafficOf(row.tag)) }}
              <span class="traffic-detail">↑ {{ HumanReadable.sizeFormat(traffic[row.tag]?.up ?? 0) }} / ↓ {{ HumanReadable.sizeFormat(traffic[row.tag]?.down ?? 0) }}</span>
            </span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('objects.tls')" width="68" align="center">
          <template #default="{ row }">
            <el-tag :type="row.tls_id > 0 ? 'success' : 'info'" size="small" effect="plain">
              {{ row.tls_id > 0 ? 'ON' : 'OFF' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('online')" width="64" align="center">
          <template #default="{ row }">
            <span v-if="onlines.includes(row.tag)" class="status-dot online"></span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('actions.action')" width="180" align="center">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="openClients(row)">
              <el-icon><User /></el-icon>客户端
            </el-button>
            <el-dropdown trigger="click" @command="(cmd: string) => onMore(cmd, row)">
              <el-button size="small">
                更多<el-icon><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="edit">
                    <el-icon style="margin-right: 6px"><Edit /></el-icon>{{ $t('actions.edit') }}
                  </el-dropdown-item>
                  <el-dropdown-item command="clone">
                    <el-icon style="margin-right: 6px"><CopyDocument /></el-icon>{{ $t('actions.clone') }}
                  </el-dropdown-item>
                  <el-dropdown-item v-if="Data().enableTraffic" command="stats">
                    <el-icon style="margin-right: 6px"><DataLine /></el-icon>{{ $t('stats.graphTitle', '流量图') }}
                  </el-dropdown-item>
                  <el-dropdown-item command="reset" divided>
                    <el-icon style="margin-right: 6px"><RefreshLeft /></el-icon>{{ $t('actions.resetTraffic', '重置流量') }}
                  </el-dropdown-item>
                  <el-dropdown-item command="del" divided>
                    <el-icon style="margin-right: 6px; color: var(--nc-danger)"><Delete /></el-icon>
                    <span style="color: var(--nc-danger)">{{ $t('actions.del') }}</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 单用户入站:Direct / Mixed / Socks / HTTP / Tun / Naive(无 client 维度) -->
    <el-card v-if="singleUser.length > 0" class="cat-card">
      <template #header>
        <div class="cat-head">
          <el-icon class="cat-head__icon"><Connection /></el-icon>
          <span class="cat-head__title">单用户入站</span>
          <span class="cat-head__sub">Direct / Mixed / Socks / HTTP / Tun / Naive — 端口本身就是入口</span>
          <span class="cat-head__count">{{ singleUser.length }}</span>
        </div>
      </template>
      <el-table :data="filtered(singleUser)" stripe size="small" class="nc-table ib-table">
        <el-table-column :label="$t('enable', '启用')" width="68" align="center">
          <template #default="{ row }">
            <el-switch
              :model-value="row.enable !== false"
              :loading="toggling[row.id]"
              @change="(v: boolean) => toggleEnable(row, v)"
            />
          </template>
        </el-table-column>
        <el-table-column :label="$t('type')" width="120">
          <template #default="{ row }">
            <span class="proto-pill" :class="`proto-${row.type}`">{{ row.type }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('objects.tag')" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mono">{{ row.tag }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('in.addr', '监听')" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mono">{{ row.listen || '0.0.0.0' }}<span class="port">:{{ row.listen_port }}</span></span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('stats.totalUsage', '总流量')" min-width="170">
          <template #default="{ row }">
            <span v-if="trafficOf(row.tag)" class="mono traffic-cell">
              {{ HumanReadable.sizeFormat(trafficOf(row.tag)) }}
              <span class="traffic-detail">↑ {{ HumanReadable.sizeFormat(traffic[row.tag]?.up ?? 0) }} / ↓ {{ HumanReadable.sizeFormat(traffic[row.tag]?.down ?? 0) }}</span>
            </span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('online')" width="64" align="center">
          <template #default="{ row }">
            <span v-if="onlines.includes(row.tag)" class="status-dot online"></span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('actions.action')" width="180" align="center">
          <template #default="{ row }">
            <el-button size="small" @click="showModal(row.id)">
              <el-icon><Edit /></el-icon>编辑
            </el-button>
            <el-dropdown trigger="click" @command="(cmd: string) => onMore(cmd, row)">
              <el-button size="small">
                更多<el-icon><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="clone">
                    <el-icon style="margin-right: 6px"><CopyDocument /></el-icon>{{ $t('actions.clone') }}
                  </el-dropdown-item>
                  <el-dropdown-item v-if="Data().enableTraffic" command="stats">
                    <el-icon style="margin-right: 6px"><DataLine /></el-icon>{{ $t('stats.graphTitle', '流量图') }}
                  </el-dropdown-item>
                  <el-dropdown-item command="reset" divided>
                    <el-icon style="margin-right: 6px"><RefreshLeft /></el-icon>{{ $t('actions.resetTraffic', '重置流量') }}
                  </el-dropdown-item>
                  <el-dropdown-item command="del" divided>
                    <el-icon style="margin-right: 6px; color: var(--nc-danger)"><Delete /></el-icon>
                    <span style="color: var(--nc-danger)">{{ $t('actions.del') }}</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <InboundVue
      v-model="modal.visible"
      :visible="modal.visible"
      :id="modal.id"
      :inTags="inTags"
      :tlsConfigs="tlsConfigs"
      @close="closeModal"
    />
    <InboundClients
      v-if="clientsModal.visible"
      v-model="clientsModal.visible"
      :visible="clientsModal.visible"
      :inbound="clientsModal.inbound"
      @close="clientsModal.visible = false"
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
import { Config } from '@/types/config'
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import HttpUtils from '@/plugins/httputil'
import { HumanReadable } from '@/plugins/utils'
import { ElMessage, ElMessageBox } from 'element-plus'
import { i18n } from '@/locales'

const InboundVue = defineAsyncComponent(() => import('@/layouts/modals/Inbound.vue'))
const InboundClients = defineAsyncComponent(() => import('@/layouts/modals/InboundClients.vue'))
const Stats = defineAsyncComponent(() => import('@/layouts/modals/Stats.vue'))
import { createInbound, Inbound } from '@/types/inbounds'
import RandomUtil from '@/plugins/randomUtil'
import {
  Plus, Edit, Delete, CopyDocument, DataLine, RefreshRight, RefreshLeft,
  Search, User, Connection, ArrowDown,
} from '@element-plus/icons-vue'

const appConfig = computed((): Config => <Config>Data().config)
void appConfig

const filter = ref('')

const inbounds = computed((): any[] => <any[]>(Data().inbounds ?? []))
const tlsConfigs = computed((): any[] => <any[]>(Data().tlsConfigs ?? []))

// 多用户协议:有 users 字段(后端按协议特性塞);其它一律单用户。
// SS 在 ss-2022 multi-user 模式下也算多用户(后端已正确暴露 users 字段)。
const multiUser = computed(() => inbounds.value.filter((i: any) => Array.isArray(i.users)))
const singleUser = computed(() => inbounds.value.filter((i: any) => !Array.isArray(i.users)))

const filtered = (list: any[]) => {
  const q = filter.value.trim().toLowerCase()
  if (!q) return list
  return list.filter((i) =>
    (i.tag || '').toLowerCase().includes(q) ||
    (i.type || '').toLowerCase().includes(q) ||
    String(i.listen_port || '').includes(q),
  )
}

const inTags = computed((): string[] => [
  ...(inbounds.value?.map((i) => i.tag) ?? []),
  ...(Data().endpoints?.filter((e: any) => e.listen_port > 0).map((e: any) => e.tag) ?? []),
])

const onlines = computed(() => Data().onlines.inbound ?? [])
const onlineUsers = computed(() => Data().onlines?.user ?? [])

const totalClients = computed(() => (Data().clients ?? []).length)
const onlineCount = computed(() => onlineUsers.value.length)

const totalUp = computed(() => Object.values(traffic.value).reduce((s, t) => s + (t.up || 0), 0))
const totalDown = computed(() => Object.values(traffic.value).reduce((s, t) => s + (t.down || 0), 0))

const modal = ref({ visible: false, id: 0 })
const showModal = (id: number) => { modal.value.id = id; modal.value.visible = true }
const closeModal = () => { modal.value.visible = false }

const clientsModal = ref<{ visible: boolean; inbound: any | null }>({ visible: false, inbound: null })
const openClients = (inbound: any) => {
  clientsModal.value.inbound = inbound
  clientsModal.value.visible = true
}

// 「更多」下拉的统一处理 — 收口在一个 switch 里,代码块复用
const onMore = (cmd: string, row: any) => {
  switch (cmd) {
    case 'edit':  showModal(row.id); break
    case 'clone': clone(row.id); break
    case 'stats': showStats(row.tag); break
    case 'reset': resetTraffic(row); break
    case 'del':   confirmDel(row); break
  }
}

const confirmDel = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      `确认删除入站 ${row.tag}(${row.type})?引用此入站的客户会被解绑。`,
      i18n.global.t('actions.del'),
      { type: 'warning', confirmButtonText: i18n.global.t('yes'), cancelButtonText: i18n.global.t('no') },
    )
  } catch { return }
  await Data().save('inbounds', 'del', row.tag)
}

const purgeAll = async () => {
  const ibCount = inbounds.value.length
  const cliCount = (Data().clients ?? []).length
  try {
    await ElMessageBox.confirm(
      `确认清除全部 ${ibCount} 个入站 + ${cliCount} 个客户端?\n` +
      `sing-box 重启后所有连接立即失效,流量历史一并删除,无法恢复。`,
      '清除全部数据',
      { type: 'error', confirmButtonText: '确认清除', cancelButtonText: i18n.global.t('no') },
    )
  } catch { return }

  // 先删客户端 — 客户端被入站引用,先解绑再删入站可避免一致性中间态被 sing-box
  // 短暂下发(虽然事务包裹,但保险起见)。两段都是按 id/tag 串行调 del。
  for (const c of [...(Data().clients ?? [])]) {
    await Data().save('clients', 'del', (c as any).id)
  }
  for (const i of [...inbounds.value]) {
    await Data().save('inbounds', 'del', (i as any).tag)
  }

  // 同时把流量样本一起清掉 — 上面的 del 不会主动清 stats 表,
  // 不清的话"总流量"列还会保留旧 tag 的数字
  for (const tag of Object.keys(traffic.value)) {
    await HttpUtils.post('api/resetTraffic', { resource: 'inbound', tag })
  }
  await loadTraffic()

  ElMessage.success(`已清除 ${ibCount} 个入站 + ${cliCount} 个客户端`)
}

const resetTraffic = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      `重置入站「${row.tag}」的累计流量?(关联客户的 per-user 流量不受影响)`,
      i18n.global.t('actions.resetTraffic', '重置流量'),
      { type: 'warning', confirmButtonText: '重置', cancelButtonText: i18n.global.t('no') },
    )
  } catch { return }
  const r = await HttpUtils.post('api/resetTraffic', { resource: 'inbound', tag: row.tag })
  if (r.success) {
    ElMessage.success('已重置')
    await loadTraffic()
  }
}

const cloneLoading = ref(false)
const clone = async (id: number) => {
  cloneLoading.value = true
  try {
    const inboundArray = await Data().loadInbounds([id])
    const inbound = inboundArray[0]
    const newTag = inbound.type + '-' + RandomUtil.randomSeq(3)
    const newInbound = createInbound(inbound.type, {
      ...inbound,
      id: 0,
      tag: newTag,
      listen_port: RandomUtil.randomIntRange(10000, 60000),
    })
    await Data().save('inbounds', 'new', newInbound)
  } finally {
    cloneLoading.value = false
  }
}

const refreshing = ref(false)
const refresh = async () => {
  refreshing.value = true
  try {
    Data().lastLoad = 0
    await Promise.all([Data().loadData(), loadTraffic(), loadFirewall()])
  } finally {
    refreshing.value = false
  }
}

const stats = ref({ visible: false, resource: 'inbound', tag: '' })
const showStats = (tag: string) => { stats.value.tag = tag; stats.value.visible = true }
const closeStats = () => { stats.value.visible = false }

// ---------- 启用/禁用 ----------
const toggling = ref<Record<number, boolean>>({})
const toggleEnable = async (row: any, v: boolean) => {
  toggling.value = { ...toggling.value, [row.id]: true }
  try {
    const inboundArray = await Data().loadInbounds([row.id])
    const inbound = inboundArray[0]
    inbound.enable = v
    const ok = await Data().save('inbounds', 'edit', inbound)
    if (ok) row.enable = v
  } finally {
    toggling.value = { ...toggling.value, [row.id]: false }
  }
}

// ---------- 总流量 ----------
const traffic = ref<Record<string, { up: number; down: number }>>({})
const trafficOf = (tag: string) => {
  const t = traffic.value[tag]
  if (!t) return 0
  return (t.up || 0) + (t.down || 0)
}
const loadTraffic = async () => {
  const r = await HttpUtils.get('api/statsTotals', { resource: 'inbound' })
  if (r.success) traffic.value = r.obj || {}
}

// ---------- 防火墙状态 ----------
interface FirewallStatus {
  active: boolean
  tool: string
  openPorts: number[] | null
  openRanges: { lo: number; hi: number }[] | null
}
const firewall = ref<FirewallStatus | null>(null)
const blockedPorts = computed<number[]>(() => {
  const fw = firewall.value
  if (!fw || !fw.active) return []
  const isOpen = (port: number): boolean => {
    if ((fw.openPorts || []).includes(port)) return true
    for (const r of fw.openRanges || []) {
      if (port >= r.lo && port <= r.hi) return true
    }
    return false
  }
  const blocked = new Set<number>()
  for (const i of inbounds.value) {
    const p = (i as any).listen_port
    if (typeof p === 'number' && (i as any).enable !== false && !isOpen(p)) blocked.add(p)
  }
  return [...blocked].sort((a, b) => a - b)
})
const firewallFix = computed(() => {
  const fw = firewall.value
  if (!fw) return ''
  const ports = blockedPorts.value
  if (fw.tool === 'firewalld') {
    return ports.map((p) => `firewall-cmd --permanent --add-port=${p}/tcp`).join(' && ') + ' && firewall-cmd --reload'
  }
  return ports.map((p) => `ufw allow ${p}/tcp`).join(' && ')
})
const loadFirewall = async () => {
  const r = await HttpUtils.get('api/firewallStatus')
  if (r.success) firewall.value = r.obj
}

// ---------- lifecycle ----------
let trafficTimer: any
onMounted(async () => {
  await Promise.all([loadTraffic(), loadFirewall()])
  trafficTimer = setInterval(() => { if (!document.hidden) loadTraffic() }, 15000)
})
onBeforeUnmount(() => { if (trafficTimer) clearInterval(trafficTimer) })
</script>

<style scoped>
.overview {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 14px 18px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.ov-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 60px;
}
.ov-stat__label {
  font-size: 11px;
  color: var(--nc-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.ov-stat__value {
  font-family: var(--font-display);
  font-size: 20px;
  font-weight: 700;
  color: var(--nc-text-1);
}
.ov-stat__value--small { font-size: 14px; font-weight: 600; }
.ov-stat--ok .ov-stat__value { color: var(--nc-success); }
.ov-toolbar { margin-left: auto; }
.ov-search { width: 240px; max-width: 100%; }

.cat-card {
  margin-bottom: 12px;
  border: 1px solid var(--nc-border);
  border-radius: var(--radius-lg);
}
.cat-card :deep(.el-card__header) {
  padding: 12px 16px;
  background: #f8fafc;
  border-bottom: 1px solid var(--nc-border);
}
.cat-card :deep(.el-card__body) { padding: 0; }

.cat-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.cat-head__icon { color: var(--nc-primary); font-size: 16px; }
.cat-head__title { font-size: 13.5px; font-weight: 600; color: var(--nc-text-1); }
.cat-head__sub { font-size: 12px; color: var(--nc-text-muted); flex: 1; }
.cat-head__count {
  font-family: var(--font-display);
  font-size: 12px;
  font-weight: 600;
  color: var(--nc-primary);
  background: var(--nc-primary-soft);
  padding: 2px 10px;
  border-radius: var(--radius-pill);
}

.ib-table { background: var(--nc-surface); }
.ib-table .mono { color: var(--nc-text-1); font-weight: 500; }
.ib-table .port { color: var(--nc-text-muted); font-weight: 400; }

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
.proto-pill.proto-direct,
.proto-pill.proto-mixed   { color: #475569; background: #e2e8f0; }
.proto-pill.proto-vless,
.proto-pill.proto-vmess   { color: #2563eb; background: #dbeafe; }
.proto-pill.proto-trojan  { color: #d97706; background: #fef3c7; }
.proto-pill.proto-shadowsocks { color: #16a34a; background: #dcfce7; }
.proto-pill.proto-hysteria,
.proto-pill.proto-hysteria2 { color: #7c3aed; background: #ede9fe; }
.proto-pill.proto-tuic    { color: #0d9488; background: #ccfbf1; }
.proto-pill.proto-naive   { color: #db2777; background: #fce7f3; }
.proto-pill.proto-anytls  { color: #4f46e5; background: #e0e7ff; }
.proto-pill.proto-tun     { color: #0891b2; background: #cffafe; }
.proto-pill.proto-socks,
.proto-pill.proto-http    { color: #0ea5e9; background: #e0f2fe; }
.proto-pill.proto-shadowtls { color: #d97706; background: #fef3c7; }

.client-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--nc-primary-soft);
  color: var(--nc-primary);
  border: none;
  padding: 2px 10px;
  border-radius: var(--radius-pill);
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: background var(--t-fast);
}
.client-pill:hover { background: var(--nc-primary); color: #fff; }

.muted { color: var(--nc-text-faint); }

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

.empty-state { padding: 40px 16px; }

.firewall-alert { margin-bottom: 12px; }
.firewall-alert-body { margin-top: 4px; font-size: 12.5px; }
.firewall-alert-cmd {
  display: inline-block;
  margin-left: 6px;
  padding: 2px 8px;
  background: rgba(0, 0, 0, 0.06);
  border-radius: 4px;
  font-family: var(--font-mono);
  font-size: 12px;
  word-break: break-all;
}

.traffic-cell {
  display: inline-flex;
  align-items: baseline;
  gap: 8px;
}
.traffic-detail {
  font-size: 11px;
  color: var(--nc-text-muted);
  font-weight: 400;
}
</style>
