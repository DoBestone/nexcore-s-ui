<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="$emit('update:modelValue', $event)"
    @close="onClose"
    class="constrained-dialog is-medium"
    :align-center="false"
    :title="$t('tls.cf.title')"
    destroy-on-close
  >
    <el-steps :active="step" finish-status="success" simple class="cf-steps">
      <el-step :title="$t('tls.cf.step1')" />
      <el-step :title="$t('tls.cf.step2')" />
      <el-step :title="$t('tls.cf.step3')" />
    </el-steps>

    <el-form label-position="top" v-if="step === 0">
      <el-alert :title="$t('tls.cf.tokenHint')" type="info" :closable="false" show-icon class="cf-alert" />
      <el-form-item :label="$t('tls.cf.token')">
        <el-input
          v-model="form.token"
          show-password
          type="password"
          :placeholder="$t('tls.cf.tokenPlaceholder')"
        />
      </el-form-item>
      <el-form-item :label="$t('tls.cf.email')">
        <el-input v-model="form.email" :placeholder="`admin@${form.fqdn || 'example.com'}`" />
      </el-form-item>
    </el-form>

    <el-form label-position="top" v-if="step === 1">
      <el-form-item :label="$t('tls.cf.zone')">
        <el-select v-model="form.zoneId" filterable :placeholder="$t('tls.cf.zonePlaceholder')" style="width:100%">
          <el-option v-for="z in zones" :key="z.id" :label="z.name" :value="z.id">
            <span class="zone-row">
              <span>{{ z.name }}</span>
              <el-tag size="small" :type="z.status === 'active' ? 'success' : 'warning'" effect="plain">{{ z.status }}</el-tag>
            </span>
          </el-option>
        </el-select>
      </el-form-item>
      <el-form-item :label="$t('tls.cf.prefixMode')">
        <el-radio-group v-model="form.prefixMode">
          <el-radio value="random">{{ $t('tls.cf.prefixRandom') }}</el-radio>
          <el-radio value="custom">{{ $t('tls.cf.prefixCustom') }}</el-radio>
          <el-radio value="apex">{{ $t('tls.cf.prefixApex') }}</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item v-if="form.prefixMode === 'random'" :label="$t('tls.cf.prefixSeed')">
        <el-input v-model="form.prefix" :placeholder="$t('tls.cf.prefixSeedPlaceholder')" />
        <p class="form-hint">{{ $t('tls.cf.prefixSeedHint') }}</p>
      </el-form-item>
      <el-form-item v-if="form.prefixMode === 'custom'" :label="$t('tls.cf.subdomain')">
        <el-input v-model="form.customName" :placeholder="`api.${zoneNameOf(form.zoneId)}`" />
      </el-form-item>
      <el-form-item :label="$t('tls.cf.ip')">
        <el-input v-model="form.ip" :placeholder="$t('tls.cf.ipPlaceholder')">
          <template #append>
            <el-button @click="detectIp">{{ $t('tls.cf.detectIp') }}</el-button>
          </template>
        </el-input>
      </el-form-item>
      <el-form-item>
        <el-checkbox v-model="form.proxied">{{ $t('tls.cf.proxied') }}</el-checkbox>
        <p class="form-hint">{{ $t('tls.cf.proxiedHint') }}</p>
      </el-form-item>
    </el-form>

    <el-form label-position="top" v-if="step === 2">
      <el-form-item :label="$t('tls.cf.tlsName')">
        <el-input v-model="form.tlsName" :placeholder="result.fqdn || 'cf-auto'" />
      </el-form-item>
      <el-alert v-if="result.fqdn" type="success" :closable="false" show-icon class="cf-alert">
        <template #title>
          <span>{{ $t('tls.cf.dnsReady') }}: <span class="mono">{{ result.fqdn }} → {{ form.ip }}</span></span>
        </template>
      </el-alert>
      <p class="form-hint">{{ $t('tls.cf.issueExplain') }}</p>
    </el-form>

    <template #footer>
      <el-button @click="onClose">{{ $t('actions.close') }}</el-button>
      <el-button v-if="step > 0" @click="step--">{{ $t('tls.cf.back') }}</el-button>
      <el-button v-if="step === 0" type="primary" :loading="loading" :disabled="!form.token || !form.email" @click="onVerify">
        {{ $t('tls.cf.verifyContinue') }}
      </el-button>
      <el-button v-if="step === 1" type="primary" :loading="loading" :disabled="!canApplyDns" @click="onApplyDns">
        {{ $t('tls.cf.applyDns') }}
      </el-button>
      <el-button v-if="step === 2" type="primary" :loading="loading" :disabled="!result.fqdn" @click="onIssue">
        {{ $t('tls.cf.issue') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import { ref, computed, watch } from 'vue'
import HttpUtils from '@/plugins/httputil'
import { ElMessage } from 'element-plus'
import { i18n } from '@/locales'
import Data from '@/store/modules/data'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ close: []; 'update:modelValue': [v: boolean]; created: [id: number] }>()

const step = ref(0)
const loading = ref(false)

interface Form {
  token: string
  email: string
  zoneId: string
  prefixMode: 'random' | 'custom' | 'apex'
  prefix: string
  customName: string
  ip: string
  proxied: boolean
  fqdn: string
  tlsName: string
}

const initialForm = (): Form => ({
  token: '',
  email: '',
  zoneId: '',
  prefixMode: 'random',
  prefix: '',
  customName: '',
  ip: '',
  proxied: false,
  fqdn: '',
  tlsName: '',
})

const form = ref<Form>(initialForm())
const zones = ref<any[]>([])
const result = ref<{ fqdn: string; recordId: string }>({ fqdn: '', recordId: '' })

const zoneNameOf = (id: string) => zones.value.find((z) => z.id === id)?.name || 'example.com'

const canApplyDns = computed(() => {
  if (!form.value.zoneId || !form.value.ip) return false
  if (form.value.prefixMode === 'custom' && !form.value.customName) return false
  return true
})

const onClose = () => {
  emit('close')
  emit('update:modelValue', false)
}

const onVerify = async () => {
  loading.value = true
  const r = await HttpUtils.post('api/cfListZones', { token: form.value.token })
  loading.value = false
  if (r.success) {
    zones.value = r.obj || []
    if (zones.value.length === 0) {
      ElMessage.warning(i18n.global.t('tls.cf.noZone'))
      return
    }
    if (zones.value.length === 1) form.value.zoneId = zones.value[0].id
    step.value = 1
  }
}

const detectIp = async () => {
  // 用 ipify 直拉公网 IP - 浏览器环境直连,与面板无关
  try {
    const r = await fetch('https://api64.ipify.org?format=json')
    const j = await r.json()
    if (j.ip) form.value.ip = j.ip
  } catch {
    ElMessage.error(i18n.global.t('tls.cf.detectIpFailed'))
  }
}

const onApplyDns = async () => {
  loading.value = true
  const payload: any = {
    token: form.value.token,
    zoneId: form.value.zoneId,
    ip: form.value.ip,
    proxied: form.value.proxied,
  }
  if (form.value.prefixMode === 'random') {
    payload.random = true
    payload.prefix = form.value.prefix
  } else if (form.value.prefixMode === 'custom') {
    payload.name = form.value.customName
  } else {
    payload.name = '@'
  }
  const r = await HttpUtils.post('api/cfUpsertA', payload)
  loading.value = false
  if (r.success) {
    result.value = r.obj
    form.value.fqdn = r.obj.fqdn
    if (!form.value.tlsName) form.value.tlsName = r.obj.fqdn
    step.value = 2
  }
}

const onIssue = async () => {
  loading.value = true
  const r = await HttpUtils.post('api/cfIssueTls', {
    name: form.value.tlsName,
    fqdn: result.value.fqdn,
    email: form.value.email,
    token: form.value.token,
  })
  loading.value = false
  if (r.success) {
    ElMessage.success(i18n.global.t('tls.cf.issueSuccess'))
    emit('created', r.obj.id)
    // 立即重新加载,前端立即看到新的 TLS 卡片
    await Data().loadData(0)
    onClose()
  }
}

watch(() => props.visible, (v) => {
  if (v) {
    step.value = 0
    form.value = initialForm()
    zones.value = []
    result.value = { fqdn: '', recordId: '' }
  }
})
</script>

<style scoped>
.cf-steps { margin-bottom: 18px; }
.cf-alert { margin-bottom: 12px; }
.zone-row { display: flex; align-items: center; justify-content: space-between; gap: 8px; width: 100%; }
.form-hint { margin: 4px 0 0; font-size: 12px; color: var(--nc-text-muted); }
.mono { font-family: var(--font-mono); }
</style>
