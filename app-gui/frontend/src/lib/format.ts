import type { SequenceInfo } from './types';
import { t } from './i18n';

const pad = (n: number, w = 2) => String(Math.floor(n)).padStart(w, '0');

export function tc(sec: number, seq: SequenceInfo | undefined, frames = true): string {
  if (!seq || seq.fpsNum <= 0 || seq.fpsDen <= 0 || !Number.isFinite(sec)) return frames ? '--:--:--:--' : '--:--:--';
  const rate = seq.fpsNum / seq.fpsDen;
  const fps = Math.round(rate);
  const total = Math.round((sec + seq.startSec) * rate);
  const f = total % fps;
  const s = Math.floor(total / fps);
  const hms = `${pad(s / 3600)}:${pad((s / 60) % 60)}:${pad(s % 60)}`;
  return frames ? `${hms}:${pad(f)}` : hms;
}

export function clock(sec: number): string {
  if (!Number.isFinite(sec) || sec < 0) return '—';
  const s = Math.round(sec);
  return `${pad(s / 3600)}:${pad((s / 60) % 60)}:${pad(s % 60)}`;
}

export function short(sec: number): string {
  if (!Number.isFinite(sec) || sec < 0) return '--:--';
  const s = Math.round(sec);
  return s >= 3600 ? `${Math.floor(s / 3600)}:${pad((s / 60) % 60)}:${pad(s % 60)}` : `${pad(s / 60)}:${pad(s % 60)}`;
}

export function human(sec: number): string {
  if (!Number.isFinite(sec) || sec < 0) return '—';
  const s = Math.round(sec);
  const u = (k: string, n: number) => `${n} ${t(`unit.${k}`)}`;
  if (s < 60) return u('s', s);
  if (s < 3600) return u('min', Math.floor(s / 60)) + (s % 60 ? ` ${u('s', s % 60)}` : '');
  const m = Math.floor(s / 60) % 60;
  return u('h', Math.floor(s / 3600)) + (m ? ` ${u('min', m)}` : '');
}

export function bytes(n: number): string {
  const u = ['B', 'KB', 'MB', 'GB', 'TB'].map((k) => t(`unit.${k}`));
  if (!Number.isFinite(n) || n <= 0) return `0 ${u[0]}`;
  let i = 0;
  while (n >= 1024 && i < u.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n >= 10 || i === 0 ? Math.round(n) : n.toFixed(1)} ${u[i]}`;
}

export function baseName(p: string): string {
  const m = /[^\\/]+$/.exec(p ?? '');
  return m ? m[0] : p;
}

export function dirName(p: string): string {
  const i = Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\'));
  return i > 0 ? p.slice(0, i) : p;
}

export function recorded(iso: string | undefined): string {
  if (!iso) return '';
  const m = /^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2})/.exec(iso);
  return m ? `${m[1]} ${m[2]}` : iso;
}

export function fixed(x: number | undefined, n = 1): string {
  if (x === undefined || x === null || !Number.isFinite(x)) return '';
  return String(Number(x.toFixed(n)));
}
