import { readFileSync } from 'node:fs'
const module = new WebAssembly.Module(readFileSync('build/runtime/plugin.wasm'))
const exports = WebAssembly.Module.exports(module).map(item => item.name)
for (const name of ['memory', 'dian115_alloc', 'dian115_handle']) {
 if (!exports.includes(name)) throw new Error(`Missing reactor export: ${name}`)
}
if (exports.includes('_start')) throw new Error('Runtime must be a reactor, not a command')
const imports = WebAssembly.Module.imports(module)
for (const name of ['host_call', 'host_read']) {
 if (!imports.some(item => item.module === 'dian115' && item.name === name)) throw new Error(`Missing broker import: ${name}`)
}
console.log('WASM reactor ABI: PASS')
