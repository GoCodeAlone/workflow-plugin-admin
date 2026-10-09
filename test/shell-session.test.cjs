const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const test = require('node:test')
const vm = require('node:vm')

const html = fs.readFileSync(path.join(__dirname, '../internal/ui_dist/index.html'), 'utf8')
// Execute the shipped storage/header functions. This checks browser effects,
// rather than asserting the implementation contains a particular guard string.
const start = html.indexOf('    function storedToken() {')
const end = html.indexOf('    function scopeStrings(', start)
assert.ok(start >= 0 && end > start)
const functions = html.slice(start, end)

function shell(mode, stored = 'fixture-stale-bearer') {
  const effects = []
  const storage = new Map([['workflow.admin.token', stored]])
  const context = vm.createContext({
    window: { localStorage: {
      getItem(key) { effects.push('read'); return storage.get(key) },
      setItem(key, value) { effects.push('write'); storage.set(key, value) },
      removeItem(key) { effects.push('remove'); storage.delete(key) },
    } },
  })
  vm.runInContext(`const requiresBearerToken = ${JSON.stringify(mode)} !== 'session'; const tokenStorageKey = 'workflow.admin.token';\n${functions}`, context)
  return { run: code => vm.runInContext(code, context), effects, storage }
}

test('session APIs and embedded bridges cannot receive a stale bearer token', () => {
  const session = shell('session')
  assert.equal(session.run('storedToken()'), '')
  assert.equal(session.run('authHeaders({accept: "application/json"}).Authorization'), undefined)
  assert.equal(session.run('authHeaders({accept: "application/json"}).accept'), 'application/json')
  session.run('storeToken("fixture-new"); clearToken()')
  assert.deepEqual(session.effects, [])
  assert.equal(session.storage.get('workflow.admin.token'), 'fixture-stale-bearer')
})

test('bearer and unknown modes retain the protected bearer behavior', () => {
  for (const mode of ['bearer', 'invalid', undefined]) {
    const bearer = shell(mode)
    assert.equal(bearer.run('authHeaders().Authorization'), 'Bearer fixture-stale-bearer')
    bearer.run('storeToken("fixture-new")')
    assert.equal(bearer.run('storedToken()'), 'fixture-new')
    bearer.run('clearToken()')
    assert.equal(bearer.storage.size, 0)
  }
})
