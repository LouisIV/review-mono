import { chromium } from 'playwright';
import { createServer } from 'http';
import { readFileSync } from 'fs';
import { resolve, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const PORT = 4567;
const HTML_PATH = resolve(__dirname, '..', 'screenshot-demo.html');

const server = createServer((_req, res) => {
  const html = readFileSync(HTML_PATH, 'utf8');
  res.writeHead(200, { 'Content-Type': 'text/html' });
  res.end(html);
});

server.listen(PORT, async () => {
  console.log(`Serving demo page on http://localhost:${PORT}`);

  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });

  try {
    await page.goto(`http://localhost:${PORT}`, { waitUntil: 'networkidle' });
    // Wait for Tailwind CDN to fully process
    await page.waitForTimeout(2000);
    await page.screenshot({ path: 'gui-screenshot.png', fullPage: true });
    console.log('Screenshot saved to gui-screenshot.png');
  } finally {
    await browser.close();
    server.close();
  }
});
