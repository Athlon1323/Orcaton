<script lang="ts">
  import { app } from '../lib/state.svelte';
  import { t } from '../lib/i18n';

  const os = $derived(app.ffmpegPlan?.os ?? app.startup?.os ?? 'windows');
  const plan = $derived(app.ffmpegPlan);
  const installing = $derived(app.busy === 'ffmpeg');
  const command = $derived(plan?.command || t(`ffmpeg.cmd.${os === 'darwin' || os === 'linux' ? os : 'windows'}`));
  const via = $derived(plan?.tool === 'winget' ? 'winget' : plan?.tool === 'brew' ? 'brew' : 'linux');
</script>

<div class="block" role="alert">
  <div class="dialog">
    <h3>{installing ? t('ffmpeg.installing') : t('ffmpeg.title')}</h3>
    <p>{t('ffmpeg.text')}</p>
    <pre class="cmd">{command}</pre>
    {#if plan?.button}
      <p class="note">{t(`ffmpeg.via.${via}`)}</p>
    {:else if plan?.problem}
      <p class="note">{t(`ffmpeg.problem.${plan.problem}`)}</p>
    {/if}
    <p class="note">{t('ffmpeg.env')}</p>
    {#if installing || app.ffmpegLines.length}
      <pre class="details" aria-live="polite">{app.ffmpegLines.slice(-10).join('\n') || '…'}</pre>
    {/if}
    {#if app.ffmpeg?.message && !installing}<details><summary style="cursor:pointer;color:var(--muted);font-size:12px">{t('error.details')}</summary><pre class="details">{app.ffmpeg.message}{app.ffmpeg.hint ? `\n${app.ffmpeg.hint}` : ''}</pre></details>{/if}
    <div class="dlg-actions">
      {#if installing}
        <button class="btn" onclick={() => app.stop()}>{t('ffmpeg.stop')}</button>
      {:else}
        {#if plan?.button}<button class="btn primary" onclick={() => app.installFFmpeg()} disabled={!!app.busy}>{t('ffmpeg.install')}</button>{/if}
        <button class="btn" class:primary={!plan?.button} onclick={() => app.checkFFmpeg()} disabled={!!app.busy}>{t('ffmpeg.check')}</button>
      {/if}
    </div>
  </div>
</div>

<style>
  .note { font-size: 12px; color: var(--muted); margin: 0; }
</style>
