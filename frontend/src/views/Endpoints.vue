<template>
  <div class="page-container">
    <div class="page-header with-actions">
      <div class="page-header-text">
        <h2 class="page-title">{{ $t('pages.endpoints') }}</h2>
        <p class="page-desc">虚拟网卡端点 · 用于让落地机加入 Cloudflare Warp / Tailscale / 自建 WireGuard 网络（与「出站管理」不同 — 这里建的是网卡，不是简单转发）</p>
      </div>
      <div class="page-header-actions">
        <el-button :loading="warpLoading" @click="quickRegisterWarp">
          <el-icon v-if="!warpLoading"><MagicStick /></el-icon>一键 Warp
        </el-button>
        <el-button @click="addTailscaleTemplate">
          <el-icon><Connection /></el-icon>一键 Tailscale
        </el-button>
        <el-button type="primary" @click="showModal(0)">
          <el-icon><Plus /></el-icon>{{ $t('actions.add') }}
        </el-button>
      </div>
    </div>

    <!-- 用途说明卡片（只在还没配端点时显示） -->
    <div v-if="endpoints.length === 0" class="nc-card guide-card">
      <h4 class="section-title">这个页面是干啥的</h4>
      <div class="guide-grid">
        <div class="guide-item">
          <div class="guide-item__icon" style="background: #f6821f">⚡</div>
          <div class="guide-item__title">解锁流媒体 / ChatGPT</div>
          <div class="guide-item__desc">落地机 IP 被 Netflix / OpenAI 拉黑时，加一个 <b>Cloudflare Warp</b> 端点，把这些域名的流量从 Cloudflare 出去，立刻解锁。</div>
          <el-button size="small" type="primary" plain :loading="warpLoading" @click="quickRegisterWarp">配 Warp →</el-button>
        </div>
        <div class="guide-item">
          <div class="guide-item__icon" style="background: #2563eb">🌐</div>
          <div class="guide-item__title">多机房节点互联</div>
          <div class="guide-item__desc">把分散在不同 IDC 的机场节点用 <b>Tailscale</b> 组成虚拟内网，统一管理。</div>
          <el-button size="small" plain @click="addTailscaleTemplate">配 Tailscale →</el-button>
        </div>
        <div class="guide-item">
          <div class="guide-item__icon" style="background: #7c3aed">🔐</div>
          <div class="guide-item__title">连自建 WG 落地</div>
          <div class="guide-item__desc">你自己有 WireGuard 服务端，把机场服务器变成中转入口，二级跳到自家落地。</div>
          <el-button size="small" plain @click="showModal(0)">手动配 WG →</el-button>
        </div>
      </div>
      <div class="guide-tip">💡 99% 的小机场用不到这一页 — 只在「IP 被识别」「跨机房组网」时才需要。商业机场最常用：<b>一键 Warp</b> 解锁 ChatGPT/Netflix。</div>
    </div>

    <div v-else class="cards-grid">
      <div v-for="item in endpoints" :key="item.id" class="entity-card nc-card">
        <div class="entity-card__head">
          <span class="entity-card__type">{{ item.type }}</span>
          <span class="entity-card__tag">{{ item.tag }}</span>
        </div>
        <dl class="entity-card__meta">
          <div class="entity-card__row">
            <dt>{{ $t('in.addr') }}</dt>
            <dd class="mono">{{ item.address?.length > 0 ? item.address[0] : '—' }}</dd>
          </div>
          <div class="entity-card__row">
            <dt>{{ $t('in.port') }}</dt>
            <dd class="mono">{{ item.listen_port > 0 ? item.listen_port : '—' }}</dd>
          </div>
          <div class="entity-card__row">
            <dt>{{ $t('types.wg.peers') }}</dt>
            <dd class="mono">{{ item.peers?.length ?? '—' }}</dd>
          </div>
          <div class="entity-card__row">
            <dt>{{ $t('online') }}</dt>
            <dd>
              <span v-if="onlines.includes(item.tag)" class="status-pill"><span class="status-dot online"></span>{{ $t('online') }}</span>
              <span v-else>—</span>
            </dd>
          </div>
        </dl>
        <div class="entity-card__actions">
          <el-tooltip :content="$t('actions.edit')" placement="top">
            <el-button text @click="showModal(item.id)"><el-icon><Edit /></el-icon></el-button>
          </el-tooltip>
          <el-popconfirm :title="$t('confirm')" :confirm-button-text="$t('yes')" :cancel-button-text="$t('no')" @confirm="delEndpoint(item.tag)">
            <template #reference>
              <el-button text>
                <el-tooltip :content="$t('actions.del')" placement="top">
                  <el-icon><Delete /></el-icon>
                </el-tooltip>
              </el-button>
            </template>
          </el-popconfirm>
          <el-tooltip v-if="item.type == 'wireguard' && item.peers?.length > 0" :content="$t('main.qr', 'QR')" placement="top">
            <el-button text @click="showQrCode(item.id)"><el-icon><Picture /></el-icon></el-button>
          </el-tooltip>
          <el-tooltip v-if="Data().enableTraffic" :content="$t('stats.graphTitle')" placement="top">
            <el-button text @click="showStats(item.tag)"><el-icon><DataLine /></el-icon></el-button>
          </el-tooltip>
        </div>
      </div>
    </div>


    <EndpointVue
      v-model="modal.visible"
      :visible="modal.visible"
      :id="modal.id"
      :data="modal.data"
      :tags="endpointTags"
      @close="closeModal"
    />
    <Stats v-model="stats.visible" :visible="stats.visible" :resource="stats.resource" :tag="stats.tag" @close="closeStats" />
    <WgQrCode v-model="qrcode.visible" :visible="qrcode.visible" :data="qrcode.data" @close="closeQrCode" />
  </div>
</template>

<script lang="ts" setup>
import Data from '@/store/modules/data'
import { Endpoint } from '@/types/endpoints'
import { computed, defineAsyncComponent, ref } from 'vue'
import { ElMessage } from 'element-plus'

const EndpointVue = defineAsyncComponent(() => import('@/layouts/modals/Endpoint.vue'))
const Stats = defineAsyncComponent(() => import('@/layouts/modals/Stats.vue'))
const WgQrCode = defineAsyncComponent(() => import('@/layouts/modals/WgQrCode.vue'))
import { Plus, Edit, Delete, DataLine, Picture, MagicStick, Connection } from '@element-plus/icons-vue'

const endpoints = computed((): Endpoint[] => <Endpoint[]>Data().endpoints)
const endpointTags = computed((): any[] => endpoints.value?.map((o: Endpoint) => o.tag) ?? [])
const onlines = computed(() => [...(Data().onlines.inbound ?? []), ...(Data().onlines.outbound ?? [])])

const modal = ref({ visible: false, id: 0, data: '' })
const showModal = (id: number) => {
  modal.value.id = id
  modal.value.data = id == 0 ? '' : JSON.stringify(endpoints.value.findLast((o: any) => o.id == id))
  modal.value.visible = true
}
const closeModal = () => { modal.value.visible = false }

const stats = ref({ visible: false, resource: 'endpoint', tag: '' })
const delEndpoint = async (tag: string) => { await Data().save('endpoints', 'del', tag) }
const showStats = (tag: string) => { stats.value.tag = tag; stats.value.visible = true }
const closeStats = () => { stats.value.visible = false }

const qrcode = ref({ visible: false, data: <any>{} })
const showQrCode = (id: number) => {
  qrcode.value.data = endpoints.value.findLast((o: any) => o.id == id)
  qrcode.value.visible = true
}
const closeQrCode = () => { qrcode.value.visible = false }

// ---------- 一键 Warp（零交互全自动） ----------
// 后端 service/endpoints.go 看到 type=warp 且 act=new 时，会自动调用
// WarpService.RegisterWarp 匿名向 Cloudflare 注册账号、生成 WG 私钥、
// 拉取 IPv4/IPv6 地址、写入对端公钥。前端只需 POST 一个最小骨架。
const warpLoading = ref(false)

const quickRegisterWarp = async () => {
  if (warpLoading.value) return
  // 自动避开重名：warp / warp-2 / warp-3 ...
  let tag = 'warp'
  let n = 2
  while (endpoints.value.some((e: any) => e.tag === tag)) {
    tag = `warp-${n++}`
  }

  warpLoading.value = true
  const tip = ElMessage({
    type: 'info',
    duration: 0,
    showClose: false,
    message: `正在向 Cloudflare 注册 Warp 账号…标签 ${tag}`,
  })
  try {
    const ok = await Data().save('endpoints', 'new', {
      type: 'warp',
      tag,
      mtu: 1408,
      address: [],
      private_key: '',
      listen_port: 0,
      peers: [{ address: '', port: 0, public_key: '' }],
    })
    if (ok) {
      ElMessage.success(`Warp 注册成功！端点 "${tag}" 已自动配好 — 可在「路由列表」把 ChatGPT/Claude/Netflix 流量导到此端点`)
    } else {
      ElMessage.error('Warp 注册失败 — 检查落地机能否访问 api.cloudflareclient.com（部分 IDC 被墙不通）')
    }
  } finally {
    tip.close()
    warpLoading.value = false
  }
}

// ---------- 一键 Tailscale ----------
const addTailscaleTemplate = () => {
  const tag = `ts-${Math.random().toString(36).slice(2, 6)}`
  modal.value.id = 0
  modal.value.data = JSON.stringify({
    type: 'tailscale',
    tag,
    domain_resolver: 'local',
    auth_key: '',
    hostname: '',
    accept_routes: true,
  })
  modal.value.visible = true
  ElMessage.info('已生成 Tailscale 模板，请在弹窗中填入 auth_key 后保存')
}
</script>

<style scoped>
.cards-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; }
.entity-card { display: flex; flex-direction: column; gap: 10px; padding: 14px 16px 10px; }
.entity-card__head { display: flex; align-items: center; justify-content: space-between; gap: 8px; border-bottom: 1px solid var(--nc-border-soft); padding-bottom: 8px; }
.entity-card__type { font-size: 11px; font-weight: 600; color: var(--nc-primary); background: var(--nc-primary-soft); padding: 2px 8px; border-radius: var(--radius-pill); text-transform: uppercase; letter-spacing: 0.04em; }
.entity-card__tag { font-family: var(--font-display); font-size: 14px; font-weight: 600; color: var(--nc-text-1); flex: 1; text-align: right; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.entity-card__meta { margin: 0; display: flex; flex-direction: column; gap: 4px; }
.entity-card__row { display: flex; justify-content: space-between; align-items: center; gap: 8px; font-size: 12.5px; }
.entity-card__row dt { color: var(--nc-text-muted); }
.entity-card__row dd { margin: 0; color: var(--nc-text-1); font-weight: 500; }
.entity-card__row .mono { font-family: var(--font-mono); }
.entity-card__actions { display: flex; gap: 4px; border-top: 1px solid var(--nc-border-soft); padding-top: 4px; margin: 4px -4px -4px; }
.entity-card__actions .el-button { flex: 1; min-width: 0; height: 32px; margin: 0 !important; }
.status-pill { display: inline-flex; align-items: center; gap: 4px; font-size: 11.5px; color: var(--nc-success); font-weight: 500; }

/* 引导卡片 */
.guide-card { display: flex; flex-direction: column; gap: 16px; }
.section-title { font-size: 12px; font-weight: 600; color: var(--nc-text-muted); text-transform: uppercase; letter-spacing: 0.06em; margin: 0; }
.guide-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 12px; }
.guide-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px;
  background: var(--nc-surface-soft, #f8fafc);
  border: 1px solid var(--nc-border-soft);
  border-radius: var(--radius-md);
}
.guide-item__icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  color: #fff;
  font-size: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.guide-item__title { font-size: 14px; font-weight: 600; color: var(--nc-text-1); }
.guide-item__desc { font-size: 12.5px; color: var(--nc-text-muted); line-height: 1.55; flex: 1; }
.guide-item__desc b { color: var(--nc-text-1); font-weight: 600; }
.guide-item .el-button { align-self: flex-start; margin-top: 4px; }
.guide-tip {
  font-size: 12px;
  color: var(--nc-text-muted);
  padding: 10px 12px;
  background: var(--nc-primary-soft);
  border-radius: var(--radius-md);
  line-height: 1.6;
}
.guide-tip b { color: var(--nc-text-1); font-weight: 600; }

</style>
