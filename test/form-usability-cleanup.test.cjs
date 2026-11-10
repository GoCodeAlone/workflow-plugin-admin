const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const filename = path.join(__dirname, 'form-usability-browser.cjs');
const source = fs.readFileSync(filename, 'utf8');

// Exercise the actual CLI lifecycle without opening a socket or starting a browser.
async function runHarness({ launchError, contextError, closeError } = {}) {
  const effects = [], logs = [], errors = [];
  const mockProcess = {
    env: { CMS_PLAYWRIGHT_MODULE: path.join(__dirname, 'fixture-package.json') },
    argv: [process.execPath, filename, '--measure-only'],
  };
  let listening = false;
  const server = {
    listen(port, host, callback) {
      assert.equal(port, 0);
      assert.equal(host, '127.0.0.1');
      effects.push('server.listen');
      listening = true;
      callback();
    },
    address() { return { port: 12345 }; },
    close(callback) {
      effects.push('server.close');
      setImmediate(() => {
        listening = false;
        effects.push('server.closed');
        callback();
      });
    },
  };
  const locator = {
    async click() {},
    async waitFor() {},
    filter() { return this; },
  };
  const page = {
    setDefaultTimeout() {},
    on() {},
    async goto() {},
    async setViewportSize() {},
    getByRole() { return locator; },
    locator() { return locator; },
    async evaluate() { return { viewport: 1440, document: 1440, controls: [], groups: [] }; },
  };
  const browser = {
    async newContext() {
      if (contextError) throw contextError;
      return { async route() {}, async newPage() { return page; } };
    },
    version() { return 'fixture-browser'; },
    async close() {
      effects.push('browser.close');
      if (closeError) throw closeError;
    },
  };
  const modules = {
    'node:assert/strict': assert,
    'node:fs': { readFileSync() { return '<html></html>'; } },
    'node:http': { createServer() { return server; } },
    'node:path': path,
    'node:module': {
      createRequire() {
        return name => {
          assert.equal(name, 'playwright');
          return { chromium: { async launch() {
            effects.push('browser.launch');
            if (launchError) throw launchError;
            return browser;
          } } };
        };
      },
    },
  };
  await vm.runInNewContext(source, {
    __dirname,
    process: mockProcess,
    console: { log: value => logs.push(value), error: error => errors.push(error) },
    require(name) {
      assert.ok(Object.hasOwn(modules, name), 'Unexpected module: ' + name);
      return modules[name];
    },
  }, { filename });
  return { effects, logs, errors, exitCode: mockProcess.exitCode, listening };
}

test('browser launch failure closes the fixture server and reports the original error', async () => {
  const error = new Error('synthetic Chromium launch failure');
  const result = await runHarness({ launchError: error });
  assert.deepEqual(result.effects, ['server.listen', 'browser.launch', 'server.close', 'server.closed']);
  assert.equal(result.listening, false);
  assert.equal(result.exitCode, 1);
  assert.deepEqual(result.errors, [error]);
  assert.deepEqual(result.logs, []);
});

test('a browser setup failure closes both resources and reports the original error', async () => {
  const error = new Error('synthetic context failure');
  const result = await runHarness({ contextError: error });
  assert.deepEqual(result.effects, ['server.listen', 'browser.launch', 'browser.close', 'server.close', 'server.closed']);
  assert.equal(result.listening, false);
  assert.equal(result.exitCode, 1);
  assert.deepEqual(result.errors, [error]);
  assert.deepEqual(result.logs, []);
});

test('browser close failure still closes the fixture server and reports failure', async () => {
  const error = new Error('synthetic browser close failure');
  const result = await runHarness({ closeError: error });
  assert.deepEqual(result.effects, ['server.listen', 'browser.launch', 'browser.close', 'server.close', 'server.closed']);
  assert.equal(result.listening, false);
  assert.equal(result.exitCode, 1);
  assert.deepEqual(result.errors, [error]);
});

test('successful measurement run closes both resources without setting a failure exit code', async () => {
  const result = await runHarness();
  assert.deepEqual(result.effects, ['server.listen', 'browser.launch', 'browser.close', 'server.close', 'server.closed']);
  assert.equal(result.listening, false);
  assert.equal(result.exitCode, undefined);
  assert.deepEqual(result.errors, []);
  assert.equal(result.logs.length, 1);
  const report = JSON.parse(result.logs[0]);
  assert.equal(report.pass, true);
  assert.equal(report.measureOnly, true);
  assert.equal(report.measurements.length, 12);
});
