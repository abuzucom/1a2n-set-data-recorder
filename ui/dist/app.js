const status = document.querySelector('#status');
const decks = document.querySelector('#decks');
const devices = document.querySelector('#devices');
const timeline = document.querySelector('#timeline');
const title = document.querySelector('#session-heading');
const connection = document.querySelector('#connection');
const apiToken = document.querySelector('#api-token');
let currentSession;

function mutate(path, options = {}) {
  const headers = new Headers(options.headers);
  headers.set('Authorization', `Bearer ${apiToken.value}`);
  return fetch(path, {...options, headers});
}

function createDeckCard(deck) {
  const card = document.createElement('article');
  card.className = `deck ${deck.isOnAir ? 'on-air' : ''}`;
  const player = document.createElement('p'); player.className = 'eyebrow'; player.textContent = `Player ${deck.playerNumber}`;
  const track = document.createElement('h2'); track.textContent = deck.title || 'No track loaded';
  const artist = document.createElement('p'); artist.textContent = deck.artist || deck.playState;
  const metadata = document.createElement('p'); metadata.className = 'deck-meta'; metadata.textContent = `${deck.bpm || '--'} BPM${deck.key ? ` ${deck.key}` : ''}${deck.isMaster ? ' Master' : ''}`;
  const source = document.createElement('p'); source.className = 'deck-source'; source.textContent = `${deck.model || 'Unknown player'} | ${deck.sourceSlot || 'No source'} | ${deck.playState || 'Unknown state'}`;
  card.append(player, track, artist, metadata, source);
  return card;
}

async function refresh() {
  const response = await fetch('/status');
  const data = await response.json();
  title.textContent = data.session.id ? data.session.name : 'No session active';
  currentSession = data.session.id ? data.session : undefined;
  status.textContent = JSON.stringify(data, null, 2);
  decks.replaceChildren(...data.decks.map(createDeckCard));
  devices.replaceChildren(...data.devices.map(device => { const row = document.createElement('li'); row.textContent = `${device.type} ${device.deviceId} ${device.name} ${device.address}`; return row; }));
  if (!currentSession) { timeline.replaceChildren(); return; }
  const eventResponse = await fetch(`/sessions/${currentSession.id}/events`);
  if (!eventResponse.ok) return;
  const eventData = await eventResponse.json();
  timeline.replaceChildren(...eventData.events.map(item => { const row = document.createElement('li'); row.textContent = `${new Date(item.timestamp).toLocaleTimeString()} ${item.eventType}`; return row; }));
}

document.querySelector('#session').onsubmit = async event => { event.preventDefault(); const response = await mutate('/sessions', { method: 'POST', headers: {'content-type': 'application/json'}, body: JSON.stringify({name: event.target.name.value}) }); if (response.ok) { event.target.reset(); refresh(); } };
document.querySelector('#end-session').onclick = async () => { if (!currentSession) return; const response = await mutate(`/sessions/${currentSession.id}/end`, {method:'POST'}); if (response.ok) refresh(); };
document.querySelector('#handoff').onsubmit = async event => { event.preventDefault(); if (!currentSession) return; await mutate(`/sessions/${currentSession.id}/dj-handoff`, {method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({nextDjName:event.target.next.value})}); event.target.reset(); };
document.querySelectorAll('[data-recording]').forEach(button => button.onclick = async () => { if (!currentSession) return; await mutate(`/sessions/${currentSession.id}/recording/${button.dataset.recording}`, {method:'POST'}); });
document.querySelector('#recording-metadata').onsubmit = async event => { event.preventDefault(); if (!currentSession) return; const start = event.target.recordingStart.value; const stop = event.target.recordingStop.value; const payload = {audioFilePath:event.target.audioPath.value,offsetSeconds:Number(event.target.offsetSeconds.value)}; if (start) payload.recordingStartTimestamp = new Date(start).toISOString(); if (stop) payload.recordingStopTimestamp = new Date(stop).toISOString(); const response = await mutate(`/sessions/${currentSession.id}/recording/metadata`, {method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify(payload)}); if (response.ok) event.target.reset(); };
refresh();
const socket = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws`);
socket.onopen = () => { connection.textContent = 'Live connection'; };
socket.onclose = () => { connection.textContent = 'Connection closed'; };
socket.onmessage = refresh;
