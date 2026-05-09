<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="$emit('update:modelValue', $event)"
    @close="closeModal"
    class="constrained-dialog is-medium"
    :align-center="false"
    :title="$t('actions.' + title) + ' ' + $t('objects.client')"
    destroy-on-close
  >
    <div v-if="loading" class="modal-loading">
      <el-icon class="is-loading"><Loading /></el-icon>
    </div>

    <el-form v-else label-position="top">
      <div class="form-grid">
        <el-form-item :label="$t('enable')">
          <el-switch v-model="client.enable" />
        </el-form-item>
        <el-form-item :label="$t('client.name')">
          <el-input v-model="client.name" />
        </el-form-item>
        <el-form-item :label="$t('client.desc')">
          <el-input v-model="client.desc" />
        </el-form-item>
        <el-form-item :label="`${$t('stats.volume')} (GiB)`">
          <el-input-number v-model="Volume" :min="0" controls-position="right" style="width: 100%" />
        </el-form-item>
        <DatePick :expiry="expDate" @submit="setDate" />
      </div>

      <div v-if="id > 0" class="usage-summary">
        <div class="usage-summary__row">
          <span>{{ $t('stats.usage') }}: <span class="mono">{{ total }}</span><sup v-if="percent > 0" class="mono">({{ percent }}%)</sup></span>
          <el-tooltip :content="$t('reset')" placement="top">
            <el-button text @click="resetUsage"><el-icon><RefreshRight /></el-icon></el-button>
          </el-tooltip>
        </div>
        <el-progress
          v-if="client.volume > 0"
          :percentage="percent"
          :status="percentStatus"
          :stroke-width="4"
          :show-text="false"
        />
        <div class="usage-summary__row mono">
          <span><el-icon style="color: var(--nc-warning)"><Upload /></el-icon> {{ up }}</span>
          <span>/</span>
          <span><el-icon style="color: var(--nc-success)"><Download /></el-icon> {{ down }}</span>
        </div>
      </div>

    </el-form>

    <template #footer>
      <el-button @click="closeModal">{{ $t('actions.close') }}</el-button>
      <el-button type="primary" :loading="loading" @click="saveChanges">{{ $t('actions.save') }}</el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import { ref, watch, computed } from 'vue'
import { createClient, randomConfigs, updateConfigs } from '@/types/clients'
import DatePick from '@/components/DateTime.vue'
import { HumanReadable } from '@/plugins/utils'
import Data from '@/store/modules/data'
import { Loading, Upload, Download, RefreshRight } from '@element-plus/icons-vue'

// 客户端语义:每条客户端都属于一条特定入站(就像入站底下的"账号"),
// 创建时由 InboundClients 弹出本 modal 时通过 defaultInboundId 注入。
// 编辑模式 client.inbounds 沿用 DB 已有值(可能多入站,这边静默保留)。
const props = defineProps<{ visible: boolean; id: number; defaultInboundId?: number }>()
const emit = defineEmits<{ close: []; 'update:modelValue': [v: boolean] }>()

const client = ref<any>(createClient())
const title = ref('add')
const loading = ref(false)
// 客户端协议侧 config(uuid / password / flow ...)对前端不再可见 — 由后端
// 在保存时以 randomConfigs 生成,用户只关心 name / group / volume / expiry / 入站绑定。
const clientConfig = ref<any>({})

const updateData = async (id: number) => {
  if (id > 0) {
    loading.value = true
    const newData = await Data().loadClients(id)
    client.value = createClient(newData)
    title.value = 'edit'
    clientConfig.value = client.value.config
    loading.value = false
  } else {
    client.value = createClient()
    title.value = 'add'
    clientConfig.value = randomConfigs('client')
    // 新建客户端默认绑定到打开本 modal 的入站,UI 上不再让用户选
    if (props.defaultInboundId && props.defaultInboundId > 0) {
      client.value.inbounds = [props.defaultInboundId]
    }
  }
  loading.value = false
}

const closeModal = () => {
  updateData(0)
  emit('close')
}

const saveChanges = async () => {
  if (!props.visible) return
  if (Data().checkClientName(props.id, client.value.name)) return
  loading.value = true
  // config 字段后端依然要 — 保存时把(可能编辑过的)clientConfig 写回 client.config。
  // 我们前端不再展示 / 重置流程,但模型字段不能丢。
  client.value.config = updateConfigs(clientConfig.value, client.value.name)
  // links 数组留空 — 老的"外部链接"页面已下线。
  client.value.links = []
  // 兜底:UI 上没让用户选入站,如果 inbounds 还是空且打开本 modal 的入站
  // 已知,强制绑过去。否则会出现"创建成功但客户列表 0 条"的灵异 bug。
  if ((!client.value.inbounds || client.value.inbounds.length === 0) &&
      props.defaultInboundId && props.defaultInboundId > 0) {
    client.value.inbounds = [props.defaultInboundId]
  }
  const success = await Data().save('clients', props.id == 0 ? 'new' : 'edit', client.value)
  if (success) closeModal()
  loading.value = false
}

const setDate = (v: number) => { client.value.expiry = v }

const resetUsage = () => {
  client.value.totalUp = (client.value.totalUp ?? 0) + client.value.up
  client.value.totalDown = (client.value.totalDown ?? 0) + client.value.down
  client.value.up = 0
  client.value.down = 0
}

const expDate = computed(() => client.value.expiry)
const Volume = computed({
  get: () => (client.value.volume === 0 ? 0 : client.value.volume / 1024 ** 3),
  set: (v: number) => { client.value.volume = v > 0 ? v * 1024 ** 3 : 0 },
})
const up = computed(() => HumanReadable.sizeFormat(client.value.up))
const down = computed(() => HumanReadable.sizeFormat(client.value.down))
const total = computed(() => HumanReadable.sizeFormat(client.value.up + client.value.down))
const percent = computed(() => (client.value.volume > 0 ? Math.round(((client.value.up + client.value.down) * 100) / client.value.volume) : 0))
const percentStatus = computed<'success' | 'warning' | 'exception'>(() =>
  client.value.up + client.value.down >= client.value.volume ? 'exception' : percent.value > 90 ? 'warning' : 'success',
)

// immediate:true — 上层用 v-if 包裹本 modal,组件首次创建时 props.visible
// 已经是 true,普通 watch 不会触发初始 updateData,导致新客户的 defaultInboundId
// 没被注入。加 immediate 让初始化逻辑立刻跑一次。
watch(() => props.visible, (v) => { if (v) updateData(props.id) }, { immediate: true })
</script>

<style scoped>
.modal-loading {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 64px 0;
  font-size: 32px;
  color: var(--nc-primary);
}
.client-tabs {
  background: transparent;
}
.client-tabs :deep(.el-tabs__nav-wrap::after) { display: none; }

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 6px 16px;
}

.usage-summary {
  background: var(--nc-border-soft);
  border-radius: var(--radius-md);
  padding: 12px 14px;
  margin: 6px 0 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 12.5px;
}

.usage-summary__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.usage-summary__reset {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 8px;
  font-size: 11.5px;
  color: var(--nc-text-muted);
}

.config-section {
  margin-bottom: 12px;
}

.config-row {
  display: grid;
  grid-template-columns: 110px 1fr;
  gap: 10px;
  margin-bottom: 10px;
  align-items: center;
}

.config-row__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 4px;
  font-size: 12px;
  font-weight: 600;
  color: var(--nc-text-muted);
}

.config-row__name {
  font-family: var(--font-display);
}

.links-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}

.link-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  background: var(--nc-border-soft);
  padding: 6px 10px;
  border-radius: var(--radius-md);
}

.link-row__index {
  font-family: var(--font-mono);
  color: var(--nc-text-muted);
  flex-shrink: 0;
}

.link-row__uri {
  font-family: var(--font-mono);
  color: var(--nc-text-3);
  word-break: break-all;
}

.links-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 14px;
}

.link-input-row {
  display: flex;
  gap: 4px;
}
</style>
