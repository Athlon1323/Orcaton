<script lang="ts">
  import Icon from './Icon.svelte';
  import { moveIndex, placement, typeAhead } from '../lib/listbox';

  type Opt = { value: string; label: string; lang?: string };
  let {
    value, options, onchange, id, label, variant = 'field', bad = false, disabled = false,
  }: {
    value: string; options: Opt[]; onchange: (v: string) => void; id?: string; label?: string;
    variant?: 'field' | 'bare'; bad?: boolean; disabled?: boolean;
  } = $props();

  let open = $state(false);
  let active = $state(-1);
  let btn: HTMLButtonElement | undefined = $state();
  let list: HTMLUListElement | undefined = $state();
  let pos = $state({ left: 0, width: 0, top: 0, bottom: 0, side: 'down' as 'down' | 'up', max: 280 });
  const uid = `sel-${Math.random().toString(36).slice(2, 8)}`;
  const current = $derived(options.find((o) => o.value === value));

  function place(): void {
    if (!btn) return;
    const r = (variant === 'bare' && btn.parentElement ? btn.parentElement : btn).getBoundingClientRect();
    const vh = window.innerHeight;
    const need = Math.min(options.length * 36 + 10, 280);
    const side = placement(r.top, r.bottom, vh, need);
    const room = side === 'down' ? vh - r.bottom - 12 : r.top - 12;
    pos = { left: r.left, width: r.width, top: r.bottom + 6, bottom: vh - r.top + 6, side, max: Math.max(120, Math.min(280, room)) };
  }

  function show(): void {
    if (disabled) return;
    place();
    active = Math.max(0, options.findIndex((o) => o.value === value));
    open = true;
    queueMicrotask(() => list?.querySelector<HTMLElement>(`[data-i="${active}"]`)?.scrollIntoView({ block: 'nearest' }));
  }

  function close(focus = true): void {
    open = false;
    if (focus) btn?.focus();
  }

  function pick(i: number): void {
    const o = options[i];
    close();
    if (o && o.value !== value) onchange(o.value);
  }

  function onKey(e: KeyboardEvent): void {
    if (!open) {
      if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(e.key)) {
        e.preventDefault();
        show();
      }
      return;
    }
    const m = moveIndex(e.key, active, options.length);
    if (m !== null) {
      e.preventDefault();
      active = m;
      list?.querySelector<HTMLElement>(`[data-i="${m}"]`)?.scrollIntoView({ block: 'nearest' });
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      pick(active);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      close();
    } else if (e.key === 'Tab') {
      close(false);
    } else if (e.key.length === 1) {
      const j = typeAhead(e.key, active, options.map((o) => o.label));
      if (j >= 0) active = j;
    }
  }

  $effect(() => {
    if (!open) return;
    const away = (e: PointerEvent) => {
      const t = e.target as Node;
      if (!btn?.contains(t) && !list?.contains(t)) close(false);
    };
    const reflow = (e: Event) => {
      if (!(e.target instanceof Node && list?.contains(e.target))) close(false);
    };
    document.addEventListener('pointerdown', away, true);
    window.addEventListener('resize', reflow);
    window.addEventListener('scroll', reflow, true);
    return () => {
      document.removeEventListener('pointerdown', away, true);
      window.removeEventListener('resize', reflow);
      window.removeEventListener('scroll', reflow, true);
    };
  });
</script>

<button bind:this={btn} {id} type="button" class="sel {variant}" class:bad class:open {disabled}
  role="combobox" aria-haspopup="listbox" aria-expanded={open} aria-controls={uid} aria-label={label}
  aria-activedescendant={open && active >= 0 ? `${uid}-${active}` : undefined}
  onclick={() => (open ? close() : show())} onkeydown={onKey}>
  <span class="cur" lang={current?.lang}>{current?.label ?? value}</span>
  <span class="chev"><Icon name="chev" size={14} /></span>
</button>

{#if open}
  <ul bind:this={list} id={uid} class="sel-pop {pos.side}" role="listbox" aria-label={label} tabindex="-1"
    style:left="{pos.left}px" style:min-width="{pos.width}px" style:max-height="{pos.max}px"
    style:top={pos.side === 'down' ? `${pos.top}px` : null} style:bottom={pos.side === 'up' ? `${pos.bottom}px` : null}>
    {#each options as o, i (o.value)}
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <li id="{uid}-{i}" data-i={i} role="option" aria-selected={o.value === value} lang={o.lang}
        class:active={i === active} class:chosen={o.value === value}
        onpointerenter={() => (active = i)} onpointerdown={(e) => e.preventDefault()} onclick={() => pick(i)}>
        <span class="mark">{#if o.value === value}<Icon name="check" size={14} />{/if}</span>
        <span class="txt">{o.label}</span>
      </li>
    {/each}
  </ul>
{/if}
