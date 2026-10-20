<script lang="ts">
  import { app } from '../lib/state.svelte';
  import { t } from '../lib/i18n';
</script>

<div class="toasts" aria-live="polite">
  {#each app.toasts as x (x.id)}
    <div class="toast {x.level}" role={x.level === 'err' ? 'alert' : 'status'}>
      <button class="close" aria-label={t('toast.close')} onclick={() => app.dismiss(x.id)}>×</button>
      <b>{x.title}</b>
      {#if x.text}<span class="mono" style="font-family:inherit;word-break:break-word">{x.text}</span>{/if}
      {#if x.actions?.length}
        <div class="acts">{#each x.actions as a (a.label)}<button class="link inline" onclick={a.run}>{a.label}</button>{/each}</div>
      {/if}
    </div>
  {/each}
</div>
