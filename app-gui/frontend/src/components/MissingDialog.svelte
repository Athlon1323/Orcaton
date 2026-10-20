<script lang="ts">
  import { app } from '../lib/state.svelte';
  import { t } from '../lib/i18n';

  const FIRST = 15;
  let expanded = $state(false);
  const m = $derived(app.missing);
  const shown = $derived(m ? (expanded ? m.files : m.files.slice(0, FIRST)) : []);
  $effect(() => {
    if (!m) expanded = false;
  });
  function onkey(e: KeyboardEvent) {
    if (m && e.key === 'Escape') app.answer('cancel');
  }
</script>

<svelte:window onkeydown={onkey} />
{#if m}
  <div class="overlay">
    <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="dlgT">
      <h3 id="dlgT">{t('missing.title', { n: m.files.length })}</h3>
      <p>{t('missing.text')}</p>
      {#if m.note}<div class="notice warn"><div>{m.note}</div></div>{/if}
      <div class="mlist">
        {#each shown as f, i (i)}
          <div><span class="mono">{f.name}</span><small>{f.originalPath || f.path}</small></div>
        {/each}
        {#if !expanded && m.files.length > FIRST}
          <button class="more" onclick={() => (expanded = true)}>{t('missing.more', { n: m.files.length - FIRST })}</button>
        {/if}
      </div>
      <div class="dlg-actions">
        <span class="hint">{t('missing.hint')}</span>
        <button class="btn ghost" onclick={() => app.answer('cancel')}>{t('missing.cancel')}</button>
        <button class="btn" onclick={() => app.answer('skip')}>{t('missing.skip')}</button>
        <button class="btn primary" onclick={() => app.answerFolder()}>{t('missing.choose')}</button>
      </div>
    </div>
  </div>
{/if}
