// Real reactor + in-memory Host Call stub. This is not a DIAN115 host acceptance test.
import { WASI } from 'node:wasi'
import { readFileSync } from 'node:fs'
import assert from 'node:assert/strict'

const module = await WebAssembly.compile(readFileSync('build/runtime/plugin.wasm'))
const docs = new Map()
let revision = 0
let instance
let hostResponse
const bytes = () => new Uint8Array(instance.exports.memory.buffer)
const encode = value => Buffer.from(JSON.stringify(value))
const wasi = new WASI({ version: 'preview1', args: [], env: {}, preopens: {} })
const imports = {
 wasi_snapshot_preview1: wasi.wasiImport,
 dian115: {
  host_call(ptr, size) {
   const rpc = JSON.parse(Buffer.from(bytes().slice(ptr, ptr + size)).toString())
   assert.equal(rpc.method, 'host.call')
   const { method, path, headers = {}, body_base64 } = rpc.params
   assert.ok(path.startsWith('/api/plugin-runtime/storage/'), 'unexpected non-storage side effect')
   let response
   if (method === 'GET') {
    const doc = docs.get(path)
    response = doc ? { status: 200, headers: { etag: [doc.etag] }, body_base64: encode({ data: { value: doc.value } }).toString('base64') } : { status: 404 }
   } else if (method === 'PUT') {
    const previous = docs.get(path)
    if (headers['if-match'] && previous?.etag !== headers['if-match']) response = { status: 412 }
    else {
     assert.ok(headers['idempotency-key'].length >= 16)
     const { value } = JSON.parse(Buffer.from(body_base64, 'base64').toString())
     const etag = `"pkv_${++revision}"`
     docs.set(path, { value, etag }); response = { status: 200, headers: { etag: [etag] } }
    }
   } else throw new Error(`unexpected method ${method}`)
   hostResponse = encode({ result: response })
   return hostResponse.length
  },
  host_read(ptr, capacity) {
   assert.ok(capacity >= hostResponse.length)
   bytes().set(hostResponse, ptr)
   return hostResponse.length
  }
 }
}
instance = await WebAssembly.instantiate(module, imports)
wasi.initialize(instance)
function rpc(method, params) {
 const request = encode({ method, params })
 const ptr = instance.exports.dian115_alloc(request.length)
 bytes().set(request, ptr)
 const result = instance.exports.dian115_handle(ptr, request.length)
 const offset = Number(result >> 32n), size = Number(result & 0xffffffffn)
 const response = JSON.parse(Buffer.from(bytes().slice(offset, offset + size)).toString())
 assert.equal(response.error, undefined, JSON.stringify(response.error))
 assert.ok(size <= 256 * 1024)
 return response.result
}
let invocation = 0
const invoke = (op, payload) => rpc('runtime.invoke', { envelope: { op, invocation_id: `smoke-${++invocation}`, payload } })
assert.equal(rpc('runtime.initialize', { protocol: 'dian115:wasm@1' }).ready, true)
const initial = invoke('state', {})
assert.equal(initial.state.live_execution, false)
assert.equal(invoke('state', { if_none_match: initial.etag }).not_modified, true)
const config = { schema_version: 1, tasks: [{ id: 'one', name: 'Test', enabled: false, site_id: '', downloader_id: '', brush_minutes: 10, check_minutes: 5, rules: { free_only: true, exclude_hr: true, include: '', exclude: '', min_bytes: 0, max_bytes: 0 } }] }
assert.equal(invoke('action', { id: 'save-config', input: { config, revision: '' } }).status, 'succeeded')
assert.equal(invoke('action', { id: 'save-config', input: { config, revision: '' } }).status, 'failed')
const saved = invoke('state', {})
assert.equal(saved.state.config.tasks[0].name, 'Test')
assert.notEqual(saved.etag, initial.etag)
const preview = invoke('action', { id: 'preview', input: { task_id: 'one', candidates: [
 { id: 'known', title: 'Movie', size_bytes: 1, free: true, hr: false },
 { id: 'unknown', title: 'Movie', size_bytes: 1, free: null, hr: null }
] } })
assert.equal(preview.status, 'succeeded')
assert.deepEqual(preview.decisions.map(x => x.reason), ['matched', 'promotion_unknown'])
for (const id of ['run', 'check', 'delete']) assert.equal(invoke('action', { id, input: {} }).status, 'skipped')
assert.equal(docs.size, 1, 'disabled actions must not mutate storage')
assert.equal(invoke('job', { id: 'unknown' }).status, 'skipped')
assert.equal(rpc('runtime.shutdown', {}).stopping, true)
console.log('Real WASM smoke: PASS (mock Broker; actual host/resident lifecycle not covered)')
