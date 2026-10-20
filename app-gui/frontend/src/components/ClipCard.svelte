<script lang="ts">
  import { app } from '../lib/state.svelte';
  import { has, t } from '../lib/i18n';
  import { clock, fixed, recorded, tc } from '../lib/format';
  import { fileKey, splitTitle } from '../lib/timeline';
  import { RESULT_COLOR, RESULT_ICON, STATE_COLOR } from '../lib/results';

  const info = $derived(app.info!);
  const seq = $derived(info.sequences[0]);
  const res = $derived(app.phase === 'done' ? app.result?.files.find((f) => fileKey(f) === app.selected) : undefined);
  const media = $derived(res ? undefined : info.files.find((f) => fileKey(f) === app.selected));
  const cam = $derived(info.cameras.find((c) => c.id === (res ?? media)?.camera));
  const camText = $derived(cam ? `${splitTitle(cam).name} · ${splitTitle(cam).tracks}` : '');
</script>

{#if res || media}
  <div class="detail" role="dialog" aria-label={(res ?? media)!.name}>
    <button class="close" aria-label={t('card.close')} onclick={() => (app.selected = null)}>×</button>
    {#if res}
      <h4>{res.name}</h4>
      <span class="st" style:color={RESULT_COLOR[res.result]}>{RESULT_ICON[res.result]} {t(`result.${res.result}`)}</span>
      <div class="kv">
        <span>{t('card.camera')}</span><span>{camText}</span>
        <span>{t('card.position')}</span><span class="mono">{res.positionSec != null ? (res.positionTC ?? tc(res.positionSec, seq)) : t('table.removed')}</span>
        <span>{t('card.before')}</span><span class="mono">{tc(res.startSec, seq)}</span>
        <span>{t('card.duration')}</span><span class="mono">{clock(res.durationSec)}</span>
        {#if res.partner}<span>{t('card.partner')}</span><span class="mono">{res.partner}</span>{/if}
        {#if res.confidence}<span>{t('card.confidence')}</span><span class="mono">{t('card.confValue', { z: fixed(res.confidence), r: fixed(res.ratio) })}</span>{/if}
        {#if res.group >= 0}<span>{t('card.group')}</span><span class="mono">{res.group}</span>{/if}
        {#if res.recordStart}<span>{t('card.recorded')}</span><span class="mono">{recorded(res.recordStart)}</span>{/if}
        {#if res.reasonCode || res.reason}
          <span>{t('card.reason')}</span>
          <span title={res.reason}>{res.reasonCode && has(`reason.${res.reasonCode}`) ? t(`reason.${res.reasonCode}`) : res.reason}</span>
        {/if}
        <span>{t('card.path')}</span><span class="mono" style="color:var(--muted)">{res.path}</span>
      </div>
    {:else if media}
      <h4>{media.name}</h4>
      <span class="st" style:color={STATE_COLOR[media.state]}>● {t(`state.${media.state}`)}</span>
      <div class="kv">
        <span>{t('card.camera')}</span><span>{camText}</span>
        <span>{t('card.before')}</span><span class="mono">{media.startTC}</span>
        <span>{t('card.duration')}</span><span class="mono">{clock(media.durationSec)}</span>
        <span>{t('card.path')}</span><span class="mono" style="color:var(--muted)">{media.path}</span>
      </div>
    {/if}
  </div>
{/if}
