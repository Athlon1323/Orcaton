import type { Stage, SyncEvent } from './types';

export const SYNC_STAGES: Stage[] = ['record_time', 'features', 'pairs', 'refine', 'solve', 'layout'];

export const WEIGHTS: Record<string, number> = {
  record_time: 5, features: 65, pairs: 12, refine: 2, solve: 14, layout: 2,
};

export interface Progress {
  stage: Stage | null;
  index: number;
  done: number;
  total: number;
  file: string;
  percent: number;
  startedAt: number;
}

export function newProgress(now: number): Progress {
  return { stage: null, index: -1, done: 0, total: 0, file: '', percent: 0, startedAt: now };
}

export function percentAt(index: number, frac: number): number {
  if (index < 0) return 0;
  let sum = 0;
  let all = 0;
  SYNC_STAGES.forEach((s, i) => {
    const w = WEIGHTS[s];
    all += w;
    if (i < index) sum += w;
    else if (i === index) sum += w * Math.min(1, Math.max(0, frac));
  });
  return (100 * sum) / all;
}

export function step(p: Progress, ev: SyncEvent): Progress {
  const index = SYNC_STAGES.indexOf(ev.stage);
  if (index < 0) {
    return { ...p, stage: ev.stage, done: ev.done, total: ev.total, file: ev.file ?? p.file };
  }
  const frac = ev.total > 0 ? ev.done / ev.total : 0;
  const percent = Math.max(p.percent, index >= p.index ? percentAt(index, frac) : p.percent);
  return {
    stage: ev.stage, index: Math.max(index, p.index), done: ev.done, total: ev.total,
    file: ev.file ?? (index === p.index ? p.file : ''), percent, startedAt: p.startedAt,
  };
}

export function leftSec(p: Progress, now: number): number | null {
  const el = (now - p.startedAt) / 1000;
  if (el < 3 || p.percent < 2 || p.percent >= 100) return null;
  return (el * (100 - p.percent)) / p.percent;
}
