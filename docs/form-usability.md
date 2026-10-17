# Shared configuration and login form usability

The configuration grid's260px minimum and unbounded label/select/error intrinsic
widths caused viewport overflow. Real browser baseline at320px measured1480px
document width with a long label and4196px after an unbroken server error.
Tracks now shrink to available width, controls can shrink, and help/results wrap.
Normal and error states both measure320px after the repair. Desktop1440px,
tablet768px and375px also fit. Login required/email validation remains native.

Controls have visible borders and focus,44px text/action targets, and16px mobile
input text. Descriptions are associated with each control; configuration results
and sign-in errors are announced. Session/bearer behavior, endpoint resolution,
validation snapshots, secret clearing and disabled-state restoration are unchanged.

Run with an existing Playwright installation and browser:

```sh
CMS_PLAYWRIGHT_MODULE=/path/to/playwright/package.json \
  CMS_BROWSER_CHANNEL=msedge CMS_UI_ARTIFACTS=/tmp/admin-forms \
  node test/form-usability-browser.cjs
node --test test/*.test.cjs
```

The browser harness serves the actual shell with synthetic loopback configuration
APIs. Session mode is injected only into its fixture HTML; no deployed settings,
credentials, mail or hosted owner session are accessed. It checks1440/768/375/320px
geometry, long selected options/labels/help/errors, keyboard/native validation,
programmatic descriptions, synthetic validate/apply, disabled state, secret
clearing and no bearer emission in session mode. `--measure-only` can record
baseline screenshots and measurements without requiring repaired geometry.

CI runs the browser harness and existing checks against the exact PR head and
retains browser artifacts for7days. Package release, host dependency adoption,
review deployment and actual hosted full-app acceptance are subsequent steps;
this source patch performs none of them. Separate contributed tools such as
identity, authorization and onboarding have their own form implementations and
are not represented as covered by this generic shell fixture.
