const { chromium } = require('playwright');
const path = require('path');
const fs = require('fs');

const BASE = process.env.BASE_URL || 'http://127.0.0.1:9090';
const OUT = path.resolve(__dirname, '../../docs/screenshots');

(async () => {
  fs.mkdirSync(OUT, { recursive: true });

  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  const page = await ctx.newPage();

  // 1. Login page
  await page.goto(`${BASE}/login`);
  await page.waitForSelector('input[id="username"]');
  await page.screenshot({ path: path.join(OUT, '01-login-page.png'), fullPage: false });
  console.log('✓ 01-login-page.png');

  // 2. Login with wrong password (error state)
  await page.fill('#username', 'admin');
  await page.fill('#password', 'wrongpass');
  await page.click('button[type="submit"]');
  await page.waitForSelector('[role="alert"]', { timeout: 5000 }).catch(() => {});
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(OUT, '02-login-error.png'), fullPage: false });
  console.log('✓ 02-login-error.png');

  // 3. Login successfully
  await page.fill('#username', 'admin');
  await page.fill('#password', 'admin1234');
  await page.click('button[type="submit"]');
  await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 15000 });
  await page.waitForTimeout(1500);
  await page.screenshot({ path: path.join(OUT, '03-user-dashboard.png'), fullPage: true });
  console.log('✓ 03-user-dashboard.png');

  // 4. Admin dashboard
  await page.goto(`${BASE}/admin`);
  await page.waitForTimeout(1500);
  await page.screenshot({ path: path.join(OUT, '04-admin-dashboard.png'), fullPage: true });
  console.log('✓ 04-admin-dashboard.png');

  // 5. Language switcher (Turkish)
  await page.click('button[aria-pressed="false"]');
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(OUT, '05-admin-turkish.png'), fullPage: true });
  console.log('✓ 05-admin-turkish.png');

  await browser.close();
  console.log('All screenshots saved to', OUT);
})();
