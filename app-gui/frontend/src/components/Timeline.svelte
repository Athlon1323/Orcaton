<script lang="ts">
  import Wave from './Wave.svelte';
  import { app } from '../lib/state.svelte';
  import { t } from '../lib/i18n';
  import { tc } from '../lib/format';
  import { layoutAfter, layoutBefore, ticks, THIN_PX } from '../lib/timeline';

  let width = $state(1000);
  const info = $derived(app.info!);
  const seq = $derived(info.sequences[0]);
  const after = $derived(app.ba === 'after' && !!app.result && app.phase !== 'syncing');
  const layout = $derived(
    after ? layoutAfter(info, app.result!.files, app.hidden)
      : layoutBefore(info, app.phase === 'syncing' ? { analyzed: app.analyzed, active: app.active } : undefined),
  );
  const trackW = 190;
  const laneW = $derived(Math.max(860, width * app.zoom) - trackW);
  const marks = $derived(ticks(layout.spanSec, 8));
  const stateText = (s: string) =>
    ({ audio: 'result.audio', clock: 'result.clock', unmatched: 'result.unmatched', offline: 'result.offline', pending: 'legend.waiting', missing: 'legend.missing', analyzed: 'legend.analyzed', active: 'legend.active' })[s] ?? s;
</script>

<div class="scroll" bind:clientWidth={width}>
  <div class="tl" style:width="{app.zoom * 100}%">
    <div class="ruler">
      <div class="pad"></div>
      <div class="ticks mono">{#each marks as m, i (i)}<span>{tc(m, seq, false).slice(0, 8)}</span>{/each}</div>
    </div>
    {#each layout.rows as row (row.camera.id)}
      <div class="track">
        <div class="track-head" title={row.camera.folder}>
          <span class="lane mono">{row.tracks}</span>
          <div><b>{row.name}</b><small>{t('view.files_n', { n: row.count })}</small></div>
        </div>
        <div class="lane-area grid-bg">
          {#each row.groups as g (g.group)}<div class="groups" style:left="{g.left}%" style:width="{g.width}%"></div>{/each}
          {#each row.clips as c (c.key)}
            {@const thin = (c.width / 100) * laneW < THIN_PX}
            <button class="clip s-{c.state}" class:thin class:sel={app.selected === c.key}
              style:left="{c.left}%" style:width="{c.width}%"
              title="{c.name} · {t(stateText(c.state))}" onclick={() => (app.selected = c.key)}>
              <span class="mono">{c.name}{c.state === 'active' ? ` · ${t('legend.active').toLowerCase()}` : ''}</span><Wave />
            </button>
          {/each}
        </div>
      </div>
    {/each}
  </div>
</div>
