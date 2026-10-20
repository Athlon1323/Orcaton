<script lang="ts">
  import Icon from './Icon.svelte';
  import Select from './Select.svelte';
  import { app } from '../lib/state.svelte';
  import { has, t } from '../lib/i18n';
  import type { Options } from '../lib/types';

  type NumKey = 'minConfidence' | 'minRatio' | 'minOverlapSec' | 'maxOffsetSec' | 'maxGapSec' | 'keepGapSec' | 'jobs' | 'threads' | 'memLimitMB';
  type BoolKey = 'autoThreshold' | 'recordTime' | 'fileOrder' | 'intraLane';

  const o = $derived(app.options);
  const locked = $derived(app.phase === 'syncing');
  const modes = $derived(app.startup?.unsyncedModes ?? ['order_time', 'edges', 'end', 'remove']);
  const orderModes = $derived(app.startup?.orderModes ?? ['name', 'time']);
  const singlesModes = $derived(app.startup?.singlesModes ?? ['sequential', 'stacked']);
  const audioModes = $derived(app.startup?.audioModes ?? ['mix', 'first_stream', 'first_channel']);
  const res = $derived(app.result?.summary.resources);

  const bad = (k: keyof Options) => app.optionError?.field === k;
  const errText = (k: keyof Options) => (has(`range.${k}`) ? t(`range.${k}`) : t('settings.invalid'));
  const setNum = (k: NumKey, v: string) => app.setOption(k, v.trim() === '' ? 0 : Number(v.replace(',', '.')));
  const toggle = (k: BoolKey) => app.setOption(k, !o[k]);
</script>

{#snippet num(k: NumKey, label: string, hint?: string, step = 'any')}
  <div class="fld">
    <label for="opt-{k}">{label}</label>
    <input id="opt-{k}" class="inp" class:bad={bad(k)} type="number" min="0" {step} value={o[k]}
      onchange={(e) => setNum(k, e.currentTarget.value)} aria-invalid={bad(k)} />
    {#if bad(k)}<span class="err" title={app.optionError?.error.message}>{errText(k)}</span>
    {:else if hint}<small>{hint}</small>{/if}
  </div>
{/snippet}

{#snippet tgl(k: BoolKey, label: string)}
  <button class="tgl" role="switch" aria-checked={o[k]} onclick={() => toggle(k)}><i></i>{label}</button>
{/snippet}

<section class="section">
  {#if locked}<div class="lock"><Icon name="lock" size={14} />{t('settings.locked')}</div>{/if}
  <div class="form" class:ro={locked}>
    <h2 class="h">{t('settings.unmatched')}</h2>
    <div class="fld">
      <label for="opt-unsynced">{t('settings.placement')}</label>
      <Select id="opt-unsynced" bad={bad('unsynced')} value={o.unsynced} onchange={(v) => app.setOption('unsynced', v)}
        options={modes.map((m) => ({ value: m, label: t(`place.${m}`) }))} />
      <small>{t(`place.hint.${o.unsynced}`)}</small>
    </div>
    <div class="fld">
      <label for="opt-order">{t('settings.orderBy')}</label>
      <Select id="opt-order" bad={bad('orderBy')} value={o.orderBy} onchange={(v) => app.setOption('orderBy', v)}
        options={orderModes.map((m) => ({ value: m, label: t(`order.${m}`) }))} />
      <small>{t(`order.hint.${o.orderBy}`)}</small>
    </div>
    <div class="fld">
      <label for="opt-singles">{t('settings.singles')}</label>
      <Select id="opt-singles" bad={bad('singles')} value={o.singles} onchange={(v) => app.setOption('singles', v)}
        options={singlesModes.map((m) => ({ value: m, label: t(`singles.${m}`) }))} />
      <small>{t(`singles.hint.${o.singles}`)}</small>
    </div>
    <div class="row2">
      <div class="fld">
        <label for="opt-label">{t('settings.label')}</label>
        <input id="opt-label" class="inp" value={o.unsyncedLabel} onchange={(e) => app.setOption('unsyncedLabel', e.currentTarget.value.trim())} />
      </div>
      <div class="fld">
        <label for="opt-tlabel">{t('settings.timeLabel')}</label>
        <input id="opt-tlabel" class="inp" value={o.timeLabel} onchange={(e) => app.setOption('timeLabel', e.currentTarget.value.trim())} />
      </div>
    </div>
    <small class="hint">{t('settings.labelHint')}</small>

    <h2 class="h">{t('settings.audio')}</h2>
    <div class="fld">
      <label for="opt-audio">{t('settings.audioMode')}</label>
      <Select id="opt-audio" bad={bad('audioMode')} value={o.audioMode} onchange={(v) => app.setOption('audioMode', v)}
        options={audioModes.map((m) => ({ value: m, label: t(`audio.${m}`) }))} />
    </div>
    <div class="row2">
      {@render num('minConfidence', t('settings.minConfidence'))}
      {@render num('maxOffsetSec', t('settings.maxOffset'), t('settings.maxOffsetHint'))}
    </div>
    {@render tgl('autoThreshold', t('settings.autoThreshold'))}
    <div class="row2">
      {@render num('minRatio', t('settings.minRatio'))}
      {@render num('minOverlapSec', t('settings.minOverlap'))}
    </div>

    <h2 class="h">{t('settings.clock')}</h2>
    {@render tgl('recordTime', t('settings.recordTime'))}
    {@render tgl('fileOrder', t('settings.fileOrder'))}
    {@render tgl('intraLane', t('settings.intraLane'))}

    <h2 class="h">{t('settings.gaps')}</h2>
    <div class="row2">
      {@render num('maxGapSec', t('settings.maxGap'), t('settings.maxGapHint'))}
      {@render num('keepGapSec', t('settings.keepGap'))}
    </div>

    <h2 class="h">{t('settings.perf')}</h2>
    {@render num('jobs', t('settings.jobs'), t('settings.jobsHint'), '1')}
    <div class="row2">
      {@render num('threads', t('settings.threads'), t('settings.threadsHint'), '1')}
      {@render num('memLimitMB', t('settings.mem'), t('settings.memHint'), '256')}
    </div>
    <div class="fld">
      <label for="opt-cache">{t('settings.cache')}</label>
      <div class="pick">
        <input id="opt-cache" class="inp" value={o.cacheDir} placeholder={t('settings.cacheHint')}
          onchange={(e) => app.setOption('cacheDir', e.currentTarget.value.trim())} />
        <button class="btn small" onclick={() => app.chooseCacheFolder()}>{t('settings.choose')}</button>
      </div>
      {#if o.cacheDir}<button class="link" style="height:auto;padding:0" onclick={() => app.setOption('cacheDir', '')}>{t('settings.cacheDefault')}</button>{/if}
    </div>
    {#if res && res.peakMB > 0}<small class="hint">{t('settings.peak', { peak: res.peakMB, limit: res.memLimitMB })}</small>{/if}

    <h2 class="h">{t('settings.save')}</h2>
    <button class="tgl" role="switch" aria-checked={app.keepPaths} onclick={() => app.setKeepPaths(!app.keepPaths)}><i></i>{t('settings.keepPaths')}</button>
    <small class="hint">{t('settings.keepPathsHint')}</small>

    <button class="link" onclick={() => app.resetOptions()}>{t('settings.reset')}</button>
    <small class="hint" title={app.startup?.settingsPath}>
      {app.startup?.settingsPath ? t('settings.file', { path: app.startup.settingsPath }) : t('settings.memoryOnly')}
      {#if app.saveState === 'saving'} · {t('settings.savingNow')}{:else if app.saveState === 'saved'} · {t('settings.saved')}{/if}
    </small>
  </div>
</section>
