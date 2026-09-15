<script lang="ts">
	import type { Snippet } from "svelte";
	import { ModeWatcher, mode } from "mode-watcher";
	import { Toaster } from "svelte-sonner";
	import { fade } from "svelte/transition";
	import { page } from "$app/state";
	import AppShell from "$lib/layout/AppShell.svelte";
	import { clock } from "$lib/state/clock.svelte";
	import { reducedMotion } from "$lib/state/motion.svelte";

	import "../app.css";
	// Шрифты только self-hosted, никакого `<link>` на Google Fonts (§8.3).

	let { children }: { children: Snippet } = $props();

	$effect(() => {
		clock.start();
		return () => clock.stop();
	});

	$effect(() => reducedMotion.track());
</script>

<!--
  `disableHeadScriptInjection`: в SPA-сборке компонент вставляет скрипт после
  бандла и вспышку темы предотвратить не может; класс `.dark` до первого кадра
  ставит внешний static/theme-init.js (см. app.html и svelte.config.js kit.csp).
-->
<ModeWatcher disableHeadScriptInjection />

<Toaster
	theme={mode.current}
	position="bottom-right"
	visibleToasts={3}
	duration={4000}
	closeButton={false}
/>

<AppShell>
	{#key page.url.pathname}
		<div in:fade={{ duration: reducedMotion.duration(120) }}>
			{@render children()}
		</div>
	{/key}
</AppShell>
