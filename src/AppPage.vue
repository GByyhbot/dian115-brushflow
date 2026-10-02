<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NAlert, NButton, NCard, NDataTable, NEmpty, NFormItem, NInput, NInputNumber, NSelect, NSpace, NSwitch, NTag, useMessage } from 'naive-ui'
import { Plus, Save, RefreshCw, FlaskConical, Play, SearchCheck, Trash2 } from '@lucide/vue'
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
const siteOptions = computed(() => config.value.sites.map(v => ({ label: v.name, value: v.id })))
const downloaderOptions = computed(() => config.value.downloaders.map(v => ({ label: v.name, value: v.id })))
const managedRows = computed(() => Object.values(state.value?.runtime?.tasks?.[selected.value || '']?.records || {}))
const columns = [
 { title: '候选 ID', key: 'candidate_id' },
 { title: '符合规则', key: 'accepted', render: (row: Decision) => row.accepted ? '是（仅预览）' : '否' },
 { title: '判断原因', key: 'reason' }
]
const managedColumns = [{ title: '标题', key: 'title' }, { title: 'Hash', key: 'hash' }, { title: '状态', key: 'status' }, { title: '删除原因', key: 'delete_reason' }]
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
  config.value.sites ||= []; config.value.downloaders ||= []; config.value.tasks ||= []
  revision.value = response.state.config_revision
  if (!config.value.tasks.some(t => t.id === selected.value)) selected.value = config.value.tasks[0]?.id ?? null
  savedConfig.value = JSON.stringify(config.value); ready.value = true
 })
}
function addTask() {
 const id = crypto.randomUUID()
 config.value.tasks.push({ id, name: `刷流任务 ${config.value.tasks.length + 1}`, enabled: false, site_id: '', downloader_id: '', brush_minutes: 10, check_minutes: 5, max_active: 3, notify: true, delete_files: true,
  rules: { free_only: true, exclude_hr: true, include: '', exclude: '', min_bytes: 0, max_bytes: 0, max_seeders: 0, max_age_minutes: 0, seed_minutes: 0, seed_ratio: 0, uploaded_bytes: 0, download_minutes: 0, delete_on_free_end: true } })
 selected.value = id
}
function addSite() { const id = `site-${crypto.randomUUID().slice(0, 8)}`; config.value.sites.push({ id, name: 'JSON Feed', feed_url: 'https://', credential_ref: '' }) }
function addDownloader() { const id = `qb-${crypto.randomUUID().slice(0, 8)}`; config.value.downloaders.push({ id, name: 'qBittorrent', base_url: 'http://', credential_ref: '', save_path: '', category: 'brushflow' }) }
function removeTask() { if (!task.value) return; config.value.tasks = config.value.tasks.filter(v => v.id !== task.value?.id); selected.value = config.value.tasks[0]?.id ?? null }
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
async function execute(action: 'run' | 'check') {
 await perform(async () => { const response = await props.api.invokeAction(action, { task_id: selected.value }); if (response.result?.status !== 'succeeded') throw new Error(response.result?.message || '执行失败'); message.success(response.result.message || '执行完成'); const snapshot = await props.api.refresh(); state.value = snapshot; revision.value = snapshot.config_revision })
}
onMounted(load)
</script>

<template>
 <main class="dian-plugin-page page">
  <header><div><p class="eyebrow">DIAN115 · BRUSHFLOW</p><h1>站点刷流</h1><p class="muted">JSON Feed + qBittorrent 自动刷流</p></div><NTag type="success">v1.0</NTag></header>
  <NAlert type="warning" :show-icon="false">删除种子属于危险操作。首次启用前先保存配置，使用“立即刷新”确认能正确识别任务标签，再配置自动删除条件。</NAlert>
  <NAlert v-if="error" type="error" title="操作失败">{{ error }}<NButton v-if="!ready" @click="load">重试加载</NButton></NAlert>
  <NSpace>
   <NButton :disabled="busy || !ready || config.tasks.length >= 50" @click="addTask"><template #icon><Plus :size="18" /></template>新建任务</NButton>
   <NButton type="primary" :loading="busy" :disabled="!ready || !dirty" @click="save"><template #icon><Save :size="18" /></template>保存配置</NButton>
   <NButton :disabled="busy || dirty" @click="load"><template #icon><RefreshCw :size="18" /></template>刷新</NButton>
   <NTag v-if="dirty" type="warning">有未保存的修改</NTag>
  </NSpace>
  <div class="adapters">
   <NCard title="站点 JSON Feed"><template #header-extra><NButton size="small" @click="addSite"><Plus :size="16"/>添加</NButton></template>
    <NEmpty v-if="!config.sites.length" description="添加返回标准候选 JSON 的站点 Feed" />
    <div v-for="(site,index) in config.sites" :key="site.id" class="adapter-row"><NInput v-model:value="site.name" placeholder="名称"/><NInput v-model:value="site.feed_url" placeholder="HTTPS Feed URL"/><NInput v-model:value="site.credential_ref" placeholder="托管凭据引用（可空）"/><NButton quaternary type="error" @click="config.sites.splice(index,1)"><Trash2 :size="16"/></NButton></div>
   </NCard>
   <NCard title="qBittorrent"><template #header-extra><NButton size="small" @click="addDownloader"><Plus :size="16"/>添加</NButton></template>
    <NEmpty v-if="!config.downloaders.length" description="添加 qBittorrent Web API" />
    <div v-for="(down,index) in config.downloaders" :key="down.id" class="adapter-row"><NInput v-model:value="down.name" placeholder="名称"/><NInput v-model:value="down.base_url" placeholder="http://qbittorrent:8080"/><NInput v-model:value="down.credential_ref" placeholder="Cookie/Authorization 凭据引用"/><NInput v-model:value="down.save_path" placeholder="保存路径（可空）"/><NInput v-model:value="down.category" placeholder="分类"/><NButton quaternary type="error" @click="config.downloaders.splice(index,1)"><Trash2 :size="16"/></NButton></div>
   </NCard>
  </div>
  <NEmpty v-if="ready && !config.tasks.length" description="暂无任务，创建任务后可验证筛选规则" />
  <div v-if="config.tasks.length" class="workspace">
   <section><NSelect v-model:value="selected" :options="options" :disabled="busy" placeholder="选择任务" />
    <NCard v-if="task" title="任务配置" class="editor">
     <fieldset :disabled="busy">
      <NFormItem label="任务名称"><NInput v-model:value="task.name" :disabled="busy" /></NFormItem>
      <div class="fields"><NFormItem label="站点"><NSelect v-model:value="task.site_id" :options="siteOptions" /></NFormItem><NFormItem label="下载器"><NSelect v-model:value="task.downloader_id" :options="downloaderOptions" /></NFormItem></div>
      <div class="fields"><NFormItem label="刷新周期（分钟）"><NInputNumber v-model:value="task.brush_minutes" :min="1" :max="1440" /></NFormItem><NFormItem label="检查周期（分钟）"><NInputNumber v-model:value="task.check_minutes" :min="1" :max="1440" /></NFormItem><NFormItem label="最多活动种子（0 不限）"><NInputNumber v-model:value="task.max_active" :min="0" :max="1000" /></NFormItem></div>
      <NFormItem label="包含规则（Go RE2 正则）"><NInput v-model:value="task.rules.include" :disabled="busy" /></NFormItem>
      <NFormItem label="排除规则（Go RE2 正则）"><NInput v-model:value="task.rules.exclude" :disabled="busy" /></NFormItem>
      <div class="fields"><NFormItem label="最小体积（字节）"><NInputNumber v-model:value="task.rules.min_bytes" :min="0" :max="Number.MAX_SAFE_INTEGER" :disabled="busy" /></NFormItem><NFormItem label="最大体积（字节；0 不限）"><NInputNumber v-model:value="task.rules.max_bytes" :min="0" :max="Number.MAX_SAFE_INTEGER" :disabled="busy" /></NFormItem></div>
      <div class="fields"><NFormItem label="最大做种人数（0 不限）"><NInputNumber v-model:value="task.rules.max_seeders" :min="0" /></NFormItem><NFormItem label="最大发布时间（分钟；0 不限）"><NInputNumber v-model:value="task.rules.max_age_minutes" :min="0" /></NFormItem></div>
      <div class="fields"><NFormItem label="做种时间后删除（分钟）"><NInputNumber v-model:value="task.rules.seed_minutes" :min="0" /></NFormItem><NFormItem label="分享率后删除"><NInputNumber v-model:value="task.rules.seed_ratio" :min="0" :step="0.1" /></NFormItem><NFormItem label="上传量后删除（字节）"><NInputNumber v-model:value="task.rules.uploaded_bytes" :min="0" /></NFormItem><NFormItem label="下载超时删除（分钟）"><NInputNumber v-model:value="task.rules.download_minutes" :min="0" /></NFormItem></div>
      <NSpace vertical><label class="toggle"><NSwitch v-model:value="task.enabled" />启用自动调度</label><label class="toggle"><NSwitch v-model:value="task.notify" />发送通知</label><label class="toggle"><NSwitch v-model:value="task.delete_files" />删种时同时删除文件</label><label class="toggle"><NSwitch v-model:value="task.rules.free_only" />仅免费种子</label><label class="toggle"><NSwitch v-model:value="task.rules.exclude_hr" />排除 H&amp;R（未知也排除）</label><label class="toggle"><NSwitch v-model:value="task.rules.delete_on_free_end" />免费结束且未完成时删除</label></NSpace>
      <NSpace class="actions"><NButton type="error" secondary @click="removeTask"><Trash2 :size="16"/>删除任务</NButton><NButton :disabled="dirty" @click="execute('run')"><Play :size="16"/>立即选种</NButton><NButton :disabled="dirty" @click="execute('check')"><SearchCheck :size="16"/>立即检查</NButton></NSpace>
     </fieldset>
    </NCard>
   </section>
   <NCard title="规则预览">
    <p class="muted">输入候选 JSON（最多 100 条）。免费或 H&amp;R 信息缺失时不会默认为安全。</p>
    <NInput v-model:value="candidates" type="textarea" :rows="13" :disabled="busy" aria-label="候选种子 JSON" />
    <NButton class="preview" type="primary" :disabled="busy || dirty || !task" @click="preview"><template #icon><FlaskConical :size="18" /></template>{{ dirty ? '请先保存配置' : '验证规则' }}</NButton>
    <NDataTable v-if="decisions.length" :columns="columns" :data="decisions" :scroll-x="480" :pagination="{ pageSize: 10 }" />
    <h3>托管种子</h3><NDataTable :columns="managedColumns" :data="managedRows" :scroll-x="640" :pagination="{ pageSize: 10 }" />
   </NCard>
  </div>
  <footer>后台心跳：{{ state?.heartbeat?.at || '尚未收到' }} · 状态：{{ state?.heartbeat?.status || '等待启动' }}</footer>
 </main>
</template>

<style scoped>
.page{display:grid;gap:22px;padding:24px;width:100%;min-width:0;box-sizing:border-box;color:var(--dian-text-primary)}
header{display:flex;align-items:center;justify-content:space-between;gap:16px;flex-wrap:wrap}h1{margin:0;font-size:28px}.eyebrow{font-size:12px;letter-spacing:2px;color:var(--dian-text-muted);margin:0 0 8px}.muted,footer{color:var(--dian-text-secondary)}
.adapters{display:grid;grid-template-columns:1fr 1fr;gap:24px}.adapter-row{display:grid;grid-template-columns:140px minmax(220px,2fr) minmax(180px,1fr) auto;gap:8px;margin-bottom:10px}.adapters>div:nth-child(2) .adapter-row{grid-template-columns:120px minmax(180px,1.5fr) minmax(170px,1fr) minmax(140px,1fr) 100px auto}.workspace{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1.15fr);gap:24px}.workspace>*,.fields>*{min-width:0}.editor{margin-top:16px}.fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}fieldset{margin:0;padding:0;border:0;min-width:0}.toggle{display:flex;gap:12px;align-items:center}.actions,.preview{margin-top:16px}footer{font-size:12px;overflow-wrap:anywhere}
@media(max-width:1100px){.adapters,.workspace{grid-template-columns:1fr}.adapter-row,.adapters>div:nth-child(2) .adapter-row{grid-template-columns:1fr}}@media(max-width:540px){.page{padding:14px}.fields{grid-template-columns:1fr;gap:0}}
</style>
