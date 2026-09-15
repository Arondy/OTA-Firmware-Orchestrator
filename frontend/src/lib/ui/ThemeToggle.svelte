<script lang="ts">
	import MoonIcon from "phosphor-svelte/lib/MoonIcon";
	import SunIcon from "phosphor-svelte/lib/SunIcon";
	import MonitorIcon from "phosphor-svelte/lib/MonitorIcon";
	import { mode } from "mode-watcher";
	import { cycleTheme, THEME_LABELS } from "$lib/state/theme.svelte";
	import { fade } from "svelte/transition";
	import Icon from "./Icon.svelte";
	import Tooltip from "./Tooltip.svelte";
	import { reducedMotion } from "$lib/state/motion.svelte";

	const ICONS = { dark: MoonIcon, light: SunIcon, system: MonitorIcon } as const;

	const current = $derived(mode.current ?? "system");
</script>

<Tooltip label={`Тема: ${THEME_LABELS[current]}`}>
	<span class="inline-flex">
		<button
			type="button"
			onclick={() => cycleTheme()}
			aria-label={`Переключить тему, текущая ${THEME_LABELS[current]}`}
			class="inline-flex size-10 items-center justify-center rounded-control border border-transparent text-fg-secondary transition hover:bg-bg-raised hover:text-fg-primary md:size-9"
		>
			{#key mode.current}
				<span in:fade={{ duration: reducedMotion.duration(180) }} class="flex">
					<Icon glyph={ICONS[current]} size={18} />
				</span>
			{/key}
		</button>
	</span>
</Tooltip>
