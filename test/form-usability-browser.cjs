// Isolated real admin shell with synthetic configuration APIs. Reuse Playwright.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const http = require('node:http');
const path = require('node:path');
const { createRequire } = require('node:module');
async function main() {
  if (!process.env.CMS_PLAYWRIGHT_MODULE) throw new Error('CMS_PLAYWRIGHT_MODULE must point to existing Playwright');
  const { chromium } = createRequire(path.resolve(process.env.CMS_PLAYWRIGHT_MODULE))('playwright');
  const html = fs.readFileSync(process.env.ADMIN_UI_HTML || path.resolve(__dirname, '../internal/ui_dist/index.html'), 'utf8');
  const artifacts = process.env.CMS_UI_ARTIFACTS, measureOnly = process.argv.includes('--measure-only');
  if (artifacts) fs.mkdirSync(artifacts, { recursive:true });
  const calls = [], measurements = [], external = [], errors = [];
  let failValidation = false;
  const surface = { id:'fixture', title:'Fixture configuration', path:'/api/admin/fixture', render_mode:'config-form', metadata:{ describe_path:'/api/admin/fixture', validate_path:'/api/admin/fixture/validate', apply_path:'/api/admin/fixture/apply' } };
  const description = { effective_config:{ name:'Fixture tenant', kind:'review', locked:'Disabled value', enabled:true }, groups:[{ key:'general', label:'Shared configuration', description:'Synthetic fixture, no real services or settings.', controls:[
    { key:'name', label:'Site label', help_text:'A visible editorial label.' },
    { key:'kind', label:'Destination', options:[{ value:'review', label:'A long review destination label '.repeat(4) }] },
    { key:'enabled', label:'Enabled', input_type:'toggle' },
    { key:'secret', label:'Provider secret', input_type:'secret', secret:true, help_text:'Leave blank to keep the configured secret.' },
    { key:'locked', label:'Disabled setting', enabled:false, disabled_reason:'Only an authorized owner can change this value.' },
    { key:'long', label:'LongLabel'.repeat(12), help_text:'LongHelp'.repeat(24) },
  ] }] };
  const server = http.createServer(async (req,res) => {
    const chunks=[]; for await (const c of req) chunks.push(c);
    const body = chunks.length ? JSON.parse(Buffer.concat(chunks)) : null;
    calls.push({ method:req.method, path:req.url, body, authorization:!!req.headers.authorization });
    res.setHeader('Content-Type', 'application/json');
    if (req.url === '/api/admin/contributions') return res.end(JSON.stringify({ contributions:[surface] }));
    if (req.url === surface.path) return res.end(JSON.stringify(description));
    if (req.url.endsWith('/validate')) {
      if (failValidation) { res.statusCode=422; return res.end(JSON.stringify({ error:'LongServerError'.repeat(30) })); }
      return res.end(JSON.stringify({ valid:true }));
    }
    if (req.url.endsWith('/apply')) return res.end(JSON.stringify({ applied:true }));
    if (req.url === '/api/admin/auth/login') { res.statusCode=401; return res.end('{}'); }
    res.setHeader('Content-Type','text/html');
    return res.end(req.url === '/login-fixture' ? html : html.replace('data-admin-shell-version="1"', 'data-admin-shell-version="1" data-auth-mode="session"'));
  });
  await new Promise(resolve => server.listen(0,'127.0.0.1',resolve));
  let browser;
  try {
    const origin = 'http://127.0.0.1:' + server.address().port;
    browser = await chromium.launch({ headless:true, ...(process.env.CMS_BROWSER_CHANNEL ? { channel:process.env.CMS_BROWSER_CHANNEL } : {}) });
    const context = await browser.newContext({ viewport:{ width:1440,height:1100 }, serviceWorkers:'block' });
    await context.route('**/*',route => { if(new URL(route.request().url()).origin !== origin) { external.push(route.request().url()); return route.abort(); } return route.continue(); });
    const page = await context.newPage(); page.setDefaultTimeout(10000); page.on('pageerror',e=>errors.push(e.message));
    async function open() { await page.goto(origin); await page.getByRole('button',{name:'Fixture configuration',exact:true}).click(); await page.locator('.config-form').waitFor(); }
    async function capture(label) {
      const measurement=await page.evaluate(()=>({viewport:innerWidth,document:document.documentElement.scrollWidth,controls:[...document.querySelectorAll('.panel.active input,.panel.active select,.panel.active button')].map(el=>{const r=el.getBoundingClientRect();return{id:el.id,width:r.width,right:r.right,x:r.x}}),groups:[...document.querySelectorAll('.config-controls')].map(el=>({width:el.clientWidth,scroll:el.scrollWidth}))}));
      measurements.push({label,...measurement});
      if(!measureOnly) { assert.ok(measurement.document <= measurement.viewport+1,label+' document fits');for(const c of measurement.controls)assert.ok(c.x>=0&&c.right<=measurement.viewport+1,label+' control fits '+c.id);for(const g of measurement.groups)assert.ok(g.scroll<=g.width+1,label+' group fits'); }
      if(artifacts)await page.screenshot({path:path.join(artifacts,label+'.png'),fullPage:true});
    }
    for(const width of [1440,768,375,320]) {
      await page.setViewportSize({width,height:1100});await open();await capture('config-'+width);
      failValidation=true;await page.getByRole('button',{name:'Validate changes',exact:true}).click();await page.locator('.config-result').filter({hasText:'LongServerError'}).waitFor();await capture('config-error-'+width);failValidation=false;
      await page.goto(origin+'/login-fixture');await page.locator('#login-form').waitFor();await capture('login-'+width);
    }
    if(!measureOnly) {
      await open();
      for(const input of await page.locator('.config-control input,.config-control select').all()) {
        const association=await input.evaluate(el=>({name:el.labels?.[0]?.textContent,help:el.getAttribute('aria-describedby'),description:document.getElementById(el.getAttribute('aria-describedby'))?.textContent}));
        assert.ok(association.name);if(await input.getAttribute('title'))assert.ok(association.help&&association.description);
      }
      await page.getByLabel('Site label',{exact:true}).fill('Edited local label');await page.getByLabel('Site label',{exact:true}).press('Tab');
      await page.getByLabel('Provider secret',{exact:true}).fill('fixture-only-secret');await page.getByLabel('Provider secret',{exact:true}).press('Tab');
      await page.getByRole('button',{name:'Validate changes',exact:true}).click();await page.locator('.config-result').filter({hasText:'"valid": true'}).waitFor();
      await page.getByRole('button',{name:'Apply changes',exact:true}).click();await page.locator('.config-result').filter({hasText:'"applied": true'}).waitFor();
      assert.equal(await page.getByLabel('Provider secret',{exact:true}).inputValue(),'');assert.equal(await page.getByLabel('Disabled setting',{exact:true}).isDisabled(),true);
      const applied=calls.findLast(c=>c.path.endsWith('/apply'));assert.equal(applied.body.desired_config.name,'Edited local label');assert.equal(applied.body.desired_config.secret,'fixture-only-secret');
      assert.equal(calls.some(c=>c.authorization),false,'session fixture never emits bearer credentials');
      await page.goto(origin+'/login-fixture');await page.getByRole('button',{name:'Sign in',exact:true}).click();assert.equal(await page.locator('#admin-email').evaluate(el=>document.activeElement===el),true);
      const n=calls.filter(c=>c.path.endsWith('/auth/login')).length;await page.getByLabel('Email',{exact:true}).fill('invalid');await page.getByLabel('Password',{exact:true}).fill('fixture-only');await page.getByRole('button',{name:'Sign in',exact:true}).click();assert.equal(calls.filter(c=>c.path.endsWith('/auth/login')).length,n);
      await page.getByLabel('Email',{exact:true}).fill('fixture@example.test');await page.getByRole('button',{name:'Sign in',exact:true}).click();await page.locator('#login-error').filter({hasText:'Sign-in failed.'}).waitFor();
    }
    assert.deepEqual(external,[]);assert.deepEqual(errors,[]);
    const result={pass:true,measureOnly,browser:browser.version(),boundary:'actual upstream shell; synthetic loopback APIs; session-mode injection confined to fixture; no live auth/config writes',measurements,checks:['configuration and login desktop/tablet/375px/320px geometry','long labels/options/help/server errors',...(measureOnly?[]:['accessible control descriptions','synthetic validate/apply and secret clearing','disabled state retained','session bearer isolation','native required/email validation and login error'])]};
    if(artifacts)fs.writeFileSync(path.join(artifacts,'results.json'),JSON.stringify(result,null,2)+'\n');console.log(JSON.stringify(result));
  } finally {
    try {
      if (browser) await browser.close();
    } finally {
      await new Promise(resolve=>server.close(resolve));
    }
  }
}
main().catch(e=>{console.error(e);process.exitCode=1});
