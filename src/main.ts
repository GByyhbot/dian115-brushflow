import { createApp, h } from 'vue'
import { NConfigProvider, NMessageProvider } from 'naive-ui'
import AppPage from './AppPage.vue'
import { emptyConfig, type Bridge, type State } from './types'
import './preview.css'

// UI-only harness: saving is in-memory, strategy previews require the real Go runtime.
let snapshot: State = { config: emptyConfig(), config_revision: '', mode: 'preview-only', live_execution: false, version: '0.1.0' }
let revision = 0
const bridge: Bridge = {
 async getState() { return { state: structuredClone(snapshot) } },
 async refresh() { return structuredClone(snapshot) },
 async invokeAction(action, input) {
  if (action === 'save-config') {
   const value = input as { config: State['config']; revision: string }
   if (value.revision !== snapshot.config_revision) return { result: { status: 'failed', message: '配置冲突' } }
   snapshot = { ...snapshot, config: structuredClone(value.config), config_revision: `preview-${++revision}` }
   return { result: { status: 'succeeded' } }
  }
  return { result: { status: 'skipped', message: '本地页面仅模拟配置保存；规则验证请安装到 dian115，或运行 Go 测试。' } }
 }
}
createApp({ render: () => h(NConfigProvider, {}, { default: () => h(NMessageProvider, {}, { default: () => h(AppPage, { api: bridge }) }) }) }).mount('#app')
