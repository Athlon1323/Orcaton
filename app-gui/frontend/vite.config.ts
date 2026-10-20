/// <reference types="vitest/config" />
import { writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { defineConfig, type Plugin } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { viteSingleFile } from 'vite-plugin-singlefile';

function keepGitkeep(): Plugin {
  return {
    name: 'orcaton-keep-gitkeep',
    apply: 'build',
    closeBundle() {
      writeFileSync(fileURLToPath(new URL('./dist/.gitkeep', import.meta.url)), '');
    },
  };
}

export default defineConfig(({ mode }) => {
  const mock = mode === 'mock';
  return {
    plugins: [svelte(), ...(mock ? [viteSingleFile()] : [keepGitkeep()])],
    base: './',
    build: {
      outDir: mock ? 'dist-mock' : 'dist',
      emptyOutDir: true,
      target: 'es2022',
      chunkSizeWarningLimit: 1500,
    },
    server: { strictPort: false, watch: { ignored: ['**/wailsjs/**', '**/dist/**', '**/dist-mock/**'] } },
    test: { include: ['src/**/*.test.ts'], environment: 'node' },
  };
});
