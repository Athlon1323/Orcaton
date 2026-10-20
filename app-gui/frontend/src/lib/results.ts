import type { FileResultKind } from './types';

export const RESULTS: FileResultKind[] = ['audio', 'clock', 'unmatched', 'offline'];

export const RESULT_COLOR: Record<FileResultKind, string> = {
  audio: 'var(--green)', clock: 'var(--amber)', unmatched: 'var(--rose)', offline: 'var(--offline)',
};

export const RESULT_ICON: Record<FileResultKind, string> = { audio: '✓', clock: '◷', unmatched: '✕', offline: '⌀' };

export const STATE_COLOR: Record<string, string> = {
  ok: 'var(--green)', missing: 'var(--amber)', offline: 'var(--offline)', nomedia: 'var(--gray)',
};
