<template>
  <div class="page-container">
    <div class="page-header with-actions">
      <div class="page-header-text">
        <h2 class="page-title">{{ $t('pages.dns') }}</h2>
        <p class="page-desc">DNS 服务器、规则与策略 — 不懂可直接点「一键推荐参数」自动套用机场最优栈;下方所有开关切换即保存生效</p>
      </div>
      <div class="page-header-actions">
        <el-button @click="applyRecommendedParams">
          <el-icon><MagicStick /></el-icon>一键推荐参数
        </el-button>
        <el-button @click="showDnsModal(-1)">
          <el-icon><Plus /></el-icon>{{ $t('dns.add') }}
        </el-button>
        <el-button @click="showDnsRuleModal(-1)">
          <el-icon><Plus /></el-icon>{{ $t('dns.rule.add') }}
        </el-button>
        <el-button type="warning" plain :loading="loading" :disabled="stateChange" @click="saveConfig">
          <el-icon><Check /></el-icon>{{ $t('actions.save') }}
        </el-button>
      </div>
    </div>

    <!-- 推荐 DNS 服务器 — 开关即用 -->
    <div class="nc-card preset-card">
      <div class="preset-head">
        <h4 class="section-title">推荐 DNS 服务器</h4>
        <span class="preset-hint">切换即自动保存并热载 sing-box</span>
      </div>
      <div class="preset-grid">
        <div v-for="p in serverPresets" :key="p.tag" class="preset-item">
          <div class="preset-item__main">
            <div class="preset-item__title">
              <span class="preset-item__icon" :style="{ background: p.color }">{{ p.iconText }}</span>
              <span class="preset-item__name">{{ p.name }}</span>
              <el-tag v-if="p.badge" size="small" :type="p.badgeType" effect="plain">{{ p.badge }}</el-tag>
            </div>
            <div class="preset-item__desc">{{ p.desc }}</div>
            <div class="preset-item__addr mono">{{ p.display }}</div>
          </div>
          <el-switch :model-value="isServerEnabled(p.tag)" @change="(v) => toggleServer(p, v)" />
        </div>
      </div>
    </div>

    <!-- 推荐 DNS 规则 — 开关即用 -->
    <div class="nc-card preset-card">
      <div class="preset-head">
        <h4 class="section-title">推荐 DNS 规则</h4>
        <span class="preset-hint">切换即自动保存并热载 sing-box</span>
      </div>
      <div class="preset-grid">
        <div v-for="p in rulePresets" :key="p.key" class="preset-item">
          <div class="preset-item__main">
            <div class="preset-item__title">
              <span class="preset-item__icon" :style="{ background: p.color }">{{ p.iconText }}</span>
              <span class="preset-item__name">{{ p.name }}</span>
              <el-tag v-if="p.badge" size="small" :type="p.badgeType" effect="plain">{{ p.badge }}</el-tag>
            </div>
            <div class="preset-item__desc">{{ p.desc }}</div>
            <div v-if="p.requires && !p.requires.every((t) => isServerEnabled(t))" class="preset-item__warn">
              ⚠ 需先启用：{{ p.requires.filter((t) => !isServerEnabled(t)).join('、') }}
            </div>
          </div>
          <el-switch :model-value="isRuleEnabled(p.key)" :disabled="p.requires && !p.requires.every((t) => isServerEnabled(t))" @change="(v) => toggleRule(p, v)" />
        </div>
      </div>
    </div>

    <!-- 基础参数 -->
    <div class="nc-card">
      <h4 class="section-title">基础参数</h4>
      <el-form label-position="top">
        <div class="form-grid">
          <el-form-item>
            <template #label>
              <span>兜底 DNS</span>
              <el-tooltip content="所有规则都没匹配上时使用的 DNS。建议设为国外 DoH，避免污染。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-select v-model="finalDns" clearable placeholder="留空使用第一个服务器">
              <el-option :label="$t('dns.firstServer')" value="" />
              <el-option v-for="t in dnsServerTags" :key="t" :label="t" :value="t" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>解析优先级</span>
              <el-tooltip content="prefer_ipv4：优先 v4，国内最稳。ipv4_only：完全不查 v6，避免落地机 v6 不通导致超时。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-select v-model="dns.strategy" clearable placeholder="推荐 prefer_ipv4">
              <el-option v-for="s in ['prefer_ipv4', 'prefer_ipv6', 'ipv4_only', 'ipv6_only']" :key="s" :label="s" :value="s" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>客户端子网（EDNS）</span>
              <el-tooltip content="把「客户端大致位置」告诉权威 DNS，让 CDN 返回更近的节点。一般留空；做流媒体解锁时填落地机所在国家 IP 段。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-input v-model="dns.client_subnet" clearable placeholder="如 1.0.1.0/24，可留空" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>缓存条数</span>
              <el-tooltip content="缓存可大幅减少 DNS 查询。商业机场推荐 4096 起步。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-input-number v-model="dns.cache_capacity" :min="0" controls-position="right" placeholder="推荐 4096" style="width: 100%" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>关闭缓存</span>
              <el-tooltip content="开启后每次都重新查询，落地机 CPU 压力大，不推荐。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-switch v-model="dns.disable_cache" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>缓存永不过期</span>
              <el-tooltip content="忽略 TTL，缓存永远有效。除非你确定上游 IP 不会变，否则别开。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-switch v-model="dns.disable_expire" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>各服务器独立缓存</span>
              <el-tooltip content="国内 DNS 与国外 DNS 缓存隔离，避免污染串味。做分流时强烈建议开启。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-switch v-model="dns.independent_cache" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>反向映射（IP→域名）</span>
              <el-tooltip content="缓存「IP 来自哪个域名」，路由按域名匹配时性能更好。商业场景建议开。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-switch v-model="dns.reverse_mapping" />
          </el-form-item>
        </div>
      </el-form>
    </div>

    <div>
      <div class="nc-divider"><span>{{ $t('dns.title') }} ({{ dns.servers?.length ?? 0 }})</span></div>
      <div v-if="!dns.servers?.length" class="empty-state">
        还没有 DNS 服务器。可在上方「推荐 DNS 服务器」打开任意开关，或点右上角「添加 DNS 服务器」手动配置。
      </div>
      <div v-else class="cards-grid">
        <div v-for="(item, index) in (dns.servers as any[])" :key="index" class="entity-card nc-card">
          <div class="entity-card__head">
            <span class="entity-card__type">{{ item.type }}</span>
            <span class="entity-card__tag">{{ item.tag }}</span>
          </div>
          <dl class="entity-card__meta">
            <div class="entity-card__row"><dt>{{ $t('dns.server') }}</dt><dd class="mono">{{ item.server ?? '—' }}</dd></div>
            <div class="entity-card__row"><dt>{{ $t('in.port') }}</dt><dd class="mono">{{ item.server_port ?? '—' }}</dd></div>
            <div class="entity-card__row">
              <dt>{{ $t('objects.tls') }}</dt>
              <dd>
                <el-tag v-if="Object.hasOwn(item, 'tls')" size="small" :type="item.tls?.enabled ? 'success' : 'info'" effect="plain">
                  {{ $t(item.tls?.enabled ? 'enable' : 'disable') }}
                </el-tag>
                <span v-else>—</span>
              </dd>
            </div>
          </dl>
          <div class="entity-card__actions">
            <el-tooltip :content="$t('actions.edit')" placement="top">
              <el-button text @click="showDnsModal(Number(index))"><el-icon><Edit /></el-icon></el-button>
            </el-tooltip>
            <el-popconfirm :title="$t('confirm')" :confirm-button-text="$t('yes')" :cancel-button-text="$t('no')" @confirm="delDns(Number(index))">
              <template #reference>
                <el-button text><el-tooltip :content="$t('actions.del')" placement="top"><el-icon><Delete /></el-icon></el-tooltip></el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </div>
    </div>

    <div>
      <div class="nc-divider"><span>{{ $t('dns.rule.title') }} ({{ dnsRules.length }})</span></div>
      <div v-if="!dnsRules.length" class="empty-state">
        还没有 DNS 规则。规则用于把不同域名分流到不同 DNS 服务器（如国内域名走阿里、国外走 Cloudflare）。可在上方「推荐 DNS 规则」启用。
      </div>
      <div v-else class="cards-grid">
        <div
          v-for="(item, index) in (dnsRules as any[])"
          :key="index"
          class="entity-card nc-card"
          draggable="true"
          @dragstart="onDragStart(Number(index))"
          @dragover.prevent
          @drop="onDrop(Number(index))"
        >
          <div class="entity-card__head">
            <span class="entity-card__type">#{{ Number(index) + 1 }}</span>
            <span class="entity-card__tag">{{ item.type ? `${$t('rule.logical')} (${item.mode})` : $t('rule.simple') }}</span>
          </div>
          <dl class="entity-card__meta">
            <div class="entity-card__row"><dt>{{ $t('admin.action') }}</dt><dd>{{ item.action }}</dd></div>
            <div class="entity-card__row"><dt>{{ $t('dns.server') }}</dt><dd>{{ item.server ?? '—' }}</dd></div>
            <div class="entity-card__row"><dt>{{ $t('pages.rules') }}</dt><dd class="mono">{{ item.rules ? item.rules.length : Object.keys(item).filter((r: string) => !actionDnsRuleKeys.includes(r)).length }}</dd></div>
            <div class="entity-card__row"><dt>{{ $t('rule.invert') }}</dt><dd>{{ $t(item.invert ? 'yes' : 'no') }}</dd></div>
          </dl>
          <div class="entity-card__actions">
            <el-tooltip :content="$t('actions.edit')" placement="top">
              <el-button text @click="showDnsRuleModal(Number(index))"><el-icon><Edit /></el-icon></el-button>
            </el-tooltip>
            <el-popconfirm :title="$t('confirm')" :confirm-button-text="$t('yes')" :cancel-button-text="$t('no')" @confirm="delDnsRule(Number(index))">
              <template #reference>
                <el-button text><el-tooltip :content="$t('actions.del')" placement="top"><el-icon><Delete /></el-icon></el-tooltip></el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </div>
    </div>

    <DnsVue
      v-model="dnsModal.visible"
      :visible="dnsModal.visible"
      :index="dnsModal.index"
      :data="dnsModal.data"
      :tsTags="tsTags"
      @close="closeDnsModal"
      @save="saveDnsModal"
    />
    <DnsRuleVue
      v-model="dnsRuleModal.visible"
      :visible="dnsRuleModal.visible"
      :index="dnsRuleModal.index"
      :data="dnsRuleModal.data"
      :clients="clients"
      :inTags="inboundTags"
      :serverTags="dnsServerTags"
      :ruleSets="ruleSets"
      @close="closeDnsRuleModal"
      @save="saveDnsRuleModal"
    />
  </div>
</template>

<script lang="ts" setup>
import Data from '@/store/modules/data'
import { computed, ref, onBeforeMount, defineAsyncComponent } from 'vue'
import { ElMessage } from 'element-plus'

const DnsVue = defineAsyncComponent(() => import('@/layouts/modals/Dns.vue'))
const DnsRuleVue = defineAsyncComponent(() => import('@/layouts/modals/DnsRule.vue'))
import { Config } from '@/types/config'
import { actionDnsRuleKeys, dnsRule } from '@/types/dns'
import { FindDiff } from '@/plugins/utils'
import { Plus, Edit, Delete, Check, MagicStick, QuestionFilled } from '@element-plus/icons-vue'

const oldConfig = ref<any>({})
const loading = ref(false)
const appConfig = computed((): Config => <Config>Data().config)

// 扫描 DNS 规则里缺失的 rule_set 依赖，返回未注册的依赖项
const scanMissingRuleSets = (): RulesetDep[] => {
  const registered = new Set<string>(
    ((appConfig.value?.route?.rule_set as any[]) ?? []).map((rs: any) => rs.tag),
  )
  const missing: RulesetDep[] = []
  for (const rule of (appConfig.value.dns?.rules as any[]) ?? []) {
    const refs: string[] = Array.isArray(rule?.rule_set) ? rule.rule_set : []
    for (const refTag of refs) {
      if (registered.has(refTag)) continue
      const preset = rulePresets.find((p) => p.ruleSets?.some((d) => d.tag === refTag))
      const dep = preset?.ruleSets?.find((d) => d.tag === refTag)
      if (dep && !missing.some((m) => m.tag === dep.tag)) missing.push(dep)
    }
  }
  return missing
}

// 检查 direct outbound 是否缺失 — sing-box 1.10+ 不再隐式提供
const isDirectOutboundMissing = (): boolean => {
  const list = (Data().outbounds as any[]) ?? []
  return !list.some((o: any) => o.tag === 'direct' && o.type === 'direct')
}

// 检查配置中是否引用了 direct(rule_set 的 download_detour 或路由规则的 outbound)
const configReferencesDirect = (): boolean => {
  const ruleSets = (appConfig.value?.route?.rule_set as any[]) ?? []
  if (ruleSets.some((rs: any) => rs.download_detour === 'direct')) return true
  const routeRules = (appConfig.value?.route?.rules as any[]) ?? []
  if (routeRules.some((r: any) => r?.outbound === 'direct')) return true
  return false
}

onBeforeMount(async () => {
  if (!appConfig.value.dns) appConfig.value.dns = { servers: [], rules: [] }
  if (!appConfig.value.dns.servers) appConfig.value.dns.servers = []
  if (!appConfig.value.dns.rules) appConfig.value.dns.rules = []

  loading.value = true
  while (Data().lastLoad === 0) await new Promise((r) => setTimeout(r, 100))

  // 全面自检与自愈 — 解决 sing-box 启动报错四大类:
  // 1) DNS 规则引用的 rule_set 没注册到 route.rule_set
  // 2) rule_set 的 download_detour='direct' 但 direct outbound 不存在
  // 3) 路由规则用 outbound='direct' 但 direct outbound 不存在
  // 4) DoH 服务器写了 domain_resolver='dns-local' 但 dns-local 服务器不存在
  const fixed: string[] = []

  if (configReferencesDirect() && isDirectOutboundMissing()) {
    await Data().save('outbounds', 'new', { type: 'direct', tag: 'direct' })
    fixed.push('补全 direct 出站')
  }

  // 兜底:outbounds 完全为空时 sing-box 启动会失败,自动加一个 direct
  if (((Data().outbounds as any[]) ?? []).length === 0) {
    await Data().save('outbounds', 'new', { type: 'direct', tag: 'direct' })
    fixed.push('补全空的 outbounds(至少需要一个出站)')
  }

  // 检测:是否有 DNS 服务器引用了 domain_resolver='dns-local' 但 dns-local 不存在
  const dnsServers = (appConfig.value?.dns?.servers as any[]) ?? []
  const hasDnsLocal = dnsServers.some((s: any) => s.tag === 'dns-local')
  const refsDnsLocal = dnsServers.some((s: any) => s.domain_resolver === 'dns-local')
  if (refsDnsLocal && !hasDnsLocal) {
    appConfig.value.dns!.servers!.unshift({ type: 'local', tag: 'dns-local' } as any)
    fixed.push('补全 dns-local 服务器(DoH 自身域名解析所需)')
  }

  const missing = scanMissingRuleSets()
  if (missing.length) {
    await ensureRuleSet(missing)
    fixed.push(`补全 rule_set: ${missing.map((d) => d.tag).join('、')}`)
  }

  // 清理悬空引用:dns.final / route.final / DNS 规则的 server 字段指向不存在的 tag
  const dnsServerTagSet = new Set(dnsServers.map((s: any) => s.tag).filter(Boolean))
  if (appConfig.value.dns?.final && !dnsServerTagSet.has(appConfig.value.dns.final)) {
    fixed.push(`清除悬空 dns.final = ${appConfig.value.dns.final}`)
    appConfig.value.dns.final = undefined
  }

  // 检查每条 DNS 规则的 server 字段;引用悬空就删掉整条规则(没 server 的 route action 没意义)
  const dnsRulesArr = (appConfig.value.dns?.rules as any[]) ?? []
  const orphanRuleIdx: number[] = []
  for (let i = 0; i < dnsRulesArr.length; i++) {
    const r = dnsRulesArr[i]
    if (r?.action === 'route' && r?.server && !dnsServerTagSet.has(r.server)) {
      orphanRuleIdx.push(i)
    }
  }
  if (orphanRuleIdx.length) {
    // 倒序删除避免 index 错位
    for (let j = orphanRuleIdx.length - 1; j >= 0; j--) {
      dnsRulesArr.splice(orphanRuleIdx[j], 1)
    }
    fixed.push(`删除 ${orphanRuleIdx.length} 条引用了不存在 server 的 DNS 规则`)
  }
  const outboundTagSet = new Set(((Data().outbounds as any[]) ?? []).map((o: any) => o.tag).filter(Boolean))
  // direct 出站若被自动补,这一轮 outbounds 数据还没刷新,把 direct 也算入
  if (configReferencesDirect()) outboundTagSet.add('direct')
  if (appConfig.value.route?.final && !outboundTagSet.has(appConfig.value.route.final)) {
    fixed.push(`清除悬空 route.final = ${appConfig.value.route.final}`)
    ;(appConfig.value.route as any).final = undefined
  }

  if (fixed.length) {
    const success = await Data().save('config', 'set', appConfig.value)
    if (success) {
      ElMessage.success(`配置已自动修复:${fixed.join(';')} — sing-box 将自动恢复`)
    } else {
      ElMessage.warning(`已自动修复:${fixed.join(';')},但保存失败,请手动点保存`)
    }
  }

  oldConfig.value = JSON.parse(JSON.stringify(Data().config))
  loading.value = false
})

const tsTags = computed(() => Data().endpoints?.filter((e: any) => e.type === 'tailscale').map((e: any) => e.tag) ?? [])
const clients = computed(() => Data().clients?.map((c: any) => c.name) ?? [])
const stateChange = computed(() => FindDiff.deepCompare(appConfig.value.dns, oldConfig.value.dns))

// 共享保存:开关切换 / 一键推荐 / modal 提交 / 删除拖拽,都走它。
// 文本输入(strategy / cache_capacity / client_subnet …)频繁 keystroke,
// 仍走头部「保存」按钮,避免每个键打到后端 sing-box reload。
const autoSave = async (label?: string): Promise<boolean> => {
  loading.value = true
  // 保存前自检:确保配置自洽
  // 1) DNS 规则引用的 rule_set 必须注册到 route.rule_set
  // 2) rule_set 的 download_detour='direct' 或路由规则的 outbound='direct' 必须有对应的 direct 出站
  const missing = scanMissingRuleSets()
  if (missing.length) await ensureRuleSet(missing)
  if (configReferencesDirect() && isDirectOutboundMissing()) {
    await Data().save('outbounds', 'new', { type: 'direct', tag: 'direct' })
  }
  const success = await Data().save('config', 'set', appConfig.value)
  if (success) {
    oldConfig.value = JSON.parse(JSON.stringify(Data().config))
    if (label) ElMessage.success(label)
  } else if (label) {
    ElMessage.error('保存失败,sing-box 未重载,请检查日志')
  }
  loading.value = false
  return success
}

const saveConfig = () => autoSave()

const inboundTags = computed(() => [
  ...(Data().inbounds?.map((o: any) => o.tag) ?? []),
  ...(Data().endpoints?.filter((e: any) => e.listen_port > 0).map((e: any) => e.tag) ?? []),
])
const dns = computed((): any => appConfig.value.dns)
const dnsServerTags = computed<string[]>(() => dns.value?.servers?.filter((s: any) => s.tag).map((s: any) => s.tag) ?? [])
const finalDns = computed({
  get: () => dns.value?.final ?? '',
  set: (v: string) => { dns.value.final = v.length > 0 ? v : undefined },
})
const dnsRules = computed((): dnsRule[] => <dnsRule[]>(dns.value.rules ?? []))
const ruleSets = computed(() => appConfig.value?.route?.rule_set?.map((r: any) => r.tag) ?? [])

// ---------- 推荐 DNS 服务器（开关即用） ----------
type ServerPreset = {
  tag: string
  name: string
  desc: string
  display: string
  iconText: string
  color: string
  badge?: string
  badgeType?: 'success' | 'info' | 'warning' | 'danger'
  build: () => any
}

const serverPresets: ServerPreset[] = [
  {
    tag: 'dns-cf',
    name: 'Cloudflare DoH',
    desc: '国外域名首选 · 防污染 · 全球速度快',
    display: 'https://1.1.1.1/dns-query',
    iconText: 'CF',
    color: '#f6821f',
    badge: '推荐',
    badgeType: 'success',
    build: () => ({ type: 'https', tag: 'dns-cf', server: '1.1.1.1', server_port: 443, domain_resolver: 'dns-local' }),
  },
  {
    tag: 'dns-google',
    name: 'Google DoH',
    desc: '国外域名备用 · 与 Cloudflare 互补',
    display: 'https://8.8.8.8/dns-query',
    iconText: 'G',
    color: '#4285f4',
    build: () => ({ type: 'https', tag: 'dns-google', server: '8.8.8.8', server_port: 443, domain_resolver: 'dns-local' }),
  },
  {
    tag: 'dns-ali',
    name: '阿里 DoH',
    desc: '国内域名解析 · 国内 CDN 调度准',
    display: 'https://dns.alidns.com/dns-query',
    iconText: '阿',
    color: '#ff6a00',
    badge: '国内',
    badgeType: 'warning',
    build: () => ({ type: 'https', tag: 'dns-ali', server: 'dns.alidns.com', server_port: 443, domain_resolver: 'dns-local' }),
  },
  {
    tag: 'dns-dnspod',
    name: 'DNSPod DoH',
    desc: '国内备用 · 腾讯系',
    display: 'https://doh.pub/dns-query',
    iconText: '腾',
    color: '#00a4ff',
    badge: '国内',
    badgeType: 'warning',
    build: () => ({ type: 'https', tag: 'dns-dnspod', server: 'doh.pub', server_port: 443, domain_resolver: 'dns-local' }),
  },
  {
    tag: 'dns-local',
    name: '本地系统 DNS',
    desc: '使用落地机系统配置的解析器（一般不用单独开）',
    display: 'local',
    iconText: 'L',
    color: '#94a3b8',
    build: () => ({ type: 'local', tag: 'dns-local' }),
  },
]

const isServerEnabled = (tag: string) => dns.value.servers?.some((s: any) => s.tag === tag) ?? false

// DoH 服务器自身的域名(如 dns.alidns.com)需要被解析。预设里写了
// domain_resolver: 'dns-local',所以启用任何 DoH 时必须确保 dns-local 存在。
const ensureDnsLocal = () => {
  if (!dns.value.servers) dns.value.servers = []
  if (!dns.value.servers.some((s: any) => s.tag === 'dns-local')) {
    dns.value.servers.unshift({ type: 'local', tag: 'dns-local' })
  }
}

const toggleServer = async (p: ServerPreset, on: boolean) => {
  if (!dns.value.servers) dns.value.servers = []
  const idx = dns.value.servers.findIndex((s: any) => s.tag === p.tag)
  if (on) {
    // 启用任意 DoH 服务器前先确保 dns-local 存在(供 DoH 自身的域名解析使用)
    if (p.tag !== 'dns-local') ensureDnsLocal()
    if (idx === -1) dns.value.servers.push(p.build())
  } else {
    if (idx >= 0) dns.value.servers.splice(idx, 1)
    // 同步关闭依赖此服务器的规则(toggleRule 自身会触发 autoSave,这里
    // 用 in-place 删除,只在最末统一 autoSave 一次,免重复 reload)
    const cascadeKeys: string[] = []
    for (const rp of rulePresets) {
      if (rp.requires?.includes(p.tag) && isRuleEnabled(rp.key)) {
        const rIdx = dns.value.rules.findIndex(rp.match)
        if (rIdx >= 0) dns.value.rules.splice(rIdx, 1)
        cascadeKeys.push(rp.key)
      }
    }
    if (dns.value.final === p.tag) dns.value.final = undefined
    // 关闭 dns-local 时检查:还有别的服务器引用它吗?
    if (p.tag === 'dns-local') {
      const stillNeeded = dns.value.servers.some((s: any) => s.domain_resolver === 'dns-local')
      if (stillNeeded) {
        // 自动加回 dns-local,避免破坏 DoH 服务器
        dns.value.servers.unshift({ type: 'local', tag: 'dns-local' })
        ElMessage.warning('其它 DoH 服务器仍依赖 dns-local,已自动保留')
      }
    }
    if (cascadeKeys.length) ElMessage.info(`级联关闭依赖规则:${cascadeKeys.join('、')}`)
  }
  await autoSave(on ? `已启用 ${p.name} 并保存` : `已停用 ${p.name} 并保存`)
}

// ---------- 推荐 DNS 规则（开关即用） ----------
// rule_set 资源源 — SagerNet 维护，与 Rules.vue 模板共用
// 用 jsdelivr CDN 镜像，国内落地机也能拉到（raw.githubusercontent.com 在国内多数 IDC 不通）
const SRS_GEOSITE = 'https://cdn.jsdelivr.net/gh/SagerNet/sing-geosite@rule-set'
const SRS_GEOIP = 'https://cdn.jsdelivr.net/gh/SagerNet/sing-geoip@rule-set'

type RulesetDep = { tag: string; url: string }
type RulePreset = {
  key: string
  name: string
  desc: string
  iconText: string
  color: string
  badge?: string
  badgeType?: 'success' | 'info' | 'warning' | 'danger'
  requires?: string[]      // 依赖的 DNS 服务器 tag
  ruleSets?: RulesetDep[]  // 依赖的 route.rule_set 资源（自动联动注册）
  match: (r: any) => boolean
  build: () => any
}

const rulePresets: RulePreset[] = [
  {
    key: 'cn-to-ali',
    name: '国内域名走阿里 DNS',
    desc: '匹配 geosite-cn 时使用 dns-ali 解析（避免国外 DNS 给国内 CDN 调度错地方）',
    iconText: '🇨🇳',
    color: '#dc2626',
    badge: '商业机场推荐',
    badgeType: 'success',
    requires: ['dns-ali'],
    ruleSets: [{ tag: 'geosite-cn', url: `${SRS_GEOSITE}/geosite-cn.srs` }],
    match: (r: any) => r?.action === 'route' && r?.server === 'dns-ali' && Array.isArray(r?.rule_set) && r.rule_set.includes('geosite-cn'),
    build: () => ({ rule_set: ['geosite-cn'], action: 'route', server: 'dns-ali' }),
  },
  {
    key: 'cn-ip-to-ali',
    name: '国内 IP 段走阿里 DNS',
    desc: '匹配 geoip-cn 时使用 dns-ali 解析',
    iconText: '🌐',
    color: '#0ea5e9',
    requires: ['dns-ali'],
    ruleSets: [{ tag: 'geoip-cn', url: `${SRS_GEOIP}/geoip-cn.srs` }],
    match: (r: any) => r?.action === 'route' && r?.server === 'dns-ali' && Array.isArray(r?.rule_set) && r.rule_set.includes('geoip-cn'),
    build: () => ({ rule_set: ['geoip-cn'], action: 'route', server: 'dns-ali' }),
  },
  {
    key: 'reject-private',
    name: '拒绝解析私有地址',
    desc: '⚠ sing-box 1.13 已知冲突:开启后 rule_set 下载会被 reject 导致核心起不来。如要防 DNS 重绑定,改去路由层加 ip_cidr 黑名单。',
    iconText: '⚠',
    color: '#94a3b8',
    badge: '不推荐',
    badgeType: 'warning',
    match: (r: any) => r?.action === 'reject' && r?.ip_is_private === true,
    build: () => ({ ip_is_private: true, action: 'reject' }),
  },
  {
    key: 'block-ad',
    name: '屏蔽广告域名',
    desc: '匹配 geosite-category-ads-all 时直接拒绝',
    iconText: '🚫',
    color: '#475569',
    ruleSets: [{ tag: 'geosite-category-ads-all', url: `${SRS_GEOSITE}/geosite-category-ads-all.srs` }],
    match: (r: any) => r?.action === 'reject' && Array.isArray(r?.rule_set) && r.rule_set.includes('geosite-category-ads-all'),
    build: () => ({ rule_set: ['geosite-category-ads-all'], action: 'reject' }),
  },
]

const isRuleEnabled = (key: string) => {
  const p = rulePresets.find((x) => x.key === key)
  if (!p) return false
  return dnsRules.value.some(p.match)
}

// 确保 outbounds 里有 direct 出站(sing-box 1.10+ 不再隐式提供)
const ensureDirectOutbound = async () => {
  const existing = (Data().outbounds as any[]) ?? []
  if (existing.some((o: any) => o.tag === 'direct' && o.type === 'direct')) return
  await Data().save('outbounds', 'new', { type: 'direct', tag: 'direct' })
}

// 把规则集注册到 route.rule_set(如果还没注册)
// 注意:rule_set 用 download_detour:'direct',所以必须先确保 direct 出站存在
const ensureRuleSet = async (deps: RulesetDep[]) => {
  if (!appConfig.value.route) appConfig.value.route = {} as any
  if (!appConfig.value.route.rule_set) appConfig.value.route.rule_set = []
  await ensureDirectOutbound()
  const list = appConfig.value.route.rule_set as any[]
  for (const d of deps) {
    if (!list.some((rs: any) => rs.tag === d.tag)) {
      list.push({
        tag: d.tag,
        type: 'remote',
        format: 'binary',
        url: d.url,
        download_detour: 'direct',
        update_interval: '24h',
      })
    }
  }
}

const toggleRule = async (p: RulePreset, on: boolean) => {
  if (!dns.value.rules) dns.value.rules = []
  const idx = dns.value.rules.findIndex(p.match)
  if (on) {
    if (p.ruleSets?.length) await ensureRuleSet(p.ruleSets)
    if (idx === -1) dns.value.rules.push(p.build())
  } else {
    if (idx >= 0) dns.value.rules.splice(idx, 1)
    // rule_set 资源保留不删 — 可能被其他地方引用，由用户在「路由列表」手动清理
  }
  await autoSave(on ? `已启用规则「${p.name}」并保存` : `已停用规则「${p.name}」并保存`)
}

// ---------- 一键推荐参数 ----------
// 完整商业机场最优栈:国外 DoH(Cloudflare) + 国内 DoH(阿里) + 国内分流
// + 防 DNS 重绑定 + 客户端缓存,做完即直接 autoSave 推到 sing-box。
const applyRecommendedParams = async () => {
  loading.value = true
  if (!dns.value.servers) dns.value.servers = []
  if (!dns.value.rules) dns.value.rules = []

  // 1. 启用国外 + 国内 DoH(自动联动 dns-local 给 DoH 自身的域名做解析)
  for (const tag of ['dns-cf', 'dns-ali']) {
    const p = serverPresets.find((x) => x.tag === tag)!
    if (!isServerEnabled(tag)) {
      ensureDnsLocal()
      dns.value.servers.push(p.build())
    }
  }

  // 2. 启用核心规则:仅国内域名走阿里(避免国外 DoH 给 CDN 调度错地方)。
  //    reject-private(ip_is_private)在 sing-box 1.13 里会阻塞 rule_set
  //    下载链(geosite-cn 拉不到 → 整个 sing-box 启动失败),不放进一键推荐。
  for (const key of ['cn-to-ali']) {
    const r = rulePresets.find((x) => x.key === key)
    if (!r || isRuleEnabled(key)) continue
    if (r.ruleSets?.length) await ensureRuleSet(r.ruleSets)
    dns.value.rules.push(r.build())
  }

  // 3. 客户端参数(sing-box 1.13 schema)
  dns.value.strategy = 'prefer_ipv4'
  dns.value.cache_capacity = 4096
  dns.value.disable_cache = false
  dns.value.disable_expire = false
  dns.value.independent_cache = true
  dns.value.reverse_mapping = true
  if (!dns.value.final) dns.value.final = 'dns-cf'

  // 4. 落库 + sing-box 重载
  await autoSave('已套用机场最优 DNS 栈并自动保存:Cloudflare + 阿里 DoH · 国内域名走阿里 · 缓存 4096 · sing-box 已重载')
}

// ---------- 自定义 modal 编辑 ----------
const dnsModal = ref({ visible: false, index: -1, data: '' })
const showDnsModal = (index: number) => {
  dnsModal.value.index = index
  dnsModal.value.data = index === -1 ? '' : JSON.stringify(dns.value.servers[index])
  dnsModal.value.visible = true
}
const closeDnsModal = () => { dnsModal.value.visible = false }
const saveDnsModal = async (data: any) => {
  if (dnsModal.value.index === -1) dns.value.servers.push(data)
  else dns.value.servers[dnsModal.value.index] = data
  dnsModal.value.visible = false
  await autoSave(dnsModal.value.index === -1 ? '已新增 DNS 服务器并保存' : '已更新 DNS 服务器并保存')
}
const delDns = async (index: number) => {
  const tag = dns.value.servers[index]?.tag
  dns.value.servers.splice(index, 1)
  // final 指向被删的服务器时清空,否则 sing-box 启动失败
  if (tag && dns.value.final === tag) dns.value.final = undefined
  await autoSave('已删除 DNS 服务器并保存')
}

const dnsRuleModal = ref({ visible: false, index: -1, data: '' })
const showDnsRuleModal = (index: number) => {
  dnsRuleModal.value.index = index
  dnsRuleModal.value.data = index === -1 ? '' : JSON.stringify(dnsRules.value[index])
  dnsRuleModal.value.visible = true
}
const closeDnsRuleModal = () => { dnsRuleModal.value.visible = false }
const saveDnsRuleModal = async (data: dnsRule) => {
  if (dnsRuleModal.value.index === -1) dnsRules.value.push(data)
  else dnsRules.value[dnsRuleModal.value.index] = data
  dnsRuleModal.value.visible = false
  await autoSave(dnsRuleModal.value.index === -1 ? '已新增 DNS 规则并保存' : '已更新 DNS 规则并保存')
}
const delDnsRule = async (index: number) => {
  dnsRules.value.splice(index, 1)
  await autoSave('已删除 DNS 规则并保存')
}

const draggedItemIndex = ref<number | null>(null)
const onDragStart = (index: number) => { draggedItemIndex.value = index }
const onDrop = async (index: number) => {
  if (draggedItemIndex.value !== null) {
    const dragged = dnsRules.value[draggedItemIndex.value]
    dnsRules.value.splice(draggedItemIndex.value, 1)
    dnsRules.value.splice(index, 0, dragged)
    draggedItemIndex.value = null
    await autoSave('已调整规则顺序并保存')
  }
}
</script>

<style scoped>
.section-title { font-size: 12px; font-weight: 600; color: var(--nc-text-muted); text-transform: uppercase; letter-spacing: 0.06em; margin: 0; }
.form-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 6px 16px; }
.cards-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; }
.entity-card { display: flex; flex-direction: column; gap: 10px; padding: 14px 16px 10px; cursor: grab; }
.entity-card:active { cursor: grabbing; }
.entity-card__head { display: flex; align-items: center; justify-content: space-between; gap: 8px; border-bottom: 1px solid var(--nc-border-soft); padding-bottom: 8px; }
.entity-card__type { font-size: 11px; font-weight: 600; color: var(--nc-primary); background: var(--nc-primary-soft); padding: 2px 8px; border-radius: var(--radius-pill); text-transform: uppercase; letter-spacing: 0.04em; }
.entity-card__tag { font-family: var(--font-display); font-size: 13px; font-weight: 600; color: var(--nc-text-1); flex: 1; text-align: right; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.entity-card__meta { margin: 0; display: flex; flex-direction: column; gap: 4px; }
.entity-card__row { display: flex; justify-content: space-between; align-items: center; gap: 8px; font-size: 12.5px; }
.entity-card__row dt { color: var(--nc-text-muted); }
.entity-card__row dd { margin: 0; color: var(--nc-text-1); font-weight: 500; }
.entity-card__row .mono { font-family: var(--font-mono); }
.entity-card__actions { display: flex; gap: 4px; border-top: 1px solid var(--nc-border-soft); padding-top: 4px; margin: 4px -4px -4px; }
.entity-card__actions .el-button { flex: 1; min-width: 0; height: 32px; margin: 0 !important; }

/* 预设卡片 */
.preset-card { display: flex; flex-direction: column; gap: 14px; }
.preset-head { display: flex; align-items: baseline; gap: 12px; }
.preset-hint { font-size: 12px; color: var(--nc-text-muted); }
.preset-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 10px; }
.preset-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: var(--nc-surface, #fff);
  border: 1px solid var(--nc-border-soft);
  border-radius: var(--radius-md);
  transition: border-color 0.15s, box-shadow 0.15s;
}
.preset-item:hover { border-color: var(--nc-primary); box-shadow: 0 2px 8px rgba(59, 130, 246, 0.08); }
.preset-item__main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.preset-item__title { display: flex; align-items: center; gap: 8px; }
.preset-item__icon {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.preset-item__name { font-size: 13.5px; font-weight: 600; color: var(--nc-text-1); }
.preset-item__desc { font-size: 12px; color: var(--nc-text-muted); line-height: 1.5; }
.preset-item__addr { font-size: 11.5px; color: var(--nc-text-muted); font-family: var(--font-mono); margin-top: 2px; }
.preset-item__warn { font-size: 11.5px; color: #d97706; margin-top: 2px; }

.label-tip { margin-left: 4px; color: var(--nc-text-muted); cursor: help; vertical-align: -2px; }

.empty-state {
  padding: 24px;
  text-align: center;
  color: var(--nc-text-muted);
  font-size: 13px;
  background: var(--nc-surface-soft, #f8fafc);
  border: 1px dashed var(--nc-border-soft);
  border-radius: var(--radius-md);
}
</style>
