<template>
  <div class="page-container">
    <div class="page-header with-actions">
      <div class="page-header-text">
        <h2 class="page-title">{{ $t('pages.rules') }}</h2>
        <p class="page-desc">{{ $t('rules.desc', '路由规则、规则集与导入导出') }}</p>
      </div>
      <div class="page-header-actions">
        <el-button @click="applyBestPractice">
          <el-icon><MagicStick /></el-icon>一键最佳实践
        </el-button>
        <el-button type="primary" @click="showRuleModal(-1)">
          <el-icon><Plus /></el-icon>{{ $t('rule.add') }}
        </el-button>
        <el-button @click="showRulesetModal(-1)">
          <el-icon><Plus /></el-icon>{{ $t('ruleset.add') }}
        </el-button>
        <el-dropdown trigger="click">
          <el-button>
            <el-icon><MagicStick /></el-icon>{{ $t('rule.tmpl.title') }}
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="applyTemplate('block-ads')">
                <el-icon style="margin-right: 6px"><CircleClose /></el-icon>{{ $t('rule.tmpl.blockAds') }}
              </el-dropdown-item>
              <el-dropdown-item @click="applyTemplate('block-tracker')">
                <el-icon style="margin-right: 6px"><Warning /></el-icon>{{ $t('rule.tmpl.blockTracker') }}
              </el-dropdown-item>
              <el-dropdown-item @click="applyTemplate('block-porn')">
                <el-icon style="margin-right: 6px"><WarnTriangleFilled /></el-icon>{{ $t('rule.tmpl.blockPorn') }}
              </el-dropdown-item>
              <el-dropdown-item @click="applyTemplate('cn-direct')">
                <el-icon style="margin-right: 6px"><Location /></el-icon>{{ $t('rule.tmpl.cnDirect') }}
              </el-dropdown-item>
              <el-dropdown-item @click="applyTemplate('private-direct')">
                <el-icon style="margin-right: 6px"><Lock /></el-icon>{{ $t('rule.tmpl.privateDirect') }}
              </el-dropdown-item>
              <el-dropdown-item divided @click="applyTemplate('block-ads,block-tracker,private-direct,cn-direct')">
                <el-icon style="margin-right: 6px"><Star /></el-icon>{{ $t('rule.tmpl.recommended') }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-dropdown trigger="click">
          <el-button>
            <el-icon><Tools /></el-icon>{{ $t('rule.import.title', '导入') }}
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item @click="showImportRule">
                <el-icon style="margin-right: 6px"><Connection /></el-icon>{{ $t('rule.import.rulesTitle') }}
              </el-dropdown-item>
              <el-dropdown-item @click="showImportRulesets">
                <el-icon style="margin-right: 6px"><Download /></el-icon>{{ $t('rule.import.title') }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-button type="warning" plain :loading="loading" :disabled="stateChange" @click="saveConfig">
          <el-icon><Check /></el-icon>{{ $t('actions.save') }}
        </el-button>
      </div>
    </div>

    <div class="nc-card">
      <h4 class="section-title">{{ $t('basic.routing.title') }}</h4>
      <el-form label-position="top">
        <div class="form-grid">
          <el-form-item>
            <template #label>
              <span>默认出站（兜底）</span>
              <el-tooltip content="所有路由规则都没匹配上时，流量走哪个出站。一般填 direct（直连）或某个代理 outbound 的 tag。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-select v-model="route.final" clearable filterable placeholder="留空使用 sing-box 默认行为">
              <el-option v-for="t in outboundTags" :key="t" :label="t" :value="t" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>默认网卡</span>
              <el-tooltip content="出站流量绑定到哪张网卡（如 eth0、en0）。一般留空，让系统自己决定。多网卡服务器才需要指定。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-input v-model="route.default_interface" clearable placeholder="如 eth0，可留空" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>路由标记（fwmark）</span>
              <el-tooltip content="Linux 流量打标，配合 iptables / ip rule 做策略路由。0 表示不打标，普通用户保持 0 就行。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-input-number v-model="routeMark" :min="0" controls-position="right" style="width: 100%" />
          </el-form-item>
          <el-form-item>
            <template #label>
              <span>自动检测默认网卡</span>
              <el-tooltip content="开启后 sing-box 自动跟随系统默认网卡变化（笔记本切换 Wi-Fi 时也能自动跟上）。服务器场景一般不用开。" placement="top">
                <el-icon class="label-tip"><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
            <el-switch v-model="route.auto_detect_interface" />
          </el-form-item>
        </div>
      </el-form>
    </div>

    <!-- 推荐路由规则 — 开关即用 -->
    <div class="nc-card preset-card">
      <div class="preset-head">
        <h4 class="section-title">推荐路由规则</h4>
        <span class="preset-hint">打开即自动加入配置 · 关闭即移除 · 需要的规则集会自动注册</span>
      </div>
      <div class="preset-grid">
        <div v-for="p in routeRulePresets" :key="p.key" class="preset-item">
          <div class="preset-item__main">
            <div class="preset-item__title">
              <span class="preset-item__icon" :style="{ background: p.color }">{{ p.iconText }}</span>
              <span class="preset-item__name">{{ p.name }}</span>
              <el-tag v-if="p.badge" size="small" :type="p.badgeType" effect="plain">{{ p.badge }}</el-tag>
            </div>
            <div class="preset-item__desc">{{ p.desc }}</div>
          </div>
          <el-switch :model-value="isPresetEnabled(p.key)" @change="(v) => togglePreset(p, v)" />
        </div>
      </div>
    </div>

    <div>
      <div class="nc-divider"><span>{{ $t('rule.ruleset') }} ({{ rulesets.length }})</span></div>
      <div v-if="!rulesets.length" class="empty-state">
        还没有规则集。规则集是预编译的域名/IP 列表（如「全部国内域名」「广告域名」），路由规则可以直接引用整个规则集，无需手工列域名。<br />
        点右上角「<b>规则模板</b>」可一键加入常用规则集（屏蔽广告、国内直连、私有地址直连等）。
      </div>
      <div v-else class="cards-grid">
        <div v-for="(item, index) in (rulesets as any[])" :key="index" class="entity-card nc-card">
          <div class="entity-card__head">
            <span class="entity-card__type">{{ $t('ruleset.' + item.type) }}</span>
            <span class="entity-card__tag">{{ item.tag }}</span>
          </div>
          <dl class="entity-card__meta">
            <div class="entity-card__row"><dt>{{ $t('ruleset.format') }}</dt><dd class="mono">{{ item.format ?? '—' }}</dd></div>
            <div class="entity-card__row"><dt>{{ $t('objects.outbound') }}</dt><dd class="mono">{{ item.download_detour ?? '—' }}</dd></div>
            <div class="entity-card__row"><dt>{{ $t('actions.update') }}</dt><dd class="mono">{{ item.update_interval ?? '—' }}</dd></div>
            <div v-if="item.url" class="entity-card__row"><dt>来源</dt><dd class="mono ellipsis" :title="item.url">{{ shortenUrl(item.url) }}</dd></div>
          </dl>
          <div class="entity-card__actions">
            <el-tooltip :content="$t('actions.edit')" placement="top">
              <el-button text @click="showRulesetModal(Number(index))"><el-icon><Edit /></el-icon></el-button>
            </el-tooltip>
            <el-popconfirm :title="$t('confirm')" :confirm-button-text="$t('yes')" :cancel-button-text="$t('no')" @confirm="delRuleset(Number(index))">
              <template #reference>
                <el-button text>
                  <el-tooltip :content="$t('actions.del')" placement="top">
                    <el-icon><Delete /></el-icon>
                  </el-tooltip>
                </el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </div>
    </div>

    <div>
      <div class="nc-divider"><span>{{ $t('pages.rules') }} ({{ rules.length }})</span></div>
      <div v-if="!rules.length" class="empty-state">
        还没有路由规则。路由规则决定每个连接走哪个出站（如「国内域名走 direct、其他走代理」「广告域名直接 reject」）。<br />
        可以用右上角的「<b>规则模板</b>」快速添加常见规则，或点「<b>添加规则</b>」手动编辑。
      </div>
      <div v-else class="cards-grid">
        <div
          v-for="(item, index) in (rules as any[])"
          :key="index"
          class="entity-card nc-card"
          draggable="true"
          @dragstart="onDragStart(Number(index))"
          @dragover.prevent
          @drop="onDrop(Number(index))"
        >
          <div class="entity-card__head">
            <span class="entity-card__type" :class="ruleActionClass(item)">#{{ Number(index) + 1 }} · {{ ruleActionLabel(item) }}</span>
            <span class="entity-card__tag">{{ item.type ? `${$t('rule.logical')} (${item.mode})` : $t('rule.simple') }}</span>
          </div>
          <dl class="entity-card__meta">
            <div class="entity-card__row"><dt>{{ $t('admin.action') }}</dt><dd>{{ item.action ?? '—' }}</dd></div>
            <div class="entity-card__row"><dt>{{ $t('objects.outbound') }}</dt><dd class="mono">{{ item.outbound ?? '—' }}</dd></div>
            <div v-if="Array.isArray(item.rule_set) && item.rule_set.length" class="entity-card__row">
              <dt>规则集</dt>
              <dd class="mono ellipsis" :title="item.rule_set.join(', ')">{{ item.rule_set.join(', ') }}</dd>
            </div>
            <div class="entity-card__row"><dt>条件数</dt><dd class="mono">{{ item.rules ? item.rules.length : Object.keys(item).filter((r: string) => !actionKeys.includes(r)).length }}</dd></div>
            <div class="entity-card__row"><dt>{{ $t('rule.invert') }}</dt><dd>{{ $t(item.invert ? 'yes' : 'no') }}</dd></div>
          </dl>
          <div class="entity-card__actions">
            <el-tooltip :content="$t('actions.edit')" placement="top">
              <el-button text @click="showRuleModal(Number(index))"><el-icon><Edit /></el-icon></el-button>
            </el-tooltip>
            <el-popconfirm :title="$t('confirm')" :confirm-button-text="$t('yes')" :cancel-button-text="$t('no')" @confirm="delRule(Number(index))">
              <template #reference>
                <el-button text>
                  <el-tooltip :content="$t('actions.del')" placement="top">
                    <el-icon><Delete /></el-icon>
                  </el-tooltip>
                </el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </div>
    </div>

    <RuleVue
      v-model="ruleModal.visible"
      :visible="ruleModal.visible"
      :index="ruleModal.index"
      :data="ruleModal.data"
      :clients="clients"
      :inTags="inboundTags"
      :outTags="outboundTags"
      :rsTags="rulesetTags"
      @close="closeRuleModal"
      @save="saveRuleModal"
    />
    <RulesetVue
      v-model="rulesetModal.visible"
      :visible="rulesetModal.visible"
      :index="rulesetModal.index"
      :data="rulesetModal.data"
      :outTags="outboundTags"
      @close="closeRulesetModal"
      @save="saveRulesetModal"
    />
    <RuleImport
      v-model="importRulesModal.visible"
      :visible="importRulesModal.visible"
      :existingRulesCount="rules.length"
      :existingRulesetsCount="rulesets.length"
      :existingRulesetTags="rulesetTags"
      @save="saveImportRule"
      @close="closeImportRule"
    />
    <RulesetImport
      v-model="importRulesetsModal.visible"
      :visible="importRulesetsModal.visible"
      :outTags="outboundTags"
      :rsTags="rulesetTags"
      @save="saveImportRulesets"
      @close="closeImportRulesets"
    />
  </div>
</template>

<script lang="ts" setup>
import Data from '@/store/modules/data'
import { computed, ref, onBeforeMount, defineAsyncComponent } from 'vue'

const RuleVue = defineAsyncComponent(() => import('@/layouts/modals/Rule.vue'))
const RulesetVue = defineAsyncComponent(() => import('@/layouts/modals/Ruleset.vue'))
const RulesetImport = defineAsyncComponent(() => import('@/layouts/modals/RulesetImport.vue'))
const RuleImport = defineAsyncComponent(() => import('@/layouts/modals/RuleImport.vue'))
import { Config } from '@/types/config'
import { actionKeys, ruleset } from '@/types/rules'
import { FindDiff } from '@/plugins/utils'
import { Plus, Edit, Delete, Tools, Connection, Download, Check, MagicStick, CircleClose, Warning, WarnTriangleFilled, Location, Lock, Star, QuestionFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { i18n } from '@/locales'

const oldConfig = ref({})
const loading = ref(false)
const appConfig = computed((): Config => <Config>Data().config)

// 自检:配置中是否引用了 direct(rule_set 的 download_detour 或路由规则的 outbound)
const configReferencesDirect = (): boolean => {
  const cfg = Data().config as any
  const ruleSets = (cfg?.route?.rule_set as any[]) ?? []
  if (ruleSets.some((rs: any) => rs.download_detour === 'direct')) return true
  const routeRules = (cfg?.route?.rules as any[]) ?? []
  if (routeRules.some((r: any) => r?.outbound === 'direct')) return true
  return false
}

const isDirectOutboundMissing = (): boolean => {
  const list = (Data().outbounds as any[]) ?? []
  return !list.some((o: any) => o.tag === 'direct' && o.type === 'direct')
}

onBeforeMount(async () => {
  loading.value = true
  while (Data().lastLoad === 0) await new Promise((r) => setTimeout(r, 100))
  // 防御性兜底:确保 route 对象存在,避免 v-model 双向绑定丢失
  const cfg = Data().config as any
  if (!cfg.route) cfg.route = { rules: [], rule_set: [] }
  if (!cfg.route.rules) cfg.route.rules = []
  if (!cfg.route.rule_set) cfg.route.rule_set = []

  // 全面自愈 — sing-box 启动失败常见类型
  const fixed: string[] = []

  // 1) direct 出站缺失(被 rule_set download_detour 或路由规则 outbound 引用)
  if (configReferencesDirect() && isDirectOutboundMissing()) {
    await Data().save('outbounds', 'new', { type: 'direct', tag: 'direct' })
    fixed.push('补全 direct 出站')
  }

  // 2) outbounds 完全为空,sing-box 启动失败
  if (((Data().outbounds as any[]) ?? []).length === 0) {
    await Data().save('outbounds', 'new', { type: 'direct', tag: 'direct' })
    fixed.push('补全空的 outbounds(至少一个出站)')
  }

  // 3) 路由规则 outbound 引用了不存在的 outbound tag → 清空 outbound 字段(规则保留,走 final)
  const outboundTags = new Set(((Data().outbounds as any[]) ?? []).map((o: any) => o.tag).filter(Boolean))
  const endpointTagsSet = new Set(((Data().endpoints as any[]) ?? []).map((e: any) => e.tag).filter(Boolean))
  if (configReferencesDirect()) outboundTags.add('direct')
  let orphanOutboundCount = 0
  for (const r of (cfg.route.rules as any[])) {
    if (r?.outbound && !outboundTags.has(r.outbound) && !endpointTagsSet.has(r.outbound)) {
      delete r.outbound
      orphanOutboundCount++
    }
  }
  if (orphanOutboundCount > 0) fixed.push(`清除 ${orphanOutboundCount} 条规则里的悬空 outbound`)

  // 4) route.final 悬空清空
  if (cfg.route.final && !outboundTags.has(cfg.route.final) && !endpointTagsSet.has(cfg.route.final)) {
    fixed.push(`清除悬空 route.final = ${cfg.route.final}`)
    delete cfg.route.final
  }

  // 5) 路由规则的 rule_set 引用了不存在的 rule_set tag → 清空 rule_set 字段(规则保留)
  // 这里不主动补 rule_set,因为 Rules 页面不知道每个 tag 应对应什么 URL;
  // 如果规则引用 geosite-cn 等预设标签,DNS.vue 进入时会负责补全。
  const ruleSetTags = new Set((cfg.route.rule_set as any[]).map((rs: any) => rs.tag).filter(Boolean))
  let orphanRuleSetCount = 0
  for (const r of (cfg.route.rules as any[])) {
    if (Array.isArray(r?.rule_set)) {
      const filtered = r.rule_set.filter((tag: string) => ruleSetTags.has(tag))
      if (filtered.length !== r.rule_set.length) {
        if (filtered.length === 0) delete r.rule_set
        else r.rule_set = filtered
        orphanRuleSetCount++
      }
    }
  }
  if (orphanRuleSetCount > 0) fixed.push(`清除 ${orphanRuleSetCount} 条规则里的悬空 rule_set 引用`)

  if (fixed.length) {
    const ok = await Data().save('config', 'set', cfg)
    if (ok) ElMessage.success(`配置已自动修复:${fixed.join(';')} — sing-box 将自动恢复`)
    else ElMessage.warning(`已修复但保存失败:${fixed.join(';')}`)
  }

  oldConfig.value = JSON.parse(JSON.stringify(Data().config))
  loading.value = false
})

const routeMark = computed({
  get: () => route.value.default_mark ?? 0,
  set: (v: number) => {
    if (v > 0) route.value.default_mark = v
    else if (appConfig.value.route) delete (appConfig.value.route as any).default_mark
  },
})

const stateChange = computed(() => FindDiff.deepCompare(appConfig.value, oldConfig.value))

const saveConfig = async () => {
  loading.value = true
  // 保存前自检:确保 direct 出站存在(防止用户手编 JSON 后保存导致 sing-box 启动失败)
  if (configReferencesDirect() && isDirectOutboundMissing()) {
    await Data().save('outbounds', 'new', { type: 'direct', tag: 'direct' })
  }
  const success = await Data().save('config', 'set', appConfig.value)
  if (success) oldConfig.value = JSON.parse(JSON.stringify(Data().config))
  loading.value = false
}

const clients = computed(() => Data().clients?.map((c: any) => c.name) ?? [])
const route = computed((): any => appConfig.value.route ?? {})

const rules = computed(() => {
  const data = route.value
  if (!data) return []
  if (!('rules' in data) || !Array.isArray(data.rules)) data.rules = []
  return data.rules
})

const rulesets = computed(() => {
  const data = route.value
  if (!data) return []
  if (!('rule_set' in data) || !Array.isArray(data.rule_set)) data.rule_set = []
  return data.rule_set
})

const rulesetTags = computed(() => rulesets.value.map((rs: any) => rs.tag))
const outboundTags = computed(() => [
  ...(Data().outbounds?.map((o: any) => o.tag) ?? []),
  ...(Data().endpoints?.map((e: any) => e.tag) ?? []),
])
const inboundTags = computed(() => [
  ...(Data().inbounds?.map((o: any) => o.tag) ?? []),
  ...(Data().endpoints?.filter((e: any) => e.listen_port > 0).map((e: any) => e.tag) ?? []),
])

const ruleModal = ref({ visible: false, index: -1, data: '' })
const showRuleModal = (index: number) => {
  ruleModal.value.index = index
  ruleModal.value.data = index === -1 ? '' : JSON.stringify(rules.value[index])
  ruleModal.value.visible = true
}
const closeRuleModal = () => { ruleModal.value.visible = false }
const saveRuleModal = (data: any) => {
  if (ruleModal.value.index === -1) rules.value.push(data)
  else rules.value[ruleModal.value.index] = data
  ruleModal.value.visible = false
}
const delRule = (index: number) => { rules.value.splice(index, 1) }

const rulesetModal = ref({ visible: false, index: -1, data: '' })
const showRulesetModal = (index: number) => {
  rulesetModal.value.index = index
  rulesetModal.value.data = index === -1 ? '' : JSON.stringify(rulesets.value[index])
  rulesetModal.value.visible = true
}
const closeRulesetModal = () => { rulesetModal.value.visible = false }
const saveRulesetModal = (data: ruleset) => {
  if (rulesetModal.value.index === -1) rulesets.value.push(data)
  else rulesets.value[rulesetModal.value.index] = data
  rulesetModal.value.visible = false
}
const delRuleset = (index: number) => { rulesets.value.splice(index, 1) }

// ---------- 一键路由模板 ----------
// URL 必须实际存在于 sing-geosite/rule-set 分支(2026-05 校验过)。
// SagerNet/sing-geosite 是从 v2fly geosite 派生,不维护安全相关分类
// (没有 malware/phishing/cryptominers),所以这里只放真实存在的 srs。
// 每个 template 给一个 rule_set + 一条 rule。reject 类用 action=reject;
// 直连类用 outbound=direct。详情可在生成后双击编辑微调。
// 用 jsdelivr CDN 镜像 SagerNet/sing-geosite 的 rule-set 分支 — 国内可访问，
// 避免国内落地机直拉 raw.githubusercontent.com 失败导致 sing-box 启动报 rejected。
const SRS_BASE = 'https://cdn.jsdelivr.net/gh/SagerNet/sing-geosite@rule-set'

const TEMPLATES: Record<string, { tag: string; url: string; action?: string; outbound?: string }> = {
  'block-ads': {
    tag: 'tmpl-ads',
    url: `${SRS_BASE}/geosite-category-ads-all.srs`,
    action: 'reject',
  },
  'block-tracker': {
    tag: 'tmpl-tracker',
    url: `${SRS_BASE}/geosite-category-public-tracker.srs`,
    action: 'reject',
  },
  'block-porn': {
    tag: 'tmpl-porn',
    url: `${SRS_BASE}/geosite-category-porn.srs`,
    action: 'reject',
  },
  'cn-direct': {
    tag: 'tmpl-cn',
    url: `${SRS_BASE}/geosite-cn.srs`,
    outbound: 'direct',
  },
  'private-direct': {
    tag: 'tmpl-private',
    url: `${SRS_BASE}/geosite-private.srs`,
    outbound: 'direct',
  },
}

const applyTemplate = async (keysCsv: string) => {
  let added = 0
  let skipped: string[] = []
  let needsDirect = false
  const queued: { rs: any; rule: any }[] = []
  for (const key of keysCsv.split(',').map((s) => s.trim()).filter(Boolean)) {
    const t = TEMPLATES[key]
    if (!t) continue
    if (rulesets.value.some((rs: any) => rs.tag === t.tag)) {
      skipped.push(t.tag)
      continue
    }
    needsDirect = true // download_detour:'direct' + 可能的 outbound:'direct'
    const rs = {
      tag: t.tag,
      type: 'remote',
      format: 'binary',
      url: t.url,
      download_detour: 'direct',
      update_interval: '24h',
    } as any
    const rule: any = { rule_set: [t.tag] }
    if (t.action) rule.action = t.action
    else if (t.outbound) rule.outbound = t.outbound
    queued.push({ rs, rule })
  }
  // 先确保 direct 出站存在,再注入规则集和规则
  if (needsDirect) await ensureDirectOutbound()
  for (const q of queued) {
    rulesets.value.push(q.rs)
    rules.value.push(q.rule)
    added++
  }
  if (added > 0) {
    ElMessage.success(`${i18n.global.t('rule.tmpl.applied')}: +${added}`)
  } else if (skipped.length > 0) {
    ElMessage.info(`${i18n.global.t('rule.tmpl.alreadyExists')}: ${skipped.join(', ')}`)
  }
}

const draggedItemIndex = ref<number | null>(null)
const onDragStart = (index: number) => { draggedItemIndex.value = index }
const onDrop = (index: number) => {
  if (draggedItemIndex.value !== null) {
    const dragged = rules.value[draggedItemIndex.value]
    rules.value.splice(draggedItemIndex.value, 1)
    rules.value.splice(index, 0, dragged)
    draggedItemIndex.value = null
  }
}

const importRulesModal = ref({ visible: false })
const showImportRule = () => { importRulesModal.value.visible = true }
const closeImportRule = () => { importRulesModal.value.visible = false }
const saveImportRule = (block: any, mode: 'merge' | 'replace', applyFinal: boolean) => {
  if (mode === 'replace') {
    route.value.rules = block.rules ?? []
    route.value.rule_set = block.rule_set ?? []
  } else {
    const existing = new Set(rulesetTags.value)
    if (block.rules) rules.value.push(...block.rules)
    if (block.rule_set) for (const rs of block.rule_set) if (!existing.has(rs.tag)) rulesets.value.push(rs)
  }
  if (applyFinal && block.final) route.value.final = block.final
  importRulesModal.value.visible = false
}

const importRulesetsModal = ref({ visible: false })
const showImportRulesets = () => { importRulesetsModal.value.visible = true }
const closeImportRulesets = () => { importRulesetsModal.value.visible = false }
const saveImportRulesets = (items: any[]) => {
  rulesets.value.push(...items)
  importRulesetsModal.value.visible = false
}

// ---------- 推荐路由规则（开关即用） ----------
type RoutePreset = {
  key: string
  name: string
  desc: string
  iconText: string
  color: string
  badge?: string
  badgeType?: 'success' | 'info' | 'warning' | 'danger'
  ruleSets?: { tag: string; url: string }[]
  match: (r: any) => boolean
  build: () => any
}

const SRS_GS = 'https://cdn.jsdelivr.net/gh/SagerNet/sing-geosite@rule-set'
const SRS_GI = 'https://cdn.jsdelivr.net/gh/SagerNet/sing-geoip@rule-set'

const routeRulePresets: RoutePreset[] = [
  {
    key: 'sniff',
    name: '流量嗅探（识别协议）',
    desc: '识别 TLS SNI / HTTP Host，让基于域名的规则也能匹配 IP 直连请求',
    iconText: '👁',
    color: '#7c3aed',
    badge: '商业机场必开',
    badgeType: 'success',
    match: (r) => r?.action === 'sniff',
    build: () => ({ action: 'sniff' }),
  },
  {
    key: 'hijack-dns',
    name: '劫持 53 端口 DNS',
    desc: '客户端的 DNS 查询交由 sing-box 内部解析。透明代理场景必开',
    iconText: '🔌',
    color: '#d97706',
    badge: '推荐',
    badgeType: 'success',
    match: (r) => r?.action === 'hijack-dns' && (r?.port === 53 || (Array.isArray(r?.port) && r.port.includes(53))),
    build: () => ({ port: 53, action: 'hijack-dns' }),
  },
  {
    key: 'private-direct',
    name: '私有地址直连',
    desc: '192.168.x / 10.x / 172.16.x 等内网地址不走代理',
    iconText: '🏠',
    color: '#10b981',
    match: (r) => r?.outbound === 'direct' && r?.ip_is_private === true,
    build: () => ({ ip_is_private: true, outbound: 'direct' }),
  },
  {
    key: 'cn-direct',
    name: '国内域名直连',
    desc: '匹配 geosite-cn（百度/淘宝/B站等）走 direct，不浪费代理流量',
    iconText: '🇨🇳',
    color: '#dc2626',
    badge: '商业机场推荐',
    badgeType: 'success',
    ruleSets: [{ tag: 'geosite-cn', url: `${SRS_GS}/geosite-cn.srs` }],
    match: (r) => r?.outbound === 'direct' && Array.isArray(r?.rule_set) && r.rule_set.includes('geosite-cn'),
    build: () => ({ rule_set: ['geosite-cn'], outbound: 'direct' }),
  },
  {
    key: 'cn-ip-direct',
    name: '国内 IP 段直连',
    desc: '匹配 geoip-cn 走 direct，覆盖 DNS 解析失败但 IP 是国内的场景',
    iconText: '🌐',
    color: '#0ea5e9',
    ruleSets: [{ tag: 'geoip-cn', url: `${SRS_GI}/geoip-cn.srs` }],
    match: (r) => r?.outbound === 'direct' && Array.isArray(r?.rule_set) && r.rule_set.includes('geoip-cn'),
    build: () => ({ rule_set: ['geoip-cn'], outbound: 'direct' }),
  },
  {
    key: 'block-ads',
    name: '屏蔽广告域名',
    desc: '匹配 geosite-category-ads-all 直接 reject，给所有用户去广告',
    iconText: '🚫',
    color: '#475569',
    ruleSets: [{ tag: 'geosite-category-ads-all', url: `${SRS_GS}/geosite-category-ads-all.srs` }],
    match: (r) => r?.action === 'reject' && Array.isArray(r?.rule_set) && r.rule_set.includes('geosite-category-ads-all'),
    build: () => ({ rule_set: ['geosite-category-ads-all'], action: 'reject' }),
  },
  {
    key: 'block-tracker',
    name: '屏蔽追踪器',
    desc: '匹配 geosite-category-public-tracker 直接 reject',
    iconText: '⚠️',
    color: '#f59e0b',
    ruleSets: [{ tag: 'geosite-category-public-tracker', url: `${SRS_GS}/geosite-category-public-tracker.srs` }],
    match: (r) => r?.action === 'reject' && Array.isArray(r?.rule_set) && r.rule_set.includes('geosite-category-public-tracker'),
    build: () => ({ rule_set: ['geosite-category-public-tracker'], action: 'reject' }),
  },
]

const isPresetEnabled = (key: string) => {
  const p = routeRulePresets.find((x) => x.key === key)
  if (!p) return false
  return rules.value.some(p.match)
}

// 确保 outbounds 里有 direct 出站。sing-box 1.10+ 不再隐式提供 direct，
// 任何引用 outbound:'direct' 或 download_detour:'direct' 的配置都会启动失败。
// 这个函数被 rule_set 注册和 outbound:'direct' 规则共用。
const ensureDirectOutbound = async () => {
  const existing = (Data().outbounds as any[]) ?? []
  if (existing.some((o: any) => o.tag === 'direct' && o.type === 'direct')) return
  await Data().save('outbounds', 'new', { type: 'direct', tag: 'direct' })
}

const ensureRulesetRegistered = async (deps: { tag: string; url: string }[]) => {
  if (!route.value.rule_set) route.value.rule_set = []
  // download_detour 引用了 direct,所以先确保 direct 出站存在
  await ensureDirectOutbound()
  for (const d of deps) {
    if (!rulesets.value.some((rs: any) => rs.tag === d.tag)) {
      rulesets.value.push({
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

const togglePreset = async (p: RoutePreset, on: boolean) => {
  if (!route.value.rules) route.value.rules = []
  const idx = rules.value.findIndex(p.match)
  if (on) {
    // 路由规则中的 outbound:'direct' 也依赖 direct 出站存在
    const built = p.build()
    if (built?.outbound === 'direct') await ensureDirectOutbound()
    if (p.ruleSets?.length) await ensureRulesetRegistered(p.ruleSets)
    if (idx === -1) rules.value.push(built)
  } else {
    if (idx >= 0) rules.value.splice(idx, 1)
  }
}

// 一键最佳实践：商业机场默认开这套（按推荐顺序加）
const applyBestPractice = () => {
  const order = ['hijack-dns', 'sniff', 'private-direct', 'cn-direct', 'block-ads']
  let added = 0
  for (const key of order) {
    const p = routeRulePresets.find((x) => x.key === key)
    if (p && !isPresetEnabled(p.key)) {
      togglePreset(p, true)
      added++
    }
  }
  if (added > 0) {
    ElMessage.success(`已套用商业机场最佳实践 — 启用 ${added} 条规则（劫持 DNS · 嗅探 · 私有直连 · 国内直连 · 屏蔽广告）`)
  } else {
    ElMessage.info('最佳实践规则已全部启用')
  }
}

// ---------- 视觉辅助 ----------
const shortenUrl = (url: string): string => {
  try {
    const u = new URL(url)
    const path = u.pathname.split('/').filter(Boolean).slice(-2).join('/')
    return `${u.host}/.../${path}`
  } catch { return url }
}

const ruleActionLabel = (rule: any): string => {
  if (rule?.action === 'reject') return '拒绝'
  if (rule?.action === 'route' || rule?.outbound) return '路由'
  if (rule?.action === 'sniff') return '嗅探'
  if (rule?.action === 'hijack-dns') return '劫持 DNS'
  if (rule?.action === 'resolve') return '解析'
  return rule?.action ?? '路由'
}

const ruleActionClass = (rule: any): string => {
  const a = rule?.action
  if (a === 'reject') return 'tag--reject'
  if (a === 'sniff') return 'tag--sniff'
  if (a === 'hijack-dns') return 'tag--hijack'
  return ''
}
</script>

<style scoped>
.section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--nc-text-muted);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin-bottom: 12px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 6px 16px;
}

.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}

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

/* 操作类型彩色徽章 */
.tag--reject { color: #dc2626 !important; background: rgba(220, 38, 38, 0.08) !important; }
.tag--sniff { color: #7c3aed !important; background: rgba(124, 58, 237, 0.08) !important; }
.tag--hijack { color: #d97706 !important; background: rgba(217, 119, 6, 0.08) !important; }

/* 来源 / 规则集 URL 的截断 */
.ellipsis { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 180px; display: inline-block; }

/* 字段标签的 ? 提示图标 */
.label-tip { margin-left: 4px; color: var(--nc-text-muted); cursor: help; vertical-align: -2px; }

/* 空状态 */
.empty-state {
  padding: 24px;
  text-align: center;
  color: var(--nc-text-muted);
  font-size: 13px;
  line-height: 1.7;
  background: var(--nc-surface-soft, #f8fafc);
  border: 1px dashed var(--nc-border-soft);
  border-radius: var(--radius-md);
}
.empty-state b { color: var(--nc-text-1); font-weight: 600; }

/* 推荐预设卡片（与 DNS 页面统一） */
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
  font-size: 13px;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.preset-item__name { font-size: 13.5px; font-weight: 600; color: var(--nc-text-1); }
.preset-item__desc { font-size: 12px; color: var(--nc-text-muted); line-height: 1.5; }
</style>
