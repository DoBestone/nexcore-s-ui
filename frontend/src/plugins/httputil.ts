import api from './api'
import { i18n } from '@/locales'
import router from '@/router'
import { ElMessage } from 'element-plus'

export interface Msg {
  success: boolean
  msg: string
  obj: any | null
}

function _handleMsg(msg: any): void {
  if (!isMsg(msg)) {
    return
  }
  if (msg.msg) {
    if (!msg.success && msg.msg == 'Invalid login') {
      ElMessage.error(i18n.global.t('invalidLogin'))
      logout()
      return
    }
    if (msg.success) {
      ElMessage.success(i18n.global.t('success') + ': ' + i18n.global.t('actions.' + msg.msg))
    } else {
      ElMessage.error(`${i18n.global.t('failed')}: ${msg.msg}`)
    }
  }
}

export const logout = async () => {
  const response = await HttpUtils.get('api/logout')
  // 清掉前端鉴权标记 — router.beforeEach 用 localStorage('admin_username') 判
  // "曾经登录过",必须跟 cookie 失效同步清,否则跳到 /login 又被守卫认定
  // 已登录回弹到 /,出现"按登出按钮卡住不动"。
  localStorage.removeItem('admin_username')
  if (response.success) {
    router.push('/login')
  }
}

function _respToMsg(resp: any): Msg {
  const data = resp.data
  if (data == null) {
    return { success: true, msg: '', obj: null }
  } else if (isMsg(data)) {
    if (Object.prototype.hasOwnProperty.call(data, 'success')) {
      return { success: data.success, msg: data.msg, obj: data.obj || null }
    } else {
      return data
    }
  } else {
    return { success: false, msg: `unknown data: ${data}`, obj: null }
  }
}

function isMsg(obj: any): obj is Msg {
  return obj != null && typeof obj === 'object'
    && Object.hasOwn(obj, 'success') && Object.hasOwn(obj, 'msg') && Object.hasOwn(obj, 'obj')
}

const HttpUtils = {
  async get(url: string, data: object = {}, options: any = {}): Promise<Msg> {
    let msg: Msg
    try {
      const resp = await api.get(url, { params: data, ...options })
      msg = _respToMsg(resp)
    } catch (e: any) {
      msg = { success: false, msg: e.toString(), obj: null }
    }
    _handleMsg(msg)
    return msg
  },
  async post(url: string, data: object | null, options: any = undefined): Promise<Msg> {
    let msg: Msg
    try {
      const resp = await api.post(url, data, options)
      msg = _respToMsg(resp)
    } catch (e: any) {
      msg = { success: false, msg: e.toString(), obj: null }
    }
    _handleMsg(msg)
    return msg
  },
}

export default HttpUtils
