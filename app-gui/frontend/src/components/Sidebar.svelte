<script lang="ts">
  import Icon from './Icon.svelte';
  import Select from './Select.svelte';
  import ImportPanel from './ImportPanel.svelte';
  import SettingsPanel from './SettingsPanel.svelte';
  import AboutPanel from './AboutPanel.svelte';
  import DevPanel from './DevPanel.svelte';
  import { app, type Nav } from '../lib/state.svelte';
  import { LANGUAGES, language, t } from '../lib/i18n';

  const items: { id: Nav; icon: string }[] = [
    { id: 'about', icon: 'about' },
    { id: 'settings', icon: 'settings' },
    { id: 'import', icon: 'import' },
  ];
  const stoppable = $derived(app.busy === 'sync' || (app.busy === 'search' && !!app.info));
</script>

<aside class="side">
  <nav class="nav" aria-label={t('nav.main')}>
    {#each items as it (it.id)}
      <button aria-current={app.nav === it.id ? 'page' : undefined} onclick={() => app.setNav(it.id)}>
        <Icon name={it.icon} />{t(`nav.${it.id}`)}
      </button>
    {/each}
    <hr />
    <button aria-current={app.nav === 'dev' ? 'page' : undefined} onclick={() => app.setNav('dev')}>
      <Icon name="dev" />{t('nav.dev')}<span class="tag">{t('nav.devTag')}</span>
    </button>
  </nav>

  {#if app.nav === 'import'}<ImportPanel />
  {:else if app.nav === 'settings'}<SettingsPanel />
  {:else if app.nav === 'about'}<AboutPanel />
  {:else}<DevPanel />{/if}

  <div class="spacer"></div>
  <div class="lang">
    <Icon name="globe" size={18} />
    <Select variant="bare" label={t('lang.label')} value={language()} onchange={(v) => app.changeLanguage(v)}
      options={LANGUAGES.map((l) => ({ value: l.id, label: l.title, lang: l.id }))} />
  </div>
  <div class="actions">
    <button class="sync" disabled={!app.canSync} onclick={() => app.synchronize()}>
      {app.phase === 'syncing' ? t('action.syncing') : t('action.sync')}
    </button>
    <button class="stop" aria-label={t('action.stop')} title={t('action.stop')} disabled={!stoppable} onclick={() => app.stop()}>
      <Icon name="stop" size={18} />
    </button>
  </div>
</aside>
