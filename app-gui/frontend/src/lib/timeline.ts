import type { Camera, FileResult, MediaFile, ProjectInfo } from './types';

export type ClipState =
  | 'audio' | 'clock' | 'unmatched' | 'offline'
  | 'pending' | 'missing' | 'analyzed' | 'active';

export interface Clip {
  key: string;
  name: string;
  left: number;
  width: number;
  state: ClipState;
}

export interface GroupBar {
  group: number;
  left: number;
  width: number;
}

export interface Row {
  camera: Camera;
  name: string;
  tracks: string;
  count: number;
  clips: Clip[];
  groups: GroupBar[];
}

export interface Layout {
  rows: Row[];
  spanSec: number;
}

export function fileKey(f: { camera: string; name: string; startSec: number }): string {
  return `${f.camera}|${f.name}|${f.startSec.toFixed(3)}`;
}

export function splitTitle(c: Camera): { name: string; tracks: string } {
  const i = c.title.lastIndexOf(' · ');
  if (i < 0) return { name: c.title || c.folder || c.lane, tracks: c.tracks.join('+') || c.lane };
  return { name: c.title.slice(0, i), tracks: c.title.slice(i + 3) };
}

export interface SyncMarks {
  analyzed: Set<string>;
  active: string;
}

function beforeState(f: MediaFile, marks?: SyncMarks): ClipState {
  if (f.state === 'offline' || f.state === 'nomedia') return 'offline';
  if (marks) {
    if (f.name === marks.active) return 'active';
    if (marks.analyzed.has(f.name)) return 'analyzed';
    return 'pending';
  }
  return f.state === 'missing' ? 'missing' : 'pending';
}

export function layoutBefore(info: ProjectInfo, marks?: SyncMarks): Layout {
  const seq = info.sequences[0];
  let span = seq?.durationSec ?? 0;
  for (const f of info.files) span = Math.max(span, f.startSec + f.durationSec);
  span = Math.max(span, 1);
  const rows = info.cameras.map((camera) => {
    const files = info.files.filter((f) => f.camera === camera.id);
    const clips = files.map((f) => ({
      key: fileKey(f), name: f.name, left: (100 * f.startSec) / span,
      width: (100 * f.durationSec) / span, state: beforeState(f, marks),
    }));
    return { camera, ...splitTitle(camera), count: files.length, clips, groups: [] };
  });
  return { rows, spanSec: span };
}

export function layoutAfter(info: ProjectInfo, files: FileResult[], hidden: Partial<Record<string, boolean>> = {}): Layout {
  let span = 1;
  for (const f of files) if (f.positionSec != null) span = Math.max(span, f.positionSec + f.durationSec);
  const rows = info.cameras.map((camera) => {
    const mine = files.filter((f) => f.camera === camera.id);
    const clips: Clip[] = [];
    const gs = new Map<number, [number, number]>();
    for (const f of mine) {
      if (f.positionSec == null || hidden[f.result]) continue;
      const l = f.positionSec;
      const r = f.positionSec + f.durationSec;
      clips.push({ key: fileKey(f), name: f.name, left: (100 * l) / span, width: (100 * f.durationSec) / span, state: f.result });
      if (f.group >= 0) {
        const g = gs.get(f.group);
        gs.set(f.group, g ? [Math.min(g[0], l), Math.max(g[1], r)] : [l, r]);
      }
    }
    const groups = [...gs.entries()].map(([group, [l, r]]) => ({ group, left: (100 * l) / span, width: (100 * (r - l)) / span }));
    return { camera, ...splitTitle(camera), count: mine.length, clips, groups };
  });
  return { rows, spanSec: span };
}

export function ticks(spanSec: number, n = 8): number[] {
  return Array.from({ length: n + 1 }, (_, i) => (spanSec * i) / n);
}

export const THIN_PX = 44;
