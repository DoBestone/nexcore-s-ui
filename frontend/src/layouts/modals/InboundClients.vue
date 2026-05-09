<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="$emit('update:modelValue', $event)"
    @close="closeModal"
    class="constrained-dialog is-wide"
    :align-center="false"
    :title="`${$t('pages.clients')} · ${inbound?.tag ?? ''}`"
    destroy-on-close
  >
    <div class="ic-toolbar">
      <el-input
        v-model="filter"
        :placeholder="$t('actions.search', '搜索名称 / 描述')"
        clearable
        class="ic-search"
      >
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
      <div class="ic-stats">
        <span class="ic-stat"><span class="ic-stat__num">{{ scopedClients.length }}</span>{{ $t('main.tiles', '客户') }}</span>
        <span v-if="onlineInScope > 0" class="ic-stat ic-stat--ok">
          <span class="status-dot online"></span>{{ onlineInScope }} {{ $t('online') }}
        </span>
      </div>
      <div class="ic-toolbar__actions">
        <el-button type="primary" size="small" @click="openAdd">
          <el-icon><Plus /></el-icon>{{ $t('actions.add') }}
        </el-button>
      </div>
    </div>

    <el-empty v-if="scopedClients.length === 0" :description="$t('clients.empty', '此入站还没有客户端')">
      <el-button type="primary" @click="openAdd">
        <el-icon><Plus /></el-icon>{{ $t('actions.add') }}
      </el-button>
    </el-empty>

    <el-table
      v-else
      :data="filteredScoped"
      stripe
      size="small"
      class="nc-table ic-table"
      empty-text=" "
    >
      <el-table-column :label="$t('enable', '启用')" width="68" align="center">
        <template #default="{ row }">
          <el-switch
            :model-value="row.enable !== false"
            :loading="toggling[row.id]"
            @change="(v: boolean) => toggleEnable(row, v)"
          />
        </template>
      </el-table-column>
      <el-table-column prop="name" :label="$t('client.name')" min-width="140" sortable show-overflow-tooltip>
        <template #default="{ row }">
          <span class="cli-name">{{ row.name }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="desc" :label="$t('client.desc')" min-width="120" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.desc" class="cli-desc">{{ row.desc }}</span>
          <span v-else class="ic-muted">—</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('stats.volume', '流量')" min-width="170">
        <template #default="{ row }">
          <div class="vol-cell">
            <span class="vol-text mono" :class="volClass(row)">
              {{ row.up + row.down > 0 ? HumanReadable.sizeFormat(row.up + row.down) : '0 B' }}
              <span class="vol-sep">/</span>
              <span class="vol-cap">{{ row.volume == 0 ? '∞' : HumanReadable.sizeFormat(row.volume) }}</span>
            </span>
            <el-progress
              v-if="row.volume > 0"
              :percentage="percent(row)"
              :status="percentStatus(row)"
              :show-text="false"
              :stroke-width="2"
            />
          </div>
        </template>
      </el-table-column>

      <el-table-column :label="$t('date.expiry', '到期')" width="130">
        <template #default="{ row }">
          <span v-if="row.expiry == 0" class="ic-muted mono">永久</span>
          <span v-else-if="row.expiry <= Date.now() / 1000" class="exp-bad mono">已过期</span>
          <span v-else class="exp-ok mono">{{ HumanReadable.remainedDays(row.expiry) }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('online')" width="64" align="center">
        <template #default="{ row }">
          <span v-if="onlineUsers.includes(row.name)" class="status-dot online" :title="$t('online')"></span>
          <span v-else class="ic-muted">—</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('actions.action')" width="156" align="center">
        <template #default="{ row }">
          <div class="row-actions">
            <el-tooltip :content="$t('main.qr', 'QR / 链接')" placement="top">
              <button class="ico-btn" @click="showQr(row)">
                <el-icon style="color: var(--nc-primary)"><Picture /></el-icon>
              </button>
            </el-tooltip>
            <el-tooltip v-if="Data().enableTraffic" :content="$t('stats.graphTitle', '流量图')" placement="top">
              <button class="ico-btn" @click="showStats(row.name)">
                <el-icon><DataLine /></el-icon>
              </button>
            </el-tooltip>
            <el-tooltip :content="$t('actions.edit')" placement="top">
              <button class="ico-btn" @click="openEdit(row.id)">
                <el-icon><Edit /></el-icon>
              </button>
            </el-tooltip>
            <el-popconfirm
              :title="$t('confirm')"
              :confirm-button-text="$t('yes')"
              :cancel-button-text="$t('no')"
              @confirm="delClient(row.id)"
            >
              <template #reference>
                <button class="ico-btn">
                  <el-icon style="color: var(--nc-danger)"><Delete /></el-icon>
                </button>
              </template>
            </el-popconfirm>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <ClientModal
      v-if="modal.visible"
      v-model="modal.visible"
      :visible="modal.visible"
      :id="modal.id"
      :default-inbound-id="inbound?.id ?? 0"
      @close="modal.visible = false"
    />
    <QrCode
      v-if="qr.visible"
      v-model="qr.visible"
      :visible="qr.visible"
      :title="qr.title"
      :link="qr.link"
      empty-text="此客户端在该入站下没有可分享的链接(可能是入站类型不支持 / 链接尚未生成)"
      @close="qr.visible = false"
    />
    <Stats
      v-if="stats.visible"
      v-model="stats.visible"
      :visible="stats.visible"
      :resource="stats.resource"
      :tag="stats.tag"
      @close="stats.visible = false"
    />
  </el-dialog>
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent, ref } from 'vue'
import Data from '@/store/modules/data'
import { HumanReadable } from '@/plugins/utils'
import { Plus, Edit, Delete, Picture, DataLine, Search } from '@element-plus/icons-vue'

const ClientModal = defineAsyncComponent(() => import('@/layouts/modals/Client.vue'))
const QrCode = defineAsyncComponent(() => import('@/layouts/modals/QrCode.vue'))
const Stats = defineAsyncComponent(() => import('@/layouts/modals/Stats.vue'))

const props = defineProps<{ visible: boolean; inbound: any | null }>()
const emit = defineEmits<{ close: []; 'update:modelValue': [v: boolean] }>()

const filter = ref('')
const onlineUsers = computed(() => Data().onlines?.user ?? [])

const allClients = computed((): any[] => Data().clients ?? [])
const scopedClients = computed((): any[] => {
  const id = props.inbound?.id
  if (!id) return []
  return allClients.value.filter((c) => Array.isArray(c.inbounds) && c.inbounds.includes(id))
})

const filteredScoped = computed(() => {
  const q = filter.value.trim().toLowerCase()
  if (!q) return scopedClients.value
  return scopedClients.value.filter((c) =>
    (c.name || '').toLowerCase().includes(q) ||
    (c.desc || '').toLowerCase().includes(q),
  )
})

const onlineInScope = computed(() => scopedClients.value.filter((c) => onlineUsers.value.includes(c.name)).length)

const modal = ref({ visible: false, id: 0 })
const openAdd = () => { modal.value.id = 0; modal.value.visible = true }
const openEdit = (id: number) => { modal.value.id = id; modal.value.visible = true }

const delClient = async (id: number) => {
  await Data().save('clients', 'del', id)
}

const toggling = ref<Record<number, boolean>>({})
const toggleEnable = async (row: any, v: boolean) => {
  toggling.value = { ...toggling.value, [row.id]: true }
  try {
    const fresh = await Data().loadClients(row.id)
    ;(fresh as any).enable = v
    const ok = await Data().save('clients', 'edit', fresh)
    if (ok) row.enable = v
  } finally {
    toggling.value = { ...toggling.value, [row.id]: false }
  }
}

// QR 弹层用纯展示组件,直接把 link / title 传过去 — 学 nexcore-x-ui 的 QrcodeDialog。
// 链接来源:client.links[] 里 remark 等于当前入站 tag 的那一条;否则取第一条。
const qr = ref({ visible: false, title: '', link: '' })
const showQr = (row: any) => {
  const tag = props.inbound?.tag
  const links: Array<{ remark?: string; uri?: string }> = row.links || []
  const matched = (tag && links.find((l) => l.remark === tag)) || links[0]
  qr.value.title = `${row.name}${tag ? ' · ' + tag : ''}`
  qr.value.link = matched?.uri || ''
  qr.value.visible = true
}

const stats = ref({ visible: false, resource: 'user', tag: '' })
const showStats = (name: string) => { stats.value.tag = name; stats.value.visible = true }

const percent = (c: any) => (c.volume > 0 ? Math.round(((c.up + c.down) * 100) / c.volume) : 0)
const percentStatus = (c: any): 'success' | 'warning' | 'exception' =>
  c.up + c.down >= c.volume ? 'exception' : percent(c) > 90 ? 'warning' : 'success'

const volClass = (c: any) => {
  if (c.volume === 0) return 'is-unlimited'
  if (c.up + c.down >= c.volume) return 'is-over'
  if (percent(c) > 90) return 'is-warn'
  return ''
}

const closeModal = () => emit('close')
</script>

<style scoped>
.ic-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.ic-search { max-width: 240px; }
.ic-stats {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 12.5px;
  color: var(--nc-text-muted);
}
.ic-stat { display: inline-flex; align-items: center; gap: 4px; }
.ic-stat__num {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: var(--nc-text-1);
  margin-right: 4px;
}
.ic-stat--ok { color: var(--nc-success); }
.ic-toolbar__actions { margin-left: auto; display: flex; gap: 8px; }
.ic-muted { color: var(--nc-text-faint); }

.cli-name { color: var(--nc-text-1); font-weight: 500; }
.cli-desc { color: var(--nc-text-3); }

.ic-table :deep(.cell) { padding-top: 6px; padding-bottom: 6px; }

.vol-cell { display: flex; flex-direction: column; gap: 4px; }
.vol-text {
  font-size: 12.5px;
  color: var(--nc-text-1);
  white-space: nowrap;
}
.vol-sep { color: var(--nc-text-faint); margin: 0 4px; }
.vol-cap { color: var(--nc-text-muted); }
.vol-text.is-unlimited .vol-cap { color: var(--nc-success); }
.vol-text.is-warn { color: var(--nc-warning, #d97706); }
.vol-text.is-over { color: var(--nc-danger); font-weight: 600; }

.exp-ok { color: var(--nc-text-1); }
.exp-bad { color: var(--nc-danger); font-weight: 600; }

.row-actions {
  display: inline-flex;
  align-items: center;
  gap: 0;
}
.ico-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  cursor: pointer;
  border-radius: 4px;
  font-size: 15px;
  color: var(--nc-text-muted);
  padding: 0;
  transition: background var(--t-fast);
}
.ico-btn:hover { background: var(--nc-border-soft); }

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
</style>
