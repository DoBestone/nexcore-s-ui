<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="$emit('update:modelValue', $event)"
    @close="closeModal"
    class="constrained-dialog is-medium"
    :align-center="false"
    :title="$t('actions.' + title) + ' ' + $t('objects.tls')"
    destroy-on-close
  >
    <el-form label-position="top" class="tls-form">
      <!-- 顶部:名称 + 三个能力开关 -->
      <div class="form-grid">
        <el-form-item :label="$t('client.name')">
          <el-input v-model="config.name" placeholder="例 tls-cf-auto" />
        </el-form-item>
        <el-form-item label="SNI · server_name">
          <el-input v-model="config.server.server_name" placeholder="客户端 ClientHello 里的 SNI" />
        </el-form-item>
      </div>

      <div class="caps">
        <label class="cap-toggle">
          <el-switch v-model="hasAcme" />
          <div>
            <span class="cap-name">ACME 自动证书</span>
            <span class="cap-hint">Let's Encrypt 自动签发并续期 — DNS-01 / HTTP-01</span>
          </div>
        </label>
        <label class="cap-toggle">
          <el-switch v-model="hasReality" />
          <div>
            <span class="cap-name">Reality</span>
            <span class="cap-hint">无证书伪装 SNI 转发 — 抗 SNI 阻断 / 主动探测</span>
          </div>
        </label>
        <label class="cap-toggle">
          <el-switch v-model="hasEch" />
          <div>
            <span class="cap-name">ECH</span>
            <span class="cap-hint">加密 ClientHello,隐藏真实 SNI(实验性)</span>
          </div>
        </label>
      </div>

      <el-tabs v-model="tab" class="tls-tabs">
        <!-- 基础(版本 / ALPN / 证书) -->
        <el-tab-pane label="基础" name="basic">
          <div class="form-grid">
            <el-form-item label="最低 TLS 版本">
              <el-select v-model="config.server.min_version" clearable placeholder="默认 1.2">
                <el-option v-for="v in TLS_VERSIONS" :key="v" :label="`TLS ${v}`" :value="v" />
              </el-select>
            </el-form-item>
            <el-form-item label="最高 TLS 版本">
              <el-select v-model="config.server.max_version" clearable placeholder="默认 1.3">
                <el-option v-for="v in TLS_VERSIONS" :key="v" :label="`TLS ${v}`" :value="v" />
              </el-select>
            </el-form-item>
            <el-form-item label="ALPN(逗号分隔,按优先级)" class="form-item--full">
              <el-input
                :model-value="(config.server.alpn || []).join(',')"
                placeholder="h2,http/1.1"
                @input="(v: string) => config.server.alpn = v ? v.split(',').map((x: string) => x.trim()) : []"
              />
              <p class="form-hint">VLESS+Vision 通常空;WS / gRPC over TLS 用 <code>h2,http/1.1</code>;Hysteria2 不需要(QUIC 自带)</p>
            </el-form-item>
            <el-form-item label="cipher_suites(高级,逗号分隔)" class="form-item--full">
              <el-input
                :model-value="(config.server.cipher_suites || []).join(',')"
                placeholder="留空 = sing-box 默认"
                @input="(v: string) => config.server.cipher_suites = v ? v.split(',').map((x: string) => x.trim()) : []"
              />
            </el-form-item>
          </div>

          <div v-if="!hasAcme && !hasReality" class="form-section">
            <h4 class="form-section__title">证书(手动指定)</h4>
            <p class="form-hint">两种方式二选一:① 写绝对路径(推荐) ② 直接粘贴 PEM 内容</p>
            <div class="form-grid">
              <el-form-item label="证书文件路径">
                <el-input v-model="config.server.certificate_path" class="mono" placeholder="/etc/ssl/example.com.fullchain.pem" />
              </el-form-item>
              <el-form-item label="私钥文件路径">
                <el-input v-model="config.server.key_path" class="mono" placeholder="/etc/ssl/example.com.key" />
              </el-form-item>
            </div>
            <div class="cert-inline">
              <el-form-item label="证书内容(PEM,可多张拼接)">
                <el-input
                  :model-value="(config.server.certificate || []).join('\n')"
                  type="textarea"
                  :rows="4"
                  spellcheck="false"
                  class="mono-input"
                  placeholder="-----BEGIN CERTIFICATE-----…"
                  @input="(v: string) => config.server.certificate = v ? v.split('\n').filter(Boolean) : []"
                />
              </el-form-item>
              <el-form-item label="私钥内容(PEM)">
                <el-input
                  :model-value="(config.server.key || []).join('\n')"
                  type="textarea"
                  :rows="4"
                  spellcheck="false"
                  class="mono-input"
                  placeholder="-----BEGIN PRIVATE KEY-----…"
                  @input="(v: string) => config.server.key = v ? v.split('\n').filter(Boolean) : []"
                />
              </el-form-item>
            </div>
          </div>
        </el-tab-pane>

        <!-- ACME -->
        <el-tab-pane v-if="hasAcme" label="ACME" name="acme">
          <div class="form-grid">
            <el-form-item label="域名(逗号分隔,首个为主域)" class="form-item--full">
              <el-input
                :model-value="(config.server.acme.domain || []).join(',')"
                placeholder="api.example.com,*.example.com"
                @input="(v: string) => config.server.acme.domain = v ? v.split(',').map((x: string) => x.trim()) : []"
              />
              <p v-if="hasWildcardDomain" class="form-hint hint-info">
                ⓘ 检测到通配符域名。生成的分享链接 <code>sni</code> 字段会自动用入站 tag 替换 <code>*</code>(如
                <code>vless-15414.example.com</code>),客户端 TLS 校验会通过通配符证书匹配,导入不报黄色警告。
                链接 <code>server</code>(实际 dial 目标)仍是面板域名/IP,无需对每个子域名单独配 DNS。
              </p>
            </el-form-item>
            <el-form-item label="联系邮箱">
              <el-input v-model="config.server.acme.email" placeholder="admin@example.com" />
            </el-form-item>
            <el-form-item label="ACME 服务商">
              <el-select v-model="config.server.acme.provider" clearable placeholder="默认 letsencrypt">
                <el-option label="Let's Encrypt(默认)" value="letsencrypt" />
                <el-option label="Let's Encrypt Staging(测试)" value="letsencrypt-staging" />
                <el-option label="ZeroSSL" value="zerossl" />
                <el-option label="Buypass" value="buypass" />
              </el-select>
            </el-form-item>
            <el-form-item label="证书数据目录">
              <el-input v-model="config.server.acme.data_directory" class="mono" placeholder="自动" />
            </el-form-item>
          </div>
          <div class="form-grid">
            <el-form-item label="禁用 HTTP-01 挑战">
              <el-switch v-model="config.server.acme.disable_http_challenge" />
            </el-form-item>
            <el-form-item label="禁用 TLS-ALPN 挑战">
              <el-switch v-model="config.server.acme.disable_tls_alpn_challenge" />
            </el-form-item>
            <el-form-item label="HTTP 备用端口">
              <el-input-number v-model="config.server.acme.alternative_http_port" :min="0" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
            <el-form-item label="TLS 备用端口">
              <el-input-number v-model="config.server.acme.alternative_tls_port" :min="0" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
          </div>
          <div class="form-section">
            <h4 class="form-section__title">DNS-01 挑战(可选)</h4>
            <p class="form-hint">支持泛域名证书。Provider 取 cloudflare / aliyun / dnspod / route53 等;字段名按 sing-box DNS 文档(<code>cloudflare_api_token</code> 等)。</p>
            <el-form-item>
              <el-input
                :model-value="config.server.acme.dns01_challenge ? JSON.stringify(config.server.acme.dns01_challenge, null, 2) : ''"
                type="textarea"
                :rows="6"
                spellcheck="false"
                class="mono-input"
                placeholder='{"provider":"cloudflare","cloudflare_api_token":"..."}'
                @input="(v: string) => setDnsChallenge(v)"
              />
            </el-form-item>
          </div>
        </el-tab-pane>

        <!-- Reality -->
        <el-tab-pane v-if="hasReality" label="Reality" name="reality">
          <div class="form-grid">
            <el-form-item label="握手目标 server(被伪装的 SNI)">
              <el-input v-model="config.server.reality.handshake.server" placeholder="例 cloud.tencent.com" />
              <p class="form-hint">必须是真实可达且支持 TLS 1.3 + X25519 的网站</p>
            </el-form-item>
            <el-form-item label="握手目标端口">
              <el-input-number v-model="config.server.reality.handshake.server_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
            <el-form-item label="Private Key(服务端)">
              <el-input v-model="config.server.reality.private_key" class="mono">
                <template #append>
                  <el-button @click="genReality"><el-icon><Refresh /></el-icon>生成</el-button>
                </template>
              </el-input>
              <p v-if="lastPub" class="form-hint">
                配套 Public Key(给客户端):<code class="mono select-all">{{ lastPub }}</code>
              </p>
            </el-form-item>
            <el-form-item label="Short ID(逗号分隔,可多个)">
              <el-input
                :model-value="(config.server.reality.short_id || []).join(',')"
                placeholder="留空 = 任意 short_id"
                @input="(v: string) => config.server.reality.short_id = v ? v.split(',').map((x: string) => x.trim()) : []"
              />
            </el-form-item>
            <el-form-item label="最大时间漂移">
              <el-input v-model="config.server.reality.max_time_difference" placeholder="留空 = 不限" />
            </el-form-item>
          </div>
        </el-tab-pane>

        <!-- ECH -->
        <el-tab-pane v-if="hasEch" label="ECH" name="ech">
          <div class="form-grid">
            <el-form-item label="启用 ECH" class="form-item--full">
              <el-switch v-model="config.server.ech.enabled" />
            </el-form-item>
            <el-form-item label="ECH Key 路径">
              <el-input v-model="config.server.ech.key_path" class="mono" placeholder="/etc/sing-box/ech.key" />
            </el-form-item>
          </div>
          <el-form-item label="ECH Key 内容(PEM)">
            <el-input
              :model-value="(config.server.ech.key || []).join('\n')"
              type="textarea"
              :rows="4"
              spellcheck="false"
              class="mono-input"
              @input="(v: string) => config.server.ech.key = v ? v.split('\n').filter(Boolean) : []"
            />
          </el-form-item>
        </el-tab-pane>

        <!-- 客户端校验(出站引用此 TLS 时拷贝过去) -->
        <el-tab-pane label="客户端默认值" name="client">
          <p class="form-hint">这里的值在节点的"分享链接"生成时作为客户端默认参数(SNI / ALPN / utls 指纹)。和上方「基础」的 server 端字段相互独立。</p>
          <div class="form-grid">
            <el-form-item label="客户端 SNI(server_name)">
              <el-input v-model="config.client.server_name" placeholder="留空 = 跟 server 端 SNI" />
            </el-form-item>
            <el-form-item label="允许不安全(insecure)">
              <el-switch v-model="config.client.insecure" />
            </el-form-item>
            <el-form-item label="客户端 ALPN(逗号分隔)" class="form-item--full">
              <el-input
                :model-value="(config.client.alpn || []).join(',')"
                @input="(v: string) => config.client.alpn = v ? v.split(',').map((x: string) => x.trim()) : []"
              />
            </el-form-item>
            <el-form-item label="uTLS 指纹">
              <el-select :model-value="config.client.utls?.fingerprint" clearable placeholder="不启用" @change="(v: string) => setUtls(v)">
                <el-option v-for="fp in UTLS_FPS" :key="fp" :label="fp" :value="fp" />
              </el-select>
            </el-form-item>
          </div>
        </el-tab-pane>

        <!-- 高级:JSON -->
        <el-tab-pane label="JSON" name="json">
          <p class="form-hint">完整 server / client 字段都在这里(包含 fragment / kernel_tx / store 等不在 UI 上的字段)。改 JSON 时上面 tabs 实时同步。</p>
          <JsonEditorBlock :data="config" :rows="20" @update:data="(v: any) => (config = v)" />
        </el-tab-pane>
      </el-tabs>
    </el-form>

    <template #footer>
      <el-button @click="closeModal">{{ $t('actions.close') }}</el-button>
      <el-button type="primary" :loading="loading" @click="saveChanges">{{ $t('actions.save') }}</el-button>
    </template>
  </el-dialog>
</template>

<script lang="ts" setup>
import { ref, watch, computed } from 'vue'
import type { tls } from '@/types/tls'
import JsonEditorBlock from '@/components/JsonEditorBlock.vue'
import HttpUtils from '@/plugins/httputil'
import { Refresh } from '@element-plus/icons-vue'

const props = defineProps<{ visible: boolean; id: number; data: string }>()
const emit = defineEmits<{ close: []; save: [data: tls]; 'update:modelValue': [v: boolean] }>()

const config = ref<any>({ name: '', server: { server_name: '' }, client: {} })
const title = ref<'add' | 'edit'>('add')
const loading = ref(false)
const tab = ref('basic')
const lastPub = ref('') // 最近一次生成的 Reality public_key,展示给用户复制给客户端

const TLS_VERSIONS = ['1.0', '1.1', '1.2', '1.3']
const UTLS_FPS = ['chrome', 'firefox', 'safari', 'ios', 'android', 'edge', 'random', 'randomized']

const hasAcme = computed({
  get: () => !!config.value.server?.acme,
  set: (v: boolean) => {
    if (!config.value.server) config.value.server = {}
    if (v) config.value.server.acme = { domain: [], email: '', provider: 'letsencrypt' }
    else delete config.value.server.acme
    if (v) tab.value = 'acme'
  },
})
// 通配符提示:绑定此 TLS 的入站生成的分享链接,sni 字段会被后端
// (util/genLink.go::wildcardSniFromAcme)用 inbound.tag 替换 *。
const hasWildcardDomain = computed((): boolean =>
  ((config.value.server?.acme?.domain as string[] | undefined) || []).some((d: string) => d.startsWith('*.')),
)
const hasReality = computed({
  get: () => !!config.value.server?.reality,
  set: (v: boolean) => {
    if (!config.value.server) config.value.server = {}
    if (v) config.value.server.reality = { enabled: true, handshake: { server: '', server_port: 443 }, private_key: '', short_id: [''] }
    else delete config.value.server.reality
    if (v) tab.value = 'reality'
  },
})
const hasEch = computed({
  get: () => !!config.value.server?.ech,
  set: (v: boolean) => {
    if (!config.value.server) config.value.server = {}
    if (v) config.value.server.ech = { enabled: true }
    else delete config.value.server.ech
    if (v) tab.value = 'ech'
  },
})

const setDnsChallenge = (raw: string) => {
  if (!raw.trim()) {
    delete config.value.server.acme.dns01_challenge
    return
  }
  try {
    config.value.server.acme.dns01_challenge = JSON.parse(raw)
  } catch {
    /* 实时键入时静默,等用户写完才会被合法解析 */
  }
}

const setUtls = (v: string) => {
  if (!v) {
    delete config.value.client.utls
  } else {
    config.value.client.utls = { enabled: true, fingerprint: v }
  }
}

// 一键调后端 keypairs 接口生成 X25519 keypair,private_key 写到 server.reality,
// public_key 暂存到 lastPub 让用户复制给客户端。
const genReality = async () => {
  const r = await HttpUtils.get('api/keypairs', { k: 'reality' })
  if (r.success && r.obj) {
    const obj = r.obj
    config.value.server.reality.private_key = obj.PrivateKey || obj.private_key || ''
    lastPub.value = obj.PublicKey || obj.public_key || ''
  }
}

const updateData = (id: number) => {
  if (id > 0) {
    config.value = JSON.parse(props.data || '{}')
    if (!config.value.server) config.value.server = {}
    if (!config.value.client) config.value.client = {}
    title.value = 'edit'
  } else {
    config.value = {
      name: 'tls-' + Math.random().toString(36).slice(2, 6),
      server: { server_name: '', alpn: ['h2', 'http/1.1'], min_version: '1.2', max_version: '1.3' },
      client: { alpn: ['h2', 'http/1.1'] },
    }
    title.value = 'add'
  }
  lastPub.value = ''
  tab.value = 'basic'
}

const closeModal = () => emit('close')

const saveChanges = async () => {
  loading.value = true
  emit('save', config.value as tls)
  loading.value = false
}

watch(() => props.visible, (v) => { if (v) updateData(props.id) })
</script>

<style scoped>
.tls-form { display: flex; flex-direction: column; gap: 14px; }

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 6px 16px;
}
.form-grid :deep(.form-item--full) { grid-column: 1 / -1; }

.caps {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 8px;
}
.cap-toggle {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border: 1px solid var(--nc-border-soft);
  border-radius: var(--radius-md);
  background: #fafbfc;
  cursor: pointer;
}
.cap-toggle div { display: flex; flex-direction: column; gap: 2px; }
.cap-name { font-size: 13px; font-weight: 600; color: var(--nc-text-1); }
.cap-hint { font-size: 11.5px; color: var(--nc-text-muted); line-height: 1.4; }

.tls-tabs :deep(.el-tabs__header) { margin-bottom: 12px; }

.form-section {
  margin-top: 8px;
  padding: 12px 14px;
  background: var(--nc-bg-3);
  border: 1px solid var(--nc-border-soft);
  border-radius: var(--radius-md);
}
.form-section__title {
  margin: 0 0 8px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--nc-text-1);
}

.form-hint {
  font-size: 12px;
  color: var(--nc-text-muted);
  margin: 4px 0;
  line-height: 1.5;
}
.form-hint.hint-info {
  color: var(--nc-text-1);
  background: var(--nc-primary-soft);
  border-left: 3px solid var(--nc-primary);
  padding: 8px 10px;
  border-radius: 4px;
}
.form-hint code {
  font-family: var(--font-mono);
  font-size: 11.5px;
  background: var(--nc-bg-3);
  padding: 1px 6px;
  border-radius: 4px;
  color: var(--nc-text-1);
}

.cert-inline {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
}

.mono-input :deep(.el-textarea__inner),
.mono-input :deep(.el-input__inner) {
  font-family: var(--font-mono);
  font-size: 12px;
}

.select-all { user-select: all; }
</style>
