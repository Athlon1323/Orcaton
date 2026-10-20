<script lang="ts">
  import TopBar from './components/TopBar.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import MainView from './components/MainView.svelte';
  import MissingDialog from './components/MissingDialog.svelte';
  import Toasts from './components/Toasts.svelte';
  import OutsideNotice from './components/OutsideNotice.svelte';
  import { app } from './lib/state.svelte';
  import { t } from './lib/i18n';

  function ondragover(e: DragEvent) {
    e.preventDefault();
  }
  function ondrop(e: DragEvent) {
    e.preventDefault();
    if (app.backend?.kind === 'mock') app.toast('info', t('toast.mockDrop'));
  }
</script>

<svelte:window {ondragover} {ondrop} />
{#if app.outside}
  <OutsideNotice />
{:else}
  <div class="app">
    <TopBar />
    <div class="body">
      <Sidebar />
      <MainView />
    </div>
  </div>
  <MissingDialog />
  <Toasts />
{/if}
