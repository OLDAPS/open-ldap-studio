import assert from 'node:assert/strict';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { basename, join } from 'node:path';
import { cwd } from 'node:process';
import { createServer } from 'vite';

// Exercise the real Vite config, React transform, and HMR WebSocket without a
// display. The fixture belongs to this test and is always removed afterward.
const fixture = await mkdtemp(join(cwd(), 'src', '.hmr-smoke-'));
const modulePath = `/src/${basename(fixture)}/module.ts`;
let server;
let socket;

function messageMatching(predicate) {
  return new Promise((resolve, reject) => {
    const timeout = globalThis.setTimeout(() => finish(new Error('HMR message timed out')), 10000);
    function finish(error, message) {
      globalThis.clearTimeout(timeout);
      socket.removeEventListener('message', onMessage);
      socket.removeEventListener('error', onError);
      if (error) reject(error);
      else resolve(message);
    }
    function onMessage(event) {
      const message = JSON.parse(event.data);
      if (predicate(message)) finish(undefined, message);
    }
    function onError() {
      finish(new Error('HMR WebSocket failed'));
    }
    socket.addEventListener('message', onMessage);
    socket.addEventListener('error', onError);
  });
}

try {
  // Dependency prebundling is unrelated to HMR and can still be running when
  // this short smoke test finishes. Disable it for a deterministic shutdown.
  server = await createServer({ optimizeDeps: { noDiscovery: true, include: [] } });
  await server.listen();
  const origin = server.resolvedUrls.local[0];
  const index = await (await globalThis.fetch(origin)).text();
  assert.match(index, /id="root"/);
  const app = await (await globalThis.fetch(`${origin}src/app/App.tsx`)).text();
  assert.match(app, /\$RefreshReg\$/);

  const source = (value) =>
    `export const marker = '${value}';\nif (import.meta.hot) import.meta.hot.accept();\n`;
  await writeFile(join(fixture, 'module.ts'), source('before'));
  assert.equal((await globalThis.fetch(`${origin.slice(0, -1)}${modulePath}`)).status, 200);

  socket = new globalThis.WebSocket(
    `${origin.replace('http:', 'ws:')}?token=${server.config.webSocketToken}`,
    'vite-hmr',
  );
  await messageMatching((message) => message.type === 'connected');
  const update = messageMatching(
    (message) =>
      message.type === 'update' && message.updates.some((item) => item.path === modulePath),
  );
  await writeFile(join(fixture, 'module.ts'), source('after'));
  await update;
  const changed = await (await globalThis.fetch(`${origin.slice(0, -1)}${modulePath}`)).text();
  assert.match(changed, /after/);
  globalThis.console.log('Frontend entrypoint, React Fast Refresh, and HMR WebSocket passed.');
} finally {
  if (socket && socket.readyState !== globalThis.WebSocket.CLOSED) {
    await new Promise((resolve) => {
      socket.addEventListener('close', resolve, { once: true });
      socket.close();
    });
  }
  await rm(fixture, { recursive: true, force: true });
  await server?.close();
}
