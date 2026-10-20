import '@fontsource-variable/manrope';
import '@fontsource-variable/noto-sans-georgian';
import '@fontsource/jetbrains-mono/400.css';
import '@fontsource/jetbrains-mono/500.css';
import './styles/tokens.css';
import './styles/app.css';
import { mount } from 'svelte';
import App from './App.svelte';
import { app } from './lib/state.svelte';

mount(App, { target: document.getElementById('app')! });

async function start(): Promise<void> {
  await app.init();
  if (import.meta.env.MODE === 'mock' && location.hash) {
    const load = import.meta.glob<{ applyScenario: (a: typeof app, s: string) => Promise<void> }>('./lib/scenarios.ts')['./lib/scenarios.ts'];
    if (load) await (await load()).applyScenario(app, location.hash.slice(1));
    window.addEventListener('hashchange', () => location.reload());
  }
}

void start();
