<script lang="ts">
  import Brand from './Brand.svelte';
  import { t } from '../lib/i18n';
</script>

<div class="outside">
  <div class="dialog" role="alert">
    <Brand />
    <h3>{t('outside.title')}</h3>
    <p>{t('outside.text')}</p>
    <h2 class="h">{t('outside.howTitle')}</h2>
    <p>{t('outside.build')}</p>
    <pre class="cmd">powershell -ExecutionPolicy Bypass -File app-gui\scripts\build-gui.ps1 -Shortcut -Run</pre>
    <p>{t('outside.dev')}</p>
    <pre class="cmd">powershell -ExecutionPolicy Bypass -File app-gui\scripts\dev-gui.ps1</pre>
    <small class="hint">{t('outside.note')}</small>
  </div>
</div>

<style>
  .outside { height: 100%; display: flex; align-items: center; justify-content: center; padding: 32px; }
  .outside .dialog { width: min(640px, 100%); }
</style>
