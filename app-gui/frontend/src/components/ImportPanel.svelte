<script lang="ts">
  import Icon from './Icon.svelte';
  import { app } from '../lib/state.svelte';
  import { t } from '../lib/i18n';
  import { baseName } from '../lib/format';

  const info = $derived(app.info);
  const seq = $derived(info?.sequences[0]);
  const counts = $derived.by(() => {
    const c = { ok: 0, missing: 0, offline: 0, nomedia: 0 };
    for (const f of info?.files ?? []) c[f.state]++;
    return c;
  });
  const locked = $derived(app.phase === 'syncing');
  const busy = $derived(!!app.busy);
</script>

<section class="section">
  {#if locked}<div class="lock"><Icon name="lock" size={14} />{t('import.locked')}</div>{/if}
  <div class="section" class:ro={locked}>
    <h2 class="h">{t('import.timeline')}</h2>
    {#if info}
      <div class="file">
        <Icon name="file" color="#4cc3bd" />
        <div style="min-width:0">
          <span class="mono name">{baseName(info.path)}</span>
          <small>{t('import.fileMeta', { format: t(`format.${info.format}`), fps: seq?.fps ?? '?', tracks: seq?.tracks ?? 0, files: info.files.length })}</small>
          <button class="link" style="height:auto;padding:4px 0 0" disabled={busy} onclick={() => app.openDialog()}>{t('import.replace')}</button>
        </div>
      </div>
    {:else}
      <button class="drop" disabled={busy} onclick={() => app.openDialog()}>{t('import.drop')} <span>{t('import.browse')}</span></button>
      <small class="hint">{t('import.exportHint')}</small>
      {#if app.openError}
        {@const e = app.openError}
        <div class="notice err" role="alert">
          <div>
            <b>{t(`error.${e.kind}.title`)}</b>{t(`error.${e.kind}.hint`)}
            <details><summary>{t('error.details')}</summary><span class="mono" style="font-size:11px">{e.message}</span></details>
          </div>
        </div>
      {/if}
    {/if}

    <h2 class="h">{t('import.folders')}</h2>
    {#if app.folders.length === 0}
      <small class="hint" style="font-size:12px">{t('import.foldersHint')}</small>
    {/if}
    {#each app.folders as f (f)}
      <div class="folder mono" title={f}>
        <Icon name="folder" size={16} color="#9aa3a1" /><span>{f}</span>
        <button class="x" aria-label={t('import.removeFolder')} title={t('import.removeFolder')} disabled={busy} onclick={() => app.removeFolder(f)}>×</button>
      </div>
    {/each}
    <button class="link" disabled={busy} onclick={() => app.addFolder()}>{t('import.addFolder')}</button>

    {#if info}
      <h2 class="h">{t('import.status')}</h2>
      <div class="media-bar" aria-hidden="true">
        {#if counts.ok}<div style:flex={counts.ok} style="background:var(--green)"></div>{/if}
        {#if counts.missing}<div style:flex={counts.missing} style="background:var(--amber)"></div>{/if}
        {#if counts.offline}<div style:flex={counts.offline} style="background:var(--gray)"></div>{/if}
        {#if counts.nomedia}<div style:flex={counts.nomedia} style="background:#3a3d41"></div>{/if}
      </div>
      <div class="media-count">
        <span><b style="color:var(--green)">{counts.ok}</b> {t('legend.found')}</span>
        {#if counts.missing}<span><b style="color:var(--amber)">{counts.missing}</b> {t('legend.missing')}</span>{/if}
        {#if counts.offline}<span><b style="color:#a3a9ae">{counts.offline}</b> {t('state.offline')}</span>{/if}
        {#if counts.nomedia}<span><b style="color:#a3a9ae">{counts.nomedia}</b> {t('state.nomedia')}</span>{/if}
      </div>
      {#if counts.missing > 0 && !locked}
        <div class="notice warn">
          <div>
            <b>{t('import.missingTitle', { n: counts.missing })}</b>{t('import.missingText')}<br />
            <button class="link" disabled={busy} onclick={() => app.locate()}>{t('import.locate')}</button>
          </div>
        </div>
      {/if}
    {/if}
  </div>
</section>
