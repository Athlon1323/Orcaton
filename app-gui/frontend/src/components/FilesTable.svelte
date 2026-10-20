<script lang="ts">
  import { app } from '../lib/state.svelte';
  import { has, t } from '../lib/i18n';
  import { clock, fixed, recorded, tc } from '../lib/format';
  import { fileKey, splitTitle } from '../lib/timeline';
  import { RESULTS, RESULT_COLOR, RESULT_ICON, STATE_COLOR } from '../lib/results';
  import type { FileResult, MediaFile } from '../lib/types';

  const ROW = 34;
  let scrollTop = $state(0);
  let height = $state(600);
  let sortKey = $state('');
  let asc = $state(true);

  const info = $derived(app.info!);
  const seq = $derived(info.sequences[0]);
  const after = $derived(!!app.result && app.phase === 'done');
  const camName = $derived(new Map(info.cameras.map((c) => [c.id, splitTitle(c).name])));
  const camOrder = $derived(new Map(info.cameras.map((c, i) => [c.id, i])));
  const reason = (f: FileResult) => (f.reasonCode && has(`reason.${f.reasonCode}`) ? t(`reason.${f.reasonCode}`) : (f.reason ?? ''));

  interface Col { key: string; label: string; w: string; get: (x: never) => string | number }
  const beforeCols: Col[] = [
    { key: 'state', label: 'col.media', w: '130px', get: (f: MediaFile) => f.state },
    { key: 'camera', label: 'col.camera', w: '120px', get: (f: MediaFile) => camOrder.get(f.camera) ?? 0 },
    { key: 'name', label: 'col.file', w: 'minmax(180px,1.2fr)', get: (f: MediaFile) => f.name.toLowerCase() },
    { key: 'start', label: 'col.start', w: '110px', get: (f: MediaFile) => f.startSec },
    { key: 'dur', label: 'col.duration', w: '90px', get: (f: MediaFile) => f.durationSec },
    { key: 'audio', label: 'col.audio', w: '64px', get: (f: MediaFile) => (f.hasAudio ? 1 : 0) },
    { key: 'path', label: 'col.path', w: 'minmax(200px,2fr)', get: (f: MediaFile) => f.path.toLowerCase() },
  ];
  const afterCols: Col[] = [
    { key: 'result', label: 'col.result', w: '160px', get: (f: FileResult) => RESULTS.indexOf(f.result) },
    { key: 'camera', label: 'col.camera', w: '110px', get: (f: FileResult) => camOrder.get(f.camera) ?? 0 },
    { key: 'name', label: 'col.file', w: 'minmax(160px,1.2fr)', get: (f: FileResult) => f.name.toLowerCase() },
    { key: 'pos', label: 'col.position', w: '110px', get: (f: FileResult) => f.positionSec ?? Infinity },
    { key: 'dur', label: 'col.duration', w: '84px', get: (f: FileResult) => f.durationSec },
    { key: 'partner', label: 'col.partner', w: 'minmax(120px,1fr)', get: (f: FileResult) => (f.partner ?? '').toLowerCase() },
    { key: 'z', label: 'col.z', w: '72px', get: (f: FileResult) => f.confidence ?? 0 },
    { key: 'rec', label: 'col.recorded', w: '164px', get: (f: FileResult) => f.recordStart ?? '' },
    { key: 'reason', label: 'col.reason', w: 'minmax(160px,1.4fr)', get: (f: FileResult) => reason(f) },
  ];
  const cols = $derived(after ? afterCols : beforeCols);
  const grid = $derived(cols.map((c) => c.w).join(' '));

  const rows = $derived.by(() => {
    const q = app.query.trim().toLowerCase();
    let list: (MediaFile | FileResult)[] = after
      ? app.result!.files.filter((f) => !app.hidden[f.result])
      : info.files.slice();
    if (q) list = list.filter((f) => f.name.toLowerCase().includes(q));
    const col = cols.find((c) => c.key === sortKey);
    if (col) {
      const get = col.get as (x: MediaFile | FileResult) => string | number;
      list = list.slice().sort((a, b) => {
        const x = get(a), y = get(b);
        return (x < y ? -1 : x > y ? 1 : 0) * (asc ? 1 : -1);
      });
    }
    return list;
  });
  const total = $derived(after ? app.result!.files.length : info.files.length);
  const first = $derived(Math.max(0, Math.floor(scrollTop / ROW) - 8));
  const last = $derived(Math.min(rows.length, Math.ceil((scrollTop + height) / ROW) + 8));
  const visible = $derived(rows.slice(first, last));

  function sortBy(k: string) {
    if (sortKey === k) asc = !asc;
    else { sortKey = k; asc = true; }
  }
</script>

{#if after}
  <div class="filters" role="group" aria-label={t('table.filter')}>
    {#each RESULTS as r (r)}
      <button class="chipf" aria-pressed={!app.hidden[r]} onclick={() => app.toggleHidden(r)}>
        <i class="dot" style:background={RESULT_COLOR[r]}></i>{t(`result.${r}`)} · {app.result!.summary[r]}
      </button>
    {/each}
    <input class="search" type="search" placeholder={t('table.search')} bind:value={app.query} />
  </div>
{/if}
<div class="vt" bind:clientHeight={height} onscroll={(e) => (scrollTop = e.currentTarget.scrollTop)} role="table">
  <div class="row head" style:grid-template-columns={grid} role="row">
    {#each cols as c (c.key)}
      <button role="columnheader" onclick={() => sortBy(c.key)}>{t(c.label)}{sortKey === c.key ? (asc ? ' ▴' : ' ▾') : ''}</button>
    {/each}
  </div>
  <div style:height="{rows.length * ROW}px" style="position:relative">
    <div style:transform="translateY({first * ROW}px)">
      {#each visible as f (fileKey(f))}
        {@const key = fileKey(f)}
        <div class="row body" class:sel={app.selected === key} style:grid-template-columns={grid} role="row"
          onclick={() => (app.selected = key)} onkeydown={(e) => e.key === 'Enter' && (app.selected = key)} tabindex="-1">
          {#if after}
            {@const r = f as FileResult}
            <span><span class="st" style:color={RESULT_COLOR[r.result]}>{RESULT_ICON[r.result]} {t(`result.${r.result}`)}</span></span>
            <span>{camName.get(r.camera) ?? r.lane}</span>
            <span class="mono" title={r.path}>{r.name}</span>
            <span class="mono">{r.positionSec != null ? (r.positionTC ?? tc(r.positionSec, seq)) : t('table.removed')}</span>
            <span class="mono">{clock(r.durationSec)}</span>
            <span class="mono">{r.partner ?? ''}</span>
            <span class="mono">{fixed(r.confidence)}</span>
            <span class="mono">{recorded(r.recordStart)}</span>
            <span title={r.reason}>{reason(r)}</span>
          {:else}
            {@const m = f as MediaFile}
            <span><span class="st" style:color={STATE_COLOR[m.state]}>● {t(`state.${m.state}`)}</span></span>
            <span>{camName.get(m.camera) ?? m.lane}</span>
            <span class="mono">{m.name}</span>
            <span class="mono">{m.startTC}</span>
            <span class="mono">{clock(m.durationSec)}</span>
            <span>{m.hasAudio ? t('table.yes') : t('table.no')}</span>
            <span class="mono dim" title={m.originalPath ?? m.path}>{m.path}</span>
          {/if}
        </div>
      {/each}
    </div>
  </div>
</div>
<div class="tfoot">{t('table.showing', { n: rows.length, total })}</div>
