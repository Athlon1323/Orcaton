<script lang="ts">
  import { app } from '../lib/state.svelte';
  import { t } from '../lib/i18n';
  import { bytes, dirName } from '../lib/format';

  const a = $derived(app.about);
  const c = $derived(app.cliTool);
</script>

<section class="about">
  <h2 class="h">{t('about.title')}</h2>
  <div class="kv">
    <span>{t('about.version')}</span>
    <span class="mono">{a?.version ?? app.startup?.version ?? '…'} · {t('about.api', { v: a?.apiVersion ?? app.startup?.apiVersion ?? '?' })}</span>
    <span>{t('about.ffmpeg')}</span>
    {#if a?.ffmpegError}
      <span style="color:var(--rose)">{t('about.notFound')}</span>
    {:else}
      <span class="mono" title={a?.ffmpeg.ffmpeg}>{a ? `${a.ffmpeg.version} · ${dirName(a.ffmpeg.ffmpeg)}` : '…'}{#if a?.ffmpeg.old}<span style="color:var(--amber, #d9a441)"> · ⚠ {t('about.ffmpegOld', { min: a.ffmpeg.min })}</span>{/if}</span>
    {/if}
    <span>{t('about.cache')}</span>
    <span class="mono" title={a?.cache.path}>{a ? t('about.cacheValue', { size: bytes(a.cache.bytes), n: a.cache.entries }) : '…'}</span>
    <span>{t('about.settings')}</span>
    <span class="mono">{a?.settingsPath || '—'}</span>
    {#if c?.available}
      <span>{t('about.cli')}</span>
      <span>
        {#if c.installed}
          <span class="mono">✓ {c.link}</span>
        {:else if c.problem}
          <span style="color:var(--amber, #d9a441)">⚠ {t(`about.cliProblem.${c.problem}`)}</span>
        {:else if c.other}
          <span class="mono">{t('about.cliOther', { path: c.other })}</span>
        {:else}
          <span style="color:var(--muted)">{t('about.cliNotInstalled')}</span>
        {/if}
        {#if c.installed}
          <button class="link inline" style="display:block;text-align:left" disabled={!!app.busy} onclick={() => app.removeCliTool()}>{t('about.cliRemove')}</button>
        {:else if !c.problem}
          <button class="link inline" style="display:block;text-align:left" disabled={!!app.busy} onclick={() => app.installCliTool()}>{t('about.cliInstall')}</button>
          <small style="display:block;color:var(--muted)">{t('about.cliHint')}</small>
        {/if}
      </span>
    {/if}
  </div>
  <button class="link" disabled={!!app.busy || !a?.cache.entries} onclick={() => app.clearCache()}>{t('about.clear')}</button>
  <small style="color:var(--muted)">{t('about.cleanroom')}</small>
  <h2 class="h">{t('about.licenses')}</h2>
  <small style="color:var(--muted)">{t('about.licensesText')}</small>
</section>
