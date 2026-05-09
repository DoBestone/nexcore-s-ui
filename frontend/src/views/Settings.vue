<template>
  <div class="page-container">
    <div class="page-header with-actions">
      <div class="page-header-text">
        <h2 class="page-title">{{ $t('pages.settings') }}</h2>
        <p class="page-desc">{{ $t('settings.desc', '面板访问与账号设置') }}</p>
      </div>
      <div class="page-header-actions">
        <el-button
          type="primary"
          :loading="loading"
          :disabled="!stateChange"
          @click="save"
        >
          <el-icon><Check /></el-icon>{{ $t('actions.save') }}
        </el-button>
        <el-button
          type="warning"
          plain
          :loading="loading"
          :disabled="stateChange"
          @click="restartApp"
        >
          <el-icon><RefreshRight /></el-icon>{{ $t('actions.restartApp') }}
        </el-button>
      </div>
    </div>

    <div class="nc-tabs settings-tabs">
      <el-tabs v-model="tab">
        <el-tab-pane :label="$t('setting.interface')" name="t1">
          <el-form label-position="top" class="settings-form">
            <div class="settings-grid">
              <el-form-item :label="$t('setting.addr')">
                <el-input v-model="settings.webListen" />
              </el-form-item>
              <el-form-item :label="$t('setting.port')">
                <el-input v-model.number="webPort" type="number" :min="1" />
              </el-form-item>
              <el-form-item :label="$t('setting.webPath')">
                <el-input v-model="settings.webPath" />
              </el-form-item>
              <el-form-item :label="$t('setting.domain')">
                <el-input v-model="settings.webDomain" />
              </el-form-item>
              <el-form-item :label="$t('setting.sslKey')">
                <el-input v-model="settings.webKeyFile" />
              </el-form-item>
              <el-form-item :label="$t('setting.sslCert')">
                <el-input v-model="settings.webCertFile" />
              </el-form-item>
              <el-form-item :label="$t('setting.webUri')">
                <el-input v-model="settings.webURI" />
              </el-form-item>
              <el-form-item :label="`${$t('setting.sessionAge')} (${$t('date.m')})`">
                <el-input-number v-model="sessionMaxAge" :min="0" controls-position="right" style="width: 100%" />
              </el-form-item>
              <el-form-item :label="`${$t('setting.trafficAge')} (${$t('date.d')})`">
                <el-input-number v-model="trafficAge" :min="0" controls-position="right" style="width: 100%" />
              </el-form-item>
              <el-form-item :label="$t('setting.timeLoc')">
                <el-input v-model="settings.timeLocation" />
              </el-form-item>
            </div>
          </el-form>
        </el-tab-pane>

        <el-tab-pane :label="$t('setting.kernel', '内核参数')" name="t6">
          <p class="kernel-intro">
            sing-box 内核自身的运行参数,与面板/订阅无关。改完保存会下发并热重启 sing-box。
          </p>

          <el-collapse v-model="kernelActive" class="kernel-collapse">
            <!-- 日志 -->
            <el-collapse-item name="log">
              <template #title>
                <span class="kernel-section-title">日志(Log)</span>
                <span class="kernel-section-sub">sing-box 自身的运行日志,不是面板访问日志</span>
              </template>
              <div class="kernel-fields">
                <div class="kernel-field">
                  <div class="kernel-field-row">
                    <label>{{ $t('basic.log.level') }}</label>
                    <el-select v-model="kernel.log.level" clearable style="width: 220px">
                      <el-option v-for="l in logLevels" :key="l" :label="l" :value="l" />
                    </el-select>
                  </div>
                  <p class="kernel-hint">日常 <code>info</code> 即可。排查协议握手/路由问题时调到 <code>debug</code> 或 <code>trace</code>;<code>error</code> 以下只记错误,日志体积最小。</p>
                </div>
                <div class="kernel-field">
                  <div class="kernel-field-row">
                    <label>{{ $t('basic.log.output') }}</label>
                    <el-input v-model="kernel.log.output" placeholder="留空 = 标准输出" style="width: 320px" />
                  </div>
                  <p class="kernel-hint">写到指定文件的绝对路径(例 <code>/var/log/sing-box.log</code>)。留空走 stdout / journald,跟随 systemd unit。</p>
                </div>
                <div class="kernel-field kernel-field--inline">
                  <el-switch v-model="kernel.log.timestamp" />
                  <div>
                    <label>{{ $t('basic.log.timestamp') }}</label>
                    <p class="kernel-hint">每行日志前加时间戳。文件输出建议开,stdout + systemd 已有时间戳可关。</p>
                  </div>
                </div>
                <div class="kernel-field kernel-field--inline">
                  <el-switch v-model="kernel.log.disabled" />
                  <div>
                    <label>{{ $t('disable') }}</label>
                    <p class="kernel-hint">完全静默 —— 不推荐,出问题没法排查。</p>
                  </div>
                </div>
              </div>
            </el-collapse-item>

            <!-- NTP -->
            <el-collapse-item name="ntp">
              <template #title>
                <span class="kernel-section-title">NTP 校时</span>
                <span class="kernel-section-sub">仅在系统时钟不准时才需要打开</span>
              </template>
              <div class="kernel-fields">
                <div class="kernel-field kernel-field--inline">
                  <el-switch v-model="kernel.ntpEnabled" />
                  <div>
                    <label>{{ $t('enable') }}</label>
                    <p class="kernel-hint">VLESS / Trojan / Shadowsocks 时钟差超 ±90 秒会握手失败。普通 VPS 系统级 NTP 已就位时<strong>不要开</strong>,会多一个 UDP 出网。仅容器/嵌入式没系统 NTP 才开。</p>
                  </div>
                </div>
                <template v-if="kernel.ntpEnabled">
                  <div class="kernel-field">
                    <div class="kernel-field-row">
                      <label>NTP 服务器</label>
                      <el-input v-model="kernel.ntp.server" placeholder="time.apple.com" style="width: 240px" />
                    </div>
                    <p class="kernel-hint">推荐 <code>time.apple.com</code> / <code>pool.ntp.org</code> / <code>ntp.aliyun.com</code>。</p>
                  </div>
                  <div class="kernel-field">
                    <div class="kernel-field-row">
                      <label>端口</label>
                      <el-input-number v-model="kernel.ntp.server_port" :min="1" :max="65535" controls-position="right" style="width: 160px" />
                    </div>
                    <p class="kernel-hint">NTP 标准端口是 <code>123</code>,极少需要改。</p>
                  </div>
                  <div class="kernel-field">
                    <div class="kernel-field-row">
                      <label>校时间隔(分钟)</label>
                      <el-input-number v-model="kernel.ntpIntervalMin" :min="0" controls-position="right" style="width: 160px" />
                    </div>
                    <p class="kernel-hint">每隔多少分钟向服务器同步一次。<code>30</code> 分钟够用。</p>
                  </div>
                </template>
              </div>
            </el-collapse-item>

            <!-- Experimental -->
            <el-collapse-item name="exp">
              <template #title>
                <span class="kernel-section-title">实验性(Experimental)</span>
                <span class="kernel-section-sub">三个互相独立的子开关 · 默认全关</span>
              </template>

              <h4 class="kernel-sub">Cache File · DNS / 路由结果落盘</h4>
              <div class="kernel-fields">
                <div class="kernel-field kernel-field--inline">
                  <el-switch v-model="kernel.cacheEnabled" />
                  <div>
                    <label>{{ $t('enable') }}</label>
                    <p class="kernel-hint">把 DNS / FakeIP / 路由判定结果落盘,重启不丢热数据。流量小的小机器关掉省 IO,大流量节点开了启动后明显更快。</p>
                  </div>
                </div>
                <template v-if="kernel.cacheEnabled">
                  <div class="kernel-field">
                    <div class="kernel-field-row">
                      <label>缓存路径</label>
                      <el-input v-model="kernel.cache.path" placeholder="留空 = 默认工作目录下的 cache.db" style="width: 320px" />
                    </div>
                    <p class="kernel-hint">绝对路径或留空。多实例共存时务必各自指定不同文件。</p>
                  </div>
                  <div class="kernel-field kernel-field--inline">
                    <el-switch v-model="kernel.cache.store_fakeip" />
                    <div>
                      <label>持久化 FakeIP 映射</label>
                      <p class="kernel-hint">用了 FakeIP 才有意义。开了之后 FakeIP ↔ 域名 映射重启不丢,客户端连接不会断。</p>
                    </div>
                  </div>
                </template>
              </div>

              <h4 class="kernel-sub">Clash API</h4>
              <div class="kernel-fields">
                <div class="kernel-field kernel-field--inline">
                  <el-switch v-model="kernel.clashEnabled" />
                  <div>
                    <label>{{ $t('enable') }}</label>
                    <p class="kernel-hint">暴露 Clash 兼容控制端口,让 ClashX / Stash / OpenClash 这类客户端能切出站、看流量。<strong>s-ui 自己有控制台,通常不需要开</strong>;只有要把本机当 Clash 后端时才开。</p>
                  </div>
                </div>
                <template v-if="kernel.clashEnabled">
                  <div class="kernel-field">
                    <div class="kernel-field-row">
                      <label>监听</label>
                      <el-input v-model="kernel.clash.external_controller" placeholder="127.0.0.1:9090" style="width: 240px" />
                    </div>
                    <p class="kernel-hint">默认只听 <code>127.0.0.1</code>。改成 <code>0.0.0.0</code> 就对外开放了 —— 必须配 secret,否则任何人都能控制内核。</p>
                  </div>
                  <div class="kernel-field">
                    <div class="kernel-field-row">
                      <label>访问 Secret</label>
                      <el-input v-model="kernel.clash.secret" type="password" show-password placeholder="强烈建议设置" style="width: 320px" />
                    </div>
                    <p class="kernel-hint">客户端调用 API 时要带的 Bearer。监听对外时<strong>必填</strong>。</p>
                  </div>
                </template>
              </div>

              <h4 class="kernel-sub">V2Ray API · gRPC stats</h4>
              <div class="kernel-fields">
                <div class="kernel-field kernel-field--inline">
                  <el-switch v-model="kernel.v2rayEnabled" />
                  <div>
                    <label>{{ $t('enable') }}</label>
                    <p class="kernel-hint">老 V2Ray 风格的 gRPC 统计接口,给第三方流量统计 / 计费系统对接用。s-ui 内部统计走自有通道,<strong>普通用户不要开</strong>。</p>
                  </div>
                </div>
                <template v-if="kernel.v2rayEnabled">
                  <div class="kernel-field">
                    <div class="kernel-field-row">
                      <label>监听</label>
                      <el-input v-model="kernel.v2ray.listen" placeholder="127.0.0.1:8080" style="width: 240px" />
                    </div>
                    <p class="kernel-hint">同样建议只听 <code>127.0.0.1</code>。</p>
                  </div>
                  <div class="kernel-field kernel-field--inline">
                    <el-switch v-model="kernel.v2ray.stats.enabled" />
                    <div>
                      <label>启用 stats</label>
                      <p class="kernel-hint">采集流量计数。关闭时只剩裸 API。</p>
                    </div>
                  </div>
                </template>
              </div>
            </el-collapse-item>
          </el-collapse>

          <div class="kernel-save">
            <el-button type="primary" :loading="kernelSaving" :disabled="!kernelDirty" @click="saveKernel">
              <el-icon><Check /></el-icon>{{ $t('actions.save') }}
            </el-button>
            <span v-if="kernelDirty" class="kernel-dirty-hint">改动未保存</span>
          </div>
        </el-tab-pane>

        <el-tab-pane :label="$t('setting.account', '账号')" name="t5">
          <el-form
            ref="accountFormRef"
            :model="account"
            :rules="accountRules"
            label-position="top"
            class="settings-form"
          >
            <div class="settings-grid">
              <el-form-item :label="$t('setting.currentUser', '当前用户')">
                <el-input :model-value="account.currentUsername" disabled />
              </el-form-item>
              <el-form-item :label="$t('admin.oldPass')" prop="oldPass">
                <el-input v-model="account.oldPass" type="password" show-password autocomplete="current-password" />
              </el-form-item>
              <el-form-item :label="$t('admin.newUname')" prop="newUsername">
                <el-input v-model="account.newUsername" autocomplete="username" />
              </el-form-item>
              <el-form-item :label="$t('admin.newPass')" prop="newPass">
                <el-input v-model="account.newPass" type="password" show-password autocomplete="new-password" />
              </el-form-item>
            </div>
            <div>
              <el-button type="primary" :loading="accountSaving" :disabled="!accountReady" @click="saveAccount">
                <el-icon><Check /></el-icon>{{ $t('actions.save') }}
              </el-button>
            </div>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Ref, computed, inject, onMounted, ref } from 'vue'
import HttpUtils from '@/plugins/httputil'
import { FindDiff } from '@/plugins/utils'
import { i18n } from '@/locales'
import Data from '@/store/modules/data'
import { ElMessage } from 'element-plus'
import { Check, RefreshRight } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'

const tab = ref('t1')
const loading: Ref<boolean> = inject('loading') ?? ref(false)
const oldSettings = ref<any>({})

// 账号修改
const accountFormRef = ref<FormInstance>()
const accountSaving = ref(false)
const account = ref({
  id: 0,
  currentUsername: localStorage.getItem('admin_username') ?? '',
  oldPass: '',
  newUsername: '',
  newPass: '',
})
const accountReady = computed(() =>
  account.value.id > 0 &&
  account.value.oldPass.length > 0 &&
  account.value.newUsername.length > 0 &&
  account.value.newPass.length > 0,
)
const accountRules: FormRules = {
  oldPass:     [{ required: true, message: () => i18n.global.t('login.pwRules'), trigger: 'blur' }],
  newUsername: [{ required: true, message: () => i18n.global.t('login.unRules'), trigger: 'blur' }],
  newPass:     [{ required: true, message: () => i18n.global.t('login.pwRules'), trigger: 'blur' }],
}

const loadAccount = async () => {
  const msg = await HttpUtils.get('api/users')
  if (!msg.success || !Array.isArray(msg.obj)) return
  const stored = localStorage.getItem('admin_username')
  const matched = stored ? msg.obj.find((u: any) => u.username === stored) : null
  const u = matched ?? msg.obj[0]
  if (u) {
    account.value.id = u.id
    account.value.currentUsername = u.username
    account.value.newUsername = u.username
  }
}

const saveAccount = async () => {
  if (!accountFormRef.value) return
  await accountFormRef.value.validate(async (valid) => {
    if (!valid) return
    accountSaving.value = true
    const r = await HttpUtils.post('api/changePass', {
      id: account.value.id,
      oldPass: account.value.oldPass,
      newUsername: account.value.newUsername,
      newPass: account.value.newPass,
    })
    accountSaving.value = false
    if (r.success) {
      ElMessage.success(i18n.global.t('success'))
      localStorage.setItem('admin_username', account.value.newUsername)
      account.value.currentUsername = account.value.newUsername
      account.value.oldPass = ''
      account.value.newPass = ''
    }
  })
}

const settings = ref<any>({
  webListen: '',
  webDomain: '',
  webPort: '3095',
  webCertFile: '',
  webKeyFile: '',
  webPath: '/app/',
  webURI: '',
  sessionMaxAge: '0',
  trafficAge: '30',
  timeLocation: 'Asia/Tehran',
})

onMounted(async () => {
  loading.value = true
  await Promise.all([loadData(), loadAccount(), loadKernel()])
  loading.value = false
})

const loadData = async () => {
  loading.value = true
  const msg = await HttpUtils.get('api/settings')
  loading.value = false
  if (msg.success) setData(msg.obj)
}

const setData = (data: any) => {
  settings.value = data
  oldSettings.value = { ...data }
}

const save = async () => {
  loading.value = true
  const msg = await HttpUtils.post('api/save', {
    object: 'settings',
    action: 'set',
    data: JSON.stringify(settings.value),
  })
  if (msg.success) {
    ElMessage.success(`${i18n.global.t('success')}: ${i18n.global.t('actions.set')} ${i18n.global.t('pages.settings')}`)
    setData(msg.obj.settings)
  }
  loading.value = false
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

const restartApp = async () => {
  loading.value = true
  const msg = await HttpUtils.post('api/restartApp', {})
  if (msg.success) {
    let url = settings.value.webURI
    if (url !== '') {
      const isTLS = settings.value.webCertFile !== '' || settings.value.webKeyFile !== ''
      url = buildURL(settings.value.webDomain, settings.value.webPort.toString(), isTLS, settings.value.webPath)
    }
    await sleep(3000)
    window.location.replace(url)
  }
  loading.value = false
}

const buildURL = (host: string, port: string, isTLS: boolean, path: string) => {
  if (!host || host.length == 0) host = window.location.hostname
  if (!port || port.length == 0) port = window.location.port
  const protocol = isTLS ? 'https:' : 'http:'
  if (port === '' || (isTLS && port === '443') || (!isTLS && port === '80')) port = ''
  else port = `:${port}`
  return `${protocol}//${host}${port}${path}settings`
}

const webPort = computed({
  get: () => (settings.value.webPort.length > 0 ? parseInt(settings.value.webPort) : 3095),
  set: (v: number) => { settings.value.webPort = v > 0 ? v.toString() : '3095' },
})

const sessionMaxAge = computed({
  get: () => (settings.value.sessionMaxAge.length > 0 ? parseInt(settings.value.sessionMaxAge) : 0),
  set: (v: number) => { settings.value.sessionMaxAge = v > 0 ? v.toString() : '0' },
})

const trafficAge = computed({
  get: () => (settings.value.trafficAge.length > 0 ? parseInt(settings.value.trafficAge) : 0),
  set: (v: number) => { settings.value.trafficAge = v > 0 ? v.toString() : '0' },
})

const stateChange = computed(() => !FindDiff.deepCompare(settings.value, oldSettings.value))

// 内核(sing-box)参数 — Log / NTP / Experimental
const logLevels = ['trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic']
const kernelActive = ref<string[]>(['log'])
const kernelSaving = ref(false)
const kernelOriginal = ref<string>('{}')
const kernel = ref<any>({
  log: { disabled: false, level: 'info', output: '', timestamp: false },
  ntpEnabled: false,
  ntp: { server: 'time.apple.com', server_port: 123 },
  ntpIntervalMin: 30,
  cacheEnabled: false,
  cache: { path: '', store_fakeip: false },
  clashEnabled: false,
  clash: { external_controller: '127.0.0.1:9090', secret: '' },
  v2rayEnabled: false,
  v2ray: { listen: '127.0.0.1:8080', stats: { enabled: false, inbounds: [], outbounds: [], users: [] } },
})

const kernelDirty = computed(() => JSON.stringify(kernel.value) !== kernelOriginal.value)

const snapshotKernel = () => { kernelOriginal.value = JSON.stringify(kernel.value) }

const loadKernel = async () => {
  while (Data().lastLoad === 0) await new Promise((r) => setTimeout(r, 100))
  const cfg: any = Data().config ?? {}

  if (cfg.log) {
    kernel.value.log = {
      disabled: !!cfg.log.disabled,
      level: cfg.log.level || 'info',
      output: cfg.log.output || '',
      timestamp: !!cfg.log.timestamp,
    }
  }

  if (cfg.ntp?.enabled) {
    kernel.value.ntpEnabled = true
    kernel.value.ntp.server = cfg.ntp.server || 'time.apple.com'
    kernel.value.ntp.server_port = cfg.ntp.server_port || 123
    kernel.value.ntpIntervalMin = cfg.ntp.interval ? parseInt(String(cfg.ntp.interval).replace(/m$/, '')) || 30 : 30
  }

  const exp: any = cfg.experimental ?? {}
  if (exp.cache_file?.enabled) {
    kernel.value.cacheEnabled = true
    kernel.value.cache.path = exp.cache_file.path || ''
    kernel.value.cache.store_fakeip = !!exp.cache_file.store_fakeip
  }
  if (exp.clash_api) {
    kernel.value.clashEnabled = true
    kernel.value.clash.external_controller = exp.clash_api.external_controller || '127.0.0.1:9090'
    kernel.value.clash.secret = exp.clash_api.secret || ''
  }
  if (exp.v2ray_api) {
    kernel.value.v2rayEnabled = true
    kernel.value.v2ray.listen = exp.v2ray_api.listen || '127.0.0.1:8080'
    kernel.value.v2ray.stats = exp.v2ray_api.stats || { enabled: false, inbounds: [], outbounds: [], users: [] }
  }

  snapshotKernel()
}

const saveKernel = async () => {
  kernelSaving.value = true
  const cfg: any = Data().config ?? {}
  cfg.log = { ...kernel.value.log }

  if (kernel.value.ntpEnabled) {
    cfg.ntp = {
      enabled: true,
      server: kernel.value.ntp.server,
      server_port: kernel.value.ntp.server_port,
      interval: (kernel.value.ntpIntervalMin > 0 ? kernel.value.ntpIntervalMin : 30) + 'm',
    }
  } else {
    delete cfg.ntp
  }

  if (!cfg.experimental) cfg.experimental = {}
  if (kernel.value.cacheEnabled) {
    cfg.experimental.cache_file = {
      enabled: true,
      ...(kernel.value.cache.path ? { path: kernel.value.cache.path } : {}),
      store_fakeip: !!kernel.value.cache.store_fakeip,
    }
  } else {
    delete cfg.experimental.cache_file
  }
  if (kernel.value.clashEnabled) {
    cfg.experimental.clash_api = {
      external_controller: kernel.value.clash.external_controller,
      ...(kernel.value.clash.secret ? { secret: kernel.value.clash.secret } : {}),
    }
  } else {
    delete cfg.experimental.clash_api
  }
  if (kernel.value.v2rayEnabled) {
    cfg.experimental.v2ray_api = {
      listen: kernel.value.v2ray.listen,
      stats: kernel.value.v2ray.stats,
    }
  } else {
    delete cfg.experimental.v2ray_api
  }
  if (Object.keys(cfg.experimental).length === 0) delete cfg.experimental

  const ok = await Data().save('config', 'set', cfg)
  kernelSaving.value = false
  if (ok) snapshotKernel()
}
</script>

<style scoped>
.settings-tabs :deep(.el-tabs__header) {
  margin: 0;
  padding: 0 20px;
  background: #f8fafc;
  border-bottom: 1px solid var(--nc-border);
}
.settings-tabs :deep(.el-tabs__nav-wrap::after) { display: none; }
.settings-tabs :deep(.el-tabs__item) {
  height: 44px;
  font-size: 13px;
  color: var(--nc-text-muted);
}
.settings-tabs :deep(.el-tabs__item.is-active) {
  color: var(--nc-primary);
  font-weight: 600;
}
.settings-tabs :deep(.el-tabs__active-bar) {
  background-color: var(--nc-primary);
  height: 2px;
}
.settings-tabs :deep(.el-tabs__content) {
  padding: 20px;
}
.settings-tabs {
  background: #fff;
  border: 1px solid var(--nc-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.settings-form {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 8px 16px;
}

.settings-row {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  margin-bottom: 4px;
}

/* 内核 tab */
.kernel-intro {
  margin: 0 0 14px;
  font-size: 12.5px;
  color: var(--nc-text-muted);
  padding: 10px 12px;
  background: var(--nc-primary-soft);
  border-left: 3px solid var(--nc-primary);
  border-radius: var(--radius-md);
}
.kernel-collapse {
  background: #fff;
  border: 1px solid var(--nc-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}
.kernel-collapse :deep(.el-collapse-item__header) {
  padding: 0 16px;
  height: 48px;
  background: #f8fafc;
  border-bottom: 1px solid var(--nc-border-soft);
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 12px;
}
.kernel-collapse :deep(.el-collapse-item__wrap) {
  border: none;
  padding: 14px 16px 16px;
}
.kernel-section-title { font-weight: 600; color: var(--nc-text-1); }
.kernel-section-sub { font-size: 12px; color: var(--nc-text-muted); font-weight: 400; }
.kernel-sub {
  margin: 14px 0 8px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--nc-text-1);
  padding-bottom: 4px;
  border-bottom: 1px dashed var(--nc-border-soft);
}
.kernel-fields { display: flex; flex-direction: column; gap: 14px; }
.kernel-field { display: flex; flex-direction: column; gap: 4px; }
.kernel-field-row { display: flex; align-items: center; gap: 12px; }
.kernel-field-row label { font-size: 13px; font-weight: 500; color: var(--nc-text-1); min-width: 120px; }
.kernel-field--inline { flex-direction: row; align-items: flex-start; gap: 12px; }
.kernel-field--inline > div { flex: 1; }
.kernel-field--inline label { display: block; font-size: 13px; font-weight: 500; color: var(--nc-text-1); margin-bottom: 2px; }
.kernel-hint { margin: 0; font-size: 12px; color: var(--nc-text-muted); line-height: 1.55; }
.kernel-hint code {
  font-family: var(--font-mono);
  font-size: 11.5px;
  background: var(--nc-bg-3);
  padding: 1px 6px;
  border-radius: 4px;
  color: var(--nc-text-1);
}
.kernel-hint strong { color: var(--nc-text-1); font-weight: 600; }
.kernel-save {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
}
.kernel-dirty-hint {
  font-size: 12px;
  color: var(--nc-warning, #d97706);
}
</style>
