<script lang="ts">
  import { app } from '../lib/state.svelte';
  import { t } from '../lib/i18n';
  import { bytes } from '../lib/format';

  let box: HTMLDivElement | undefined = $state();
  const tail = $derived(app.log.slice(-300));
  $effect(() => {
    void tail.length;
    if (box) box.scrollTop = box.scrollHeight;
  });
</script>

<section class="section">
  <h2 class="h">{t('dev.log')}</h2>
  <div class="log" bind:this={box}>{tail.length ? tail.join('\n') : t('dev.empty')}</div>
  <button class="link" disabled={!app.log.length} onclick={() => app.copy(app.log.join('\n'))}>{t('dev.copy')}</button>
  <h2 class="h">{t('dev.logs')}</h2>
  <small class="hint mono">{app.about?.logDir || app.startup?.logDir || '—'}</small>
  <button class="link" onclick={() => app.openLogs()}>{t('dev.openLogs')}</button>
  <h2 class="h">{t('dev.cache')}</h2>
  <div class="media-count">
    <span class="mono">{app.about ? t('about.cacheValue', { size: bytes(app.about.cache.bytes), n: app.about.cache.entries }) : '…'}</span>
  </div>
  <button class="link" disabled={!!app.busy || !app.about?.cache.entries} onclick={() => app.clearCache()}>{t('about.clear')}</button>
</section>
