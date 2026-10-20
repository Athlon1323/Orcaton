<script lang="ts">
  import Brand from './Brand.svelte';
  import { app } from '../lib/state.svelte';
  import { t } from '../lib/i18n';
  import { SYNC_STAGES, leftSec } from '../lib/progress';
  import { human, short } from '../lib/format';
  import { RESULTS, RESULT_COLOR } from '../lib/results';

  const p = $derived(app.progress);
  const n = SYNC_STAGES.length;
  const elapsed = $derived(Math.max(0, (app.now - p.startedAt) / 1000));
  const left = $derived(leftSec(p, app.now));
  const pct = $derived(Math.floor(p.percent));
  const formats = $derived(app.info?.outputFormats ?? []);
  const extOf = (f: { extensions: string[] }) => f.extensions[0] ?? '';
  const primary = $derived(formats.find((f) => f.name === app.info?.format) ?? formats[0]);
  const others = $derived(formats.filter((f) => f !== primary));
  const saving = $derived(app.busy === 'save');
</script>

<header class="top">
  <Brand />
  <div class="progress">
    {#if app.phase === 'done' && app.result}
      {@const s = app.result.summary}
      <div class="summary">
        <span class="big">{t('top.done', { t: human(s.seconds) })}</span>
        {#each RESULTS as r (r)}
          <span><i class="dot" style:background={RESULT_COLOR[r]}></i>{t(`result.${r}`)} <b class="mono" style="color:var(--text)">{s[r]}</b></span>
        {/each}
        <div class="save">
          {#if primary}
            <button class="btn primary" disabled={saving} onclick={() => app.saveTimeline(primary.name)}>{t('top.saveTimeline', { ext: extOf(primary) })}</button>
          {/if}
          {#each others as f (f.name)}
            <button class="btn" disabled={saving} onclick={() => app.saveTimeline(f.name)}>{t('top.saveAs', { ext: extOf(f) })}</button>
          {/each}
          <button class="btn" disabled={saving} onclick={() => app.saveReport()}>{t('top.saveReport')}</button>
        </div>
      </div>
    {:else if app.phase === 'syncing'}
      {#if p.index < 0}
        <div class="progress-head">
          <span class="status">{p.stage === 'search' ? t('top.searching') : t('top.syncing')}</span>
          {#if p.stage === 'search' && p.total > 0}<span class="what">{t('top.count', { done: p.done, total: p.total })}</span>{/if}
        </div>
        <div class="bar indeterminate" role="progressbar" aria-label={t('top.progress')}><div></div></div>
      {:else}
        <div class="progress-head">
          <span class="status">{t('top.syncing')}</span>
          <span class="what">
            {t('top.step', { n: p.index + 1, total: n, title: t(`stage.${p.stage}`) })}
            {#if p.file}— <span class="mono" style="color:var(--text)">{p.file}</span>{/if}
            {#if p.total > 1 && (p.stage === 'features' || p.stage === 'record_time')}{t('top.count', { done: p.done, total: p.total })}{/if}
          </span>
          <span class="pct mono">{pct}%</span>
        </div>
        <div class="bar" role="progressbar" aria-label={t('top.progress')} aria-valuenow={pct} aria-valuemin="0" aria-valuemax="100"><div style:width="{p.percent}%"></div></div>
      {/if}
      <div class="stages">
        {#each SYNC_STAGES as st, i (st)}
          <span class={i < p.index ? 'done' : i === p.index ? 'now' : ''}>{i < p.index ? '✓ ' : ''}{t(`stage.${st}`)}</span>
        {/each}
      </div>
    {:else}
      <div class="progress-head">
        {#if app.phase === 'cancelled'}
          <span class="status cancel">{t('top.cancelled')}</span>
          <span class="what">{p.index >= 0 ? t('top.stoppedAt', { n: p.index + 1, total: n, title: t(`stage.${p.stage}`) }) : t('top.stoppedEarly')}</span>
        {:else}
          <span class="status ready">{t('top.ready')}</span>
          <span class="what" style:color={app.info ? 'var(--text-2)' : 'var(--muted)'}>
            {app.info ? t('top.readyProject', { files: app.info.files.length, cameras: app.info.cameras.length }) : t('top.openToStart')}
          </span>
        {/if}
      </div>
      <div class="segs">
        {#each SYNC_STAGES as st, i (st)}<i class={app.phase === 'cancelled' && i < p.index ? 'done' : ''}></i>{/each}
      </div>
      <div class="stages">
        {#each SYNC_STAGES as st (st)}<span>{t(`stage.${st}`)}</span>{/each}
      </div>
    {/if}
  </div>
  {#if app.phase === 'syncing'}
    <div class="time mono">{t('top.elapsed', { t: short(elapsed) })}<br />{left != null ? t('top.left', { t: short(left) }) : ''}</div>
  {:else if app.phase === 'cancelled'}
    <div class="time mono">{short(elapsed)}</div>
  {/if}
</header>
