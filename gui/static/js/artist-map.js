import { API } from './api.js';

export async function loadArtistNameMap(limit = 5000) {
  const rows = await API.getArtists({ limit, offset: 0, sort: 'name' });
  const map = new Map();
  for (const row of Array.isArray(rows) ? rows : []) {
    const id = String(row.id || row.ID || '').trim();
    if (!id) continue;
    const name = String(row.name || row.Name || '').trim();
    if (!name) continue;
    map.set(id, name);
  }
  return map;
}

export function resolveArtistName(track, artistMap) {
  const direct = String(track.artist || track.Artist || track.artist_name || track.artistName || '').trim();
  if (direct) return direct;
  const artistID = String(track.artistId || track.ArtistID || track.artist_id || '').trim();
  if (!artistID) return '';
  return artistMap.get(artistID) || '';
}
