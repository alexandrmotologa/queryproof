const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');
const puppeteer = require('puppeteer');

async function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function waitForServer(url, timeoutMs = 15000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url);
      if (res.ok) return true;
    } catch (_) {}
    await sleep(250);
  }
  return false;
}

async function main() {
  const rootDir = path.join(__dirname, '..');
  const imgDir = path.join(rootDir, 'docs', 'images');
  const framesDir = path.join(rootDir, 'docs', 'images', 'frames');

  if (!fs.existsSync(imgDir)) fs.mkdirSync(imgDir, { recursive: true });
  if (!fs.existsSync(framesDir)) fs.mkdirSync(framesDir, { recursive: true });

  console.log('[1/5] Starting queryproof server on :7373...');
  const exePath = path.join(rootDir, 'bin', 'queryproof.exe');
  const server = spawn(exePath, ['serve', '--port', '7373', '--catalog', 'examples/catalog.json', '--metrics', 'examples/metrics.yaml'], {
    cwd: rootDir,
    stdio: 'ignore',
  });

  const ready = await waitForServer('http://127.0.0.1:7373/api/health');
  if (!ready) {
    server.kill();
    throw new Error('Server failed to start in time');
  }
  console.log('✓ queryproof server ready on http://127.0.0.1:7373');

  console.log('[2/5] Launching Puppeteer browser...');
  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

  await page.goto('http://127.0.0.1:7373', { waitUntil: 'networkidle0' });
  await sleep(1000);

  console.log('[3/5] Capturing high-resolution screenshots...');

  // 1. Dashboard Desktop Overview
  await page.screenshot({
    path: path.join(imgDir, 'dashboard_desktop.png'),
    fullPage: false,
  });
  console.log('✓ Saved dashboard_desktop.png');

  // 2. Visual Fan-Out Flow
  const fanoutTab = await page.waitForSelector('button ::-p-text(Visual Fan-Out Flow)');
  if (fanoutTab) {
    await fanoutTab.click();
    await sleep(800);
    await page.screenshot({
      path: path.join(imgDir, 'fanout_flow.png'),
      fullPage: false,
    });
    console.log('✓ Saved fanout_flow.png');
  }

  // 3. Cryptographic Passport with Signature Verification
  const voucherTab = await page.waitForSelector('button ::-p-text(Integrity Passport)');
  if (voucherTab) {
    await voucherTab.click();
    await sleep(500);

    const verifyBtn = await page.waitForSelector('button ::-p-text(Verify Signature)');
    if (verifyBtn) {
      await verifyBtn.click();
      await sleep(600);
    }

    await page.screenshot({
      path: path.join(imgDir, 'passport_verified.png'),
      fullPage: false,
    });
    console.log('✓ Saved passport_verified.png');
  }

  // 4. Audit Ledger & Trends
  const ledgerTab = await page.waitForSelector('button ::-p-text(Audit Ledger & Trends)');
  if (ledgerTab) {
    await ledgerTab.click();
    await sleep(600);
    await page.screenshot({
      path: path.join(imgDir, 'audit_ledger.png'),
      fullPage: false,
    });
    console.log('✓ Saved audit_ledger.png');
  }

  // 5. Mobile Cockpit View (iPhone 14/15 Pro: 393 x 852)
  await page.setViewport({ width: 393, height: 852, deviceScaleFactor: 2, isMobile: true });
  const probesTab = await page.waitForSelector('button ::-p-text(Falsification Probes)');
  if (probesTab) await probesTab.click();
  await sleep(600);
  await page.screenshot({
    path: path.join(imgDir, 'mobile_cockpit.png'),
    fullPage: false,
  });
  console.log('✓ Saved mobile_cockpit.png');

  // 6. Record Frames for Demo GIF (1280x780)
  console.log('[4/5] Recording sequence frames for queryproof_demo.gif...');
  await page.setViewport({ width: 1280, height: 780, deviceScaleFactor: 1.5, isMobile: false });
  await page.goto('http://127.0.0.1:7373', { waitUntil: 'networkidle0' });
  await sleep(800);

  let frameIdx = 0;
  const captureFrame = async (count = 1) => {
    for (let c = 0; c < count; c++) {
      const framePath = path.join(framesDir, `frame_${String(frameIdx).padStart(4, '0')}.png`);
      await page.screenshot({ path: framePath });
      frameIdx++;
    }
  };

  // State 1: Desktop Overview on Scenario 1
  await captureFrame(10);

  // State 2: Select Scenario 2 (Silent Mean Bias)
  const sc2 = await page.waitForSelector('button ::-p-text(Silent Mean Bias)');
  if (sc2) {
    await sc2.click();
    await sleep(500);
    await captureFrame(12);
  }

  // State 3: Switch back to Scenario 1 (Cartesian Fanout)
  const sc1 = await page.waitForSelector('button ::-p-text(1:N Join Fan-Out)');
  if (sc1) {
    await sc1.click();
    await sleep(500);
    await captureFrame(8);
  }

  // State 4: Click Visual Fan-Out Flow tab
  const tabFan = await page.waitForSelector('button ::-p-text(Visual Fan-Out Flow)');
  if (tabFan) {
    await tabFan.click();
    await sleep(600);
    await captureFrame(14);
  }

  // State 5: Click Cryptographic Passport tab and verify signature
  const tabVch = await page.waitForSelector('button ::-p-text(Integrity Passport)');
  if (tabVch) {
    await tabVch.click();
    await sleep(500);
    await captureFrame(8);

    const vBtn = await page.waitForSelector('button ::-p-text(Verify Signature)');
    if (vBtn) {
      await vBtn.click();
      await sleep(600);
      await captureFrame(14);
    }
  }

  // State 6: Click Audit Ledger tab
  const tabLed = await page.waitForSelector('button ::-p-text(Audit Ledger & Trends)');
  if (tabLed) {
    await tabLed.click();
    await sleep(500);
    await captureFrame(14);
  }

  // Close browser and kill server
  await browser.close();
  server.kill();

  console.log(`✓ Recorded ${frameIdx} frames in ${framesDir}`);
  console.log('[5/5] Assembling animated GIF with ffmpeg...');
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
