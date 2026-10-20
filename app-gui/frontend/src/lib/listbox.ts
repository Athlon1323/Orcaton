
export function moveIndex(key: string, cur: number, n: number): number | null {
  if (n <= 0) return null;
  switch (key) {
    case 'ArrowDown': return cur < 0 ? 0 : Math.min(cur + 1, n - 1);
    case 'ArrowUp': return cur < 0 ? n - 1 : Math.max(cur - 1, 0);
    case 'Home': case 'PageUp': return 0;
    case 'End': case 'PageDown': return n - 1;
    default: return null;
  }
}

export function typeAhead(ch: string, cur: number, labels: string[]): number {
  const c = ch.toLocaleLowerCase();
  const n = labels.length;
  for (let i = 1; i <= n; i++) {
    const j = (cur + i + n) % n;
    if (labels[j].toLocaleLowerCase().startsWith(c)) return j;
  }
  return -1;
}

export function placement(top: number, bottom: number, viewport: number, need: number): 'down' | 'up' {
  const below = viewport - bottom;
  return below >= need || below >= top ? 'down' : 'up';
}
