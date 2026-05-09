<template>
  <div class="page-container">
    <div class="page-header">
      <h2 class="page-title">{{ $t('pages.home') }}</h2>
      <p class="page-desc">{{ $t('home.desc', 'Sing-Box runtime overview') }}</p>
    </div>

    <Main />

    <div class="home-summary">
      <div class="summary-card nc-card">
        <div class="summary-card__head">
          <h4 class="summary-card__title">{{ $t('home.online.title') }}</h4>
          <el-tag type="success" size="small" effect="plain">
            {{ onlineUsers.length }} / {{ totalUsers }}
          </el-tag>
        </div>
        <ul v-if="onlineUsers.length" class="online-list">
          <li v-for="name in onlineUsers" :key="name" class="online-row">
            <span class="online-row__dot"></span>
            <span class="online-row__name mono">{{ name }}</span>
          </li>
        </ul>
        <p v-else class="summary-empty">{{ $t('home.online.empty') }}</p>
      </div>

      <div class="summary-card nc-card">
        <div class="summary-card__head">
          <h4 class="summary-card__title">{{ $t('home.topTraffic.title') }}</h4>
          <span class="summary-card__hint">{{ $t('home.topTraffic.hint') }}</span>
        </div>
        <table v-if="topTraffic.length" class="traffic-table">
          <thead>
            <tr>
              <th>#</th>
              <th>{{ $t('client.name') }}</th>
              <th class="num">{{ $t('home.topTraffic.up') }}</th>
              <th class="num">{{ $t('home.topTraffic.down') }}</th>
              <th class="num">{{ $t('home.topTraffic.total') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(c, i) in topTraffic" :key="c.name">
              <td>{{ i + 1 }}</td>
              <td>
                <span class="mono">{{ c.name }}</span>
                <span v-if="onlineSet.has(c.name)" class="row-online-dot" :title="$t('online')"></span>
              </td>
              <td class="num mono">{{ fmtBytes(c.up) }}</td>
              <td class="num mono">{{ fmtBytes(c.down) }}</td>
              <td class="num mono total">{{ fmtBytes(c.up + c.down) }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="summary-empty">{{ $t('home.topTraffic.empty') }}</p>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import Main from '@/components/Main.vue'
import Data from '@/store/modules/data'

const onlineUsers = computed<string[]>(() => Data().onlines?.user ?? [])
const onlineSet = computed(() => new Set(onlineUsers.value))
const totalUsers = computed(() => (Data().clients ?? []).length)

interface Row { name: string; up: number; down: number }
const topTraffic = computed<Row[]>(() => {
  const list: Row[] = (Data().clients ?? [])
    .map((c: any) => ({
      name: c.name,
      up: Number(c.up || 0) + Number(c.totalUp || 0),
      down: Number(c.down || 0) + Number(c.totalDown || 0),
    }))
    .filter((r: Row) => r.up + r.down > 0)
    .sort((a: Row, b: Row) => (b.up + b.down) - (a.up + a.down))
  return list.slice(0, 5)
})

const fmtBytes = (n: number): string => {
  if (!n || n < 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let u = 0
  let v = n
  while (v >= 1024 && u < units.length - 1) { v /= 1024; u++ }
  return v.toFixed(v >= 10 || u === 0 ? 0 : 1) + ' ' + units[u]
}
</script>

<style scoped>
.home-summary {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.4fr);
  gap: 16px;
  margin-top: 16px;
}
@media (max-width: 880px) {
  .home-summary { grid-template-columns: 1fr; }
}

.summary-card {
  padding: 14px 16px 12px;
}
.summary-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}
.summary-card__title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--nc-text-1);
}
.summary-card__hint {
  font-size: 11.5px;
  color: var(--nc-text-muted);
}

.online-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 6px 12px;
}
.online-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.online-row__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--nc-success, #16a34a);
  box-shadow: 0 0 0 3px rgba(22, 163, 74, 0.18);
  flex-shrink: 0;
}
.online-row__name {
  color: var(--nc-text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.summary-empty {
  margin: 8px 0 4px;
  font-size: 12.5px;
  color: var(--nc-text-muted);
}

.traffic-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.traffic-table thead th {
  text-align: left;
  font-weight: 500;
  color: var(--nc-text-muted);
  padding: 6px 6px;
  border-bottom: 1px solid var(--nc-border-soft);
}
.traffic-table thead th.num,
.traffic-table tbody td.num { text-align: right; }
.traffic-table tbody td {
  padding: 6px;
  border-bottom: 1px solid var(--nc-border-soft);
}
.traffic-table tbody tr:last-child td { border-bottom: none; }
.traffic-table .total { color: var(--nc-text-1); font-weight: 600; }

.row-online-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--nc-success, #16a34a);
  margin-left: 6px;
  vertical-align: middle;
}

.mono { font-family: var(--font-mono); }
</style>
