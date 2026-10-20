<script lang="ts">
  let { name, size = 20, color = 'currentColor' }: { name: string; size?: number; color?: string } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} stroke-width="1.75"
  stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" style="flex:0 0 auto">
  {#if name === 'about'}
    <circle cx="12" cy="12" r="9" /><path d="M12 11v6M12 7.5v.01" />
  {:else if name === 'settings'}
    <path d="M4 7h10M18 7h2M4 17h4M12 17h8" /><circle cx="16" cy="7" r="2" /><circle cx="10" cy="17" r="2" />
  {:else if name === 'import'}
    <path d="M12 4v11M7 10l5 5 5-5M5 20h14" />
  {:else if name === 'dev'}
    <path d="M8 8l-4 4 4 4M16 8l4 4-4 4M13.5 5l-3 14" />
  {:else if name === 'globe'}
    <circle cx="12" cy="12" r="9" /><path d="M3 12h18M12 3c2.5 2.6 3.8 5.6 3.8 9s-1.3 6.4-3.8 9c-2.5-2.6-3.8-5.6-3.8-9S9.5 5.6 12 3z" />
  {:else if name === 'chev'}
    <path d="M6 9l6 6 6-6" />
  {:else if name === 'check'}
    <path d="M5 12.5l4.5 4.5L19 7.5" />
  {:else if name === 'file'}
    <path d="M6 3h8l4 4v14H6z" /><path d="M14 3v4h4" />
  {:else if name === 'folder'}
    <path d="M3 6h7l2 2h9v11H3z" />
  {:else if name === 'lock'}
    <rect x="5" y="11" width="14" height="9" rx="2" /><path d="M8 11V8a4 4 0 0 1 8 0v3" />
  {:else if name === 'stop'}
    <rect x="6" y="6" width="12" height="12" rx="2" fill={color} stroke="none" />
  {/if}
</svg>
