import { createI18n } from 'vue-i18n'
import en from './en'
import zhHans from './zhcn'

// 只剩两种语言,直接全量同步加载 — 避免懒加载带来的"首屏 key 缺失"
// 警告(loadData 在 zhHans 异步 import 完成前就触发了 t())。
// 两文件加起来 ~50KB,可接受。
const stored = localStorage.getItem('locale')
const initial = stored === 'zhHans' ? 'zhHans' : 'en'

export const i18n = createI18n({
  legacy: false,
  locale: initial,
  fallbackLocale: 'en',
  messages: { en, zhHans },
})

// 兼容老调用:其它代码可能还在调 loadLocale,留个 no-op 保持 API。
export async function loadLocale(_lang: string) {
  return
}

export const locale = i18n.global.locale.value === 'zhHans' ? 'zh-cn' : 'en'

export const languages = [
  { title: 'English', value: 'en' },
  { title: '简体中文', value: 'zhHans' },
]
