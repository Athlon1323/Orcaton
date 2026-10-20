<script lang="ts">
  import Icon from './Icon.svelte';
  import Timeline from './Timeline.svelte';
  import FilesTable from './FilesTable.svelte';
  import ClipCard from './ClipCard.svelte';
  import FfmpegBanner from './FfmpegBanner.svelte';
  import { app } from '../lib/state.svelte';
  import { t } from '../lib/i18n';
  import { clock } from '../lib/format';
  import { RESULTS, RESULT_COLOR } from '../lib/results';

  const info = $derived(app.info);
  const seq = $derived(info?.sequences[0]);
  const done = $derived(app.phase === 'done' && !!app.result);
  const counts = $derived.by(() => {
    const c = { ok: 0, missing: 0, offline: 0, nomedia: 0 };
    for (const f of info?.files ?? []) c[f.state]++;
    return c;
  });
  const syncCounts = $derived.by(() => {
    if (!info || app.phase !== 'syncing') return null;
    let offline = 0;
    for (const f of info.files) if (f.state === 'offline' || f.state === 'nomedia') offline++;
    const analyzed = Math.max(0, app.analyzed.size - (app.active ? 1 : 0));
    return { analyzed, active: app.active ? 1 : 0, waiting: Math.max(0, info.files.length - offline - analyzed - (app.active ? 1 : 0)), offline };
  });
  const zoomBy = (k: number) => (app.zoom = Math.min(64, Math.max(1, app.zoom * k)));
</script>

<main class="main">
  {#if !info}
    <div class="content">
      <div class="empty">
        <button class="dropzone" disabled={!!app.busy} onclick={() => app.openDialog()}>
          <Icon name="import" size={44} color="var(--teal)" />
          <h3>{t('main.openTitle')}</h3>
          <p>{t('main.openText')} <span class="link inline">{t('main.openBrowse')}</span></p>
          <p>{t('main.openAbout')}</p>
        </button>
      </div>
    </div>
  {:else}
    <div class="toolbar">
      <span class="title">{seq?.name ?? info.name}</span>
      <span class="meta mono">{t('view.meta', { fps: seq?.fps ?? '?', duration: clock(seq?.durationSec ?? 0), cameras: info.cameras.length, files: info.files.length })}</span>
      <div class="seg first" role="group" aria-label={t('view.label')}>
        <button aria-pressed={app.view === 'timeline'} onclick={() => app.setView('timeline')}>{t('view.timeline')}</button>
        <button aria-pressed={app.view === 'files'} onclick={() => app.setView('files')}>{t('view.files')}</button>
      </div>
      {#if app.view === 'timeline'}
        <div class="seg" role="group" aria-label={t('view.beforeAfter')}>
          <button aria-pressed={app.ba === 'before' || !done} onclick={() => (app.ba = 'before')}>{t('view.before')}</button>
          <button aria-pressed={app.ba === 'after' && done} disabled={!done} onclick={() => (app.ba = 'after')}>{t('view.after')}</button>
        </div>
        <div class="zoom">
          <button aria-label={t('view.zoomOut')} title={t('view.zoomOut')} disabled={app.zoom <= 1} onclick={() => zoomBy(1 / 1.6)}>−</button>
          <button aria-label={t('view.zoomIn')} title={t('view.zoomIn')} onclick={() => zoomBy(1.6)}>+</button>
          <button class="fit" onclick={() => (app.zoom = 1)}>{t('view.fit')}</button>
        </div>
      {/if}
    </div>
    <div class="content">
      {#if app.view === 'timeline'}<Timeline />{:else}<FilesTable />{/if}
    </div>
    {#if app.view === 'timeline' || !done}
      <div class="legend">
        {#if syncCounts}
          <span><i class="s-analyzed"></i>{t('legend.analyzed')} · {syncCounts.analyzed}</span>
          <span><i class="s-active"></i>{t('legend.active')} · {syncCounts.active}</span>
          <span><i class="s-pending"></i>{t('legend.waiting')} · {syncCounts.waiting}</span>
          <span><i class="s-offline"></i>{t('legend.offline')} · {syncCounts.offline}</span>
        {:else if done && app.ba === 'after'}
          {#each RESULTS as r (r)}
            <button class="chip" aria-pressed={!app.hidden[r]} onclick={() => app.toggleHidden(r)}>
              <i style:background={RESULT_COLOR[r]}></i>{t(`result.${r}`)} · {app.result!.summary[r]}
            </button>
          {/each}
          <span><i style="background:rgba(76,195,189,.55);height:3px;width:16px"></i>{t('legend.group')}</span>
          <span class="tip">{t('view.clipTip')}</span>
        {:else}
          <span><i class={app.view === 'timeline' ? 's-pending' : ''} style:background={app.view === 'files' ? 'var(--green)' : null}></i>{t('legend.found')} · {counts.ok}</span>
          {#if counts.missing}<span><i class={app.view === 'timeline' ? 's-missing' : ''} style:background={app.view === 'files' ? 'var(--amber)' : null}></i>{t('legend.missing')} · {counts.missing}</span>{/if}
          {#if counts.offline + counts.nomedia}<span><i class="s-offline"></i>{t('state.offline')} · {counts.offline + counts.nomedia}</span>{/if}
        {/if}
      </div>
    {/if}
    {#if app.selected}<ClipCard />{/if}
  {/if}
  {#if app.ffmpeg}<FfmpegBanner />{/if}
</main>
