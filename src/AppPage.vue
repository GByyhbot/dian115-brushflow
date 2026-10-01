<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NCard, NDataTable, NEmpty, NFormItem, NInput, NInputNumber, NSelect, NSpace, NSwitch, NTag, useMessage } from 'naive-ui'
import { Plus, Save, RefreshCw, FlaskConical } from '@lucide/vue'
import { emptyConfig, type Bridge, type Config, type Decision, type State } from './types'

const props = defineProps<{ api: Bridge; runtimeState?: State }>()
const message = useMessage()
const config = ref<Config>(emptyConfig())
const savedConfig = ref('')
const revision = ref('')
const state = ref<State>()
const selected = ref<string | null>(null)
const busy = ref(false)
const dirty = computed(() => JSON.stringify(config.value) !== savedConfig.value)
const error = ref('')
const ready = ref(false)
const decisions = ref<Decision[]>([])
const candidates = ref(JSON.stringify([
 { id: 'sample-free', title: 'Example.Movie.1080p', size_bytes: 1073741824, free: true, hr: false },
 { id: 'sample-unknown', title: 'Unknown.Promotion', size_bytes: 1073741824, free: null, hr: null }
], null, 2))
const task = computed(() => config.value.tasks.find(t => t.id === selected.value))
const options = computed(() => config.value.tasks.map(t => ({ label: t.name, value: t.id })))
const columns = [
 { title: '候选 ID', key: 'candidate_id' },
 { title: '符合规则', key: 'accepted', render: (row: Decision) => row.accepted ? '是（仅预览）' : '否' },
 { title: '判断原因', key: 'reason' }
]
async function perform(fn: () => Promise<void>) {
 if (busy.value) return
 busy.value = true; error.value = ''
 try { await fn() } catch (e) { error.value = e instanceof Error ? e.message : String(e); message.error(error.value) }
 finally { busy.value = false }
}
async function load() {
 await perform(async () => {
  const response = await props.api.getState('main')
  if (!response.state) throw new Error('宿主未返回完整状态')
  state.value = response.state
  config.value = structuredClone(response.state.config)
  revision.value = response.state.config_revision
  if (!config.value.tasks.some(t => t.id === selected.value)) selected.value = config.value.tasks[0]?.id ?? null
  savedConfig.value = JSON.stringify(config.value); ready.value = true
 })
}
function addTask() {
 const id = crypto.randomUUID()
 config.value.tasks.push({ id, name: `刷流任务 ${config.value.tasks.length + 1}`, enabled: false, site_id: '', downloader_id: '', brush_minutes: 10, check_minutes: 5,
  rules: { free_only: true, exclude_hr: true, include: '', exclude: '', min_bytes: 0, max_bytes: 0 } })
 selected.value = id
}
async function save() {
 await perform(async () => {
  const response = await props.api.invokeAction('save-config', { config: JSON.parse(JSON.stringify(config.value)), revision: revision.value })
  if (response.result?.status !== 'succeeded') throw new Error(response.result?.message || '保存失败')
  const snapshot = await props.api.refresh()
  state.value = snapshot; revision.value = snapshot.config_revision
  config.value = structuredClone(snapshot.config); savedConfig.value = JSON.stringify(config.value)
  message.success('配置已保存')
 })
}
async function preview() {
 await perform(async () => {
  const response = await props.api.invokeAction('preview', { task_id: selected.value, candidates: JSON.parse(candidates.value) })
  if (response.result?.status !== 'succeeded') throw new Error(response.result?.message || '预览失败')
  decisions.value = response.result.decisions ?? []
 })
}
onMounted(load)
</script>

<template>
 <main class="dian-plugin-page page">
  <header><div><p class="eyebrow">DIAN115 · BRUSHFLOW</p><h1>站点刷流</h1><p class="muted">任务配置与规则验证工作台</p></div><NTag type="warning">v0.1 · 仅规则预览</NTag></header>
  <NAlert type="info" :show-icon="false">初版已接入配置存储与后台心跳。站点和下载器适配器尚未接通，不会下载或删除种子。</NAlert>
  <NAlert v-if="error" type="error" title="操作失败">{{ error }}<NButton v-if="!ready" @click="load">重试加载</NButton></NAlert>
  <NSpace>
   <NButton :disabled="busy || !ready || config.tasks.length >= 50" @click="addTask"><template #icon><Plus :size="18" /></template>新建任务</NButton>
   <NButton type="primary" :loading="busy" :disabled="!ready || !dirty" @click="save"><template #icon><Save :size="18" /></template>保存配置</NButton>
   <NButton :disabled="busy || dirty" @click="load"><template #icon><RefreshCw :size="18" /></template>刷新</NButton>
   <NTag v-if="dirty" type="warning">有未保存的修改</NTag>
  </NSpace>
  <NEmpty v-if="ready && !config.tasks.length" description="暂无任务，创建任务后可验证筛选规则" />
  <div v-if="config.tasks.length" class="workspace">
   <section><NSelect v-model:value="selected" :options="options" :disabled="busy" placeholder="选择任务" />
    <NCard v-if="task" title="任务配置" class="editor">
     <fieldset :disabled="busy">
      <NFormItem label="任务名称"><NInput v-model:value="task.name" :disabled="busy" /></NFormItem>
      <div class="fields"><NFormItem label="站点引用（预留）"><NInput v-model:value="task.site_id" :disabled="busy" /></NFormItem><NFormItem label="下载器引用（预留）"><NInput v-model:value="task.downloader_id" :disabled="busy" /></NFormItem></div>
      <div class="fields"><NFormItem label="刷新周期（分钟，预留）"><NInputNumber v-model:value="task.brush_minutes" :min="1" :max="1440" :disabled="busy" /></NFormItem><NFormItem label="检查周期（分钟，预留）"><NInputNumber v-model:value="task.check_minutes" :min="1" :max="1440" :disabled="busy" /></NFormItem></div>
      <NFormItem label="包含规则（Go RE2 正则）"><NInput v-model:value="task.rules.include" :disabled="busy" /></NFormItem>
      <NFormItem label="排除规则（Go RE2 正则）"><NInput v-model:value="task.rules.exclude" :disabled="busy" /></NFormItem>
      <div class="fields"><NFormItem label="最小体积（字节）"><NInputNumber v-model:value="task.rules.min_bytes" :min="0" :max="Number.MAX_SAFE_INTEGER" :disabled="busy" /></NFormItem><NFormItem label="最大体积（字节；0 不限）"><NInputNumber v-model:value="task.rules.max_bytes" :min="0" :max="Number.MAX_SAFE_INTEGER" :disabled="busy" /></NFormItem></div>
      <NSpace vertical><label class="toggle"><NSwitch v-model:value="task.rules.free_only" :disabled="busy" />仅免费种子</label><label class="toggle"><NSwitch v-model:value="task.rules.exclude_hr" :disabled="busy" />排除 H&amp;R（未知也排除）</label></NSpace>
     </fieldset>
    </NCard>
   </section>
   <NCard title="规则预览">
    <p class="muted">输入候选 JSON（最多 100 条）。免费或 H&amp;R 信息缺失时不会默认为安全。</p>
    <NInput v-model:value="candidates" type="textarea" :rows="13" :disabled="busy" aria-label="候选种子 JSON" />
    <NButton class="preview" type="primary" :disabled="busy || dirty || !task" @click="preview"><template #icon><FlaskConical :size="18" /></template>{{ dirty ? '请先保存配置' : '验证规则' }}</NButton>
    <NDataTable v-if="decisions.length" :columns="columns" :data="decisions" :scroll-x="480" :pagination="{ pageSize: 10 }" />
   </NCard>
  </div>
  <footer>后台心跳：{{ state?.heartbeat?.at || '尚未收到' }} · 执行状态：适配器未接通</footer>
 </main>
</template>

<style scoped>
.page{display:grid;gap:22px;padding:24px;width:100%;min-width:0;box-sizing:border-box;color:var(--dian-text-primary)}
header{display:flex;align-items:center;justify-content:space-between;gap:16px;flex-wrap:wrap}h1{margin:0;font-size:28px}.eyebrow{font-size:12px;letter-spacing:2px;color:var(--dian-text-muted);margin:0 0 8px}.muted,footer{color:var(--dian-text-secondary)}
.workspace{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1.15fr);gap:24px}.workspace>*,.fields>*{min-width:0}.editor{margin-top:16px}.fields{display:grid;grid-template-columns:1fr 1fr;gap:16px}fieldset{margin:0;padding:0;border:0;min-width:0}.toggle{display:flex;gap:12px;align-items:center}.preview{margin:16px 0}footer{font-size:12px;overflow-wrap:anywhere}
@media(max-width:960px){.workspace{grid-template-columns:1fr}}@media(max-width:540px){.page{padding:14px}.fields{grid-template-columns:1fr;gap:0}}
</style>
