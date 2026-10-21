// tweb appGifsManager.searchGifs transport: server config -> bot -> inline results.
// Runs outside the frontend; no Telegram session/credentials cross the HTTP API.
import fs from 'node:fs/promises';
import path from 'node:path';
import http from 'node:http';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';
import { createInterface } from 'node:readline/promises';
import { TelegramClient, Api } from 'teleproto';
import { StringSession } from 'teleproto/sessions';

const directory = path.dirname(fileURLToPath(import.meta.url));
for (const line of (await fs.readFile(path.join(directory, '.env'), 'utf8').catch(() => '')).split('\n')) {
 const match = line.match(/^([A-Z_]+)=(.*)$/);
 if (match && !process.env[match[1]]) process.env[match[1]] = match[2].trim().replace(/^(['"])(.*)\1$/, '$2');
}
const apiId = Number(process.env.TELEGRAM_API_ID), apiHash = process.env.TELEGRAM_API_HASH;
if (!Number.isInteger(apiId) || !apiId || !apiHash) throw new Error('Set this project’s TELEGRAM_API_ID and TELEGRAM_API_HASH in tools/telegram-media/.env');
const sessionPath = path.join(directory, '.session');
const session = await fs.readFile(sessionPath, 'utf8').catch(() => '');
const client = new TelegramClient(new StringSession(session), apiId, apiHash, { maxConcurrentDownloads: 2, connectionRetries: 2, requestRetries: 1, floodSleepThreshold: 0, receiveUpdates: false });
client.setLogLevel('none');
if (process.argv.includes('--login')) {
 const terminal = createInterface({ input: process.stdin, output: process.stdout });
 try {
  await client.start({ phoneNumber: () => terminal.question('Telegram phone number: '), phoneCode: () => terminal.question('Telegram login code: '), password: () => terminal.question('Telegram two-step verification password: '), onError: () => { console.error('Telegram sign-in failed; check the entered details.'); } });
  await fs.writeFile(sessionPath, client.session.save(), { mode: 0o600 }); await fs.chmod(sessionPath, 0o600);
  console.log('Telegram media account connected. Run yarn start in this directory.');
 } finally { terminal.close(); await client.disconnect(); }
 process.exit(0);
}
if (!session) throw new Error('Telegram media session missing. Run yarn run connect first.');
await client.connect();
if (!(await client.isUserAuthorized())) throw new Error('Telegram media session expired. Run yarn run connect again.');
const config = await client.invoke(new Api.help.GetConfig());
const bot = await client.getInputEntity(config.gifSearchUsername || 'gif');
const files = path.resolve(directory, '../../static/telegram/gifs'); await fs.mkdir(files, { recursive: true });
const cache = new Map(), media = new Map(), downloads = new Map();
let queue = Promise.resolve();
async function search(query, offset) {
 const key = JSON.stringify([query, offset]), cached = cache.get(key);
 if (cached && cached.until > Date.now()) return cached.result;
 const work = async () => {
  const response = await client.invoke(new Api.messages.GetInlineBotResults({ bot, peer: new Api.InputPeerEmpty(), query, offset }));
  const documents = [];
  for (const result of response.results) {
   const doc = result.document;
   if (!doc || !['video/mp4', 'image/gif'].includes(doc.mimeType) || Number(doc.size) > 20971520) continue;
   const extension = doc.mimeType === 'video/mp4' ? 'mp4' : 'gif';
   const id = createHash('sha256').update(doc.id.toString() + ':' + doc.size.toString()).digest('hex');
   const filename = `${id}.${extension}`;
   media.set(filename, doc);
   while (media.size > 2048) media.delete(media.keys().next().value);
   const size = doc.attributes.find(attribute => attribute.w && attribute.h);
   documents.push({ kind: 'gif', id, url: `/static/telegram/gifs/${id}.${extension}`, width: size?.w || 320, height: size?.h || 240 });
  }
  const data = { documents, nextOffset: response.nextOffset || '' };
  cache.set(key, { result: data, until: Date.now() + 300000 });
  while (cache.size > 64) cache.delete(cache.keys().next().value);
  return data;
 };
 const pending = queue.then(work); queue = pending.catch(() => {}); return pending;
}
async function prepare(filename) {
 if (!/^[a-f0-9]{64}\.(mp4|gif)$/.test(filename)) throw new Error('Invalid media');
 const destination = path.join(files, filename);
 if (await fs.stat(destination).catch(() => null)) return;
 const doc = media.get(filename);
 if (!doc) throw new Error('Search reference expired');
 if (downloads.has(filename)) return downloads.get(filename);
 const pending = (async () => {
  const bytes = await client.downloadMedia(doc, { workers: 1 });
  if (!bytes?.length || bytes.length > 20971520) throw new Error('Invalid media size');
  const temporary = `${destination}.${process.pid}.tmp`;
  await fs.writeFile(temporary, bytes); await fs.rename(temporary, destination);
 })().finally(() => downloads.delete(filename));
 downloads.set(filename, pending); return pending;
}
const socket = process.env.TELEGRAM_MEDIA_SOCKET || path.join(directory, '.socket');
// Do not unlink a live bridge socket.
if (await fs.stat(socket).catch(() => null)) throw new Error('Telegram media socket already exists. Stop the old bridge before starting another.');
const server = http.createServer(async (request, response) => {
 const url = new URL(request.url, 'http://telegram-media');
 const query = url.searchParams.get('query') || '', offset = url.searchParams.get('offset') || '';
 if (request.method !== 'GET' || (!url.pathname.startsWith('/prepare/') && url.pathname !== '/gifs') || query.length > 200 || offset.length > 1024) { response.writeHead(400).end(); return; }
 try { const data = url.pathname.startsWith('/prepare/') ? (await prepare(url.pathname.slice('/prepare/'.length)), {ready:true}) : await search(query, offset); response.writeHead(200, { 'Content-Type': 'application/json' }).end(JSON.stringify(data)); }
 catch { response.writeHead(503, { 'Content-Type': 'application/json' }).end(JSON.stringify({ error: 'telegram_media_unavailable' })); }
});
await new Promise(resolve => server.listen(socket, resolve)); await fs.chmod(socket, 0o600);
console.log('Telegram GIF bridge ready.');
for (const event of ['SIGTERM', 'SIGINT']) process.on(event, async () => { server.close(); await client.disconnect(); await fs.rm(socket, { force: true }); process.exit(0); });
