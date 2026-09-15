<script lang="ts">
	import ArrowDownIcon from "phosphor-svelte/lib/ArrowDownIcon";
	import type { LogLine } from "$lib/state/batch-runner.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import Panel from "$lib/ui/Panel.svelte";
	import { cn } from "$lib/ui/cn";

	/**
	 * Журнал прогона: моноширинный, с метками времени, не больше 500 строк
	 * (старые выпадают). Автопрокрутка держится за низом, только пока оператор
	 * сам не ушёл вверх: тогда появляется кнопка «вниз». Журнал сессионный.
	 */
	let { lines }: { lines: (LogLine & { time: string })[] } = $props();

	let region: HTMLDivElement | undefined = $state(undefined);
	let pinned = $state(true);
	let showDown = $state(false);

	function syncPin(): void {
		if (!region) return;
		const distance = region.scrollHeight - region.scrollTop - region.clientHeight;
		pinned = distance < 24;
		showDown = !pinned;
	}

	$effect(() => {
		const count = lines.length;
		if (count === 0 || !pinned || !region) return;
		region.scrollTop = region.scrollHeight;
	});

	const TONE = {
		success: "text-state-success",
		danger: "text-state-danger",
		neutral: "text-fg-secondary",
	} as const;
</script>

<Panel title="Журнал">
	<div class="relative grid gap-2">
		<div
			bind:this={region}
			onscroll={syncPin}
			aria-live="polite"
			class="h-80 overflow-y-auto rounded-chip bg-bg-inset p-2 font-mono text-dense leading-relaxed"
		>
			{#if lines.length === 0}
				<p class="text-fg-muted">Строк пока нет</p>
			{:else}
				{#each lines as line (line.at + line.device + line.kind)}
					<p class={cn("whitespace-pre-wrap", TONE[line.tone])}>
						<span class="text-fg-muted">{line.time}</span>
						<span class="text-fg-muted">&nbsp;&nbsp;</span>
						{line.kind}
						<span class="text-fg-muted">&nbsp;&nbsp;</span>
						{line.device}
						{#if line.status !== undefined}
							<span class="text-fg-muted">&nbsp;&nbsp;</span>
							{line.status}
						{/if}
						<span class="text-fg-muted">&nbsp;&nbsp;</span>
						{line.note}
					</p>
				{/each}
			{/if}
		</div>

		{#if showDown}
			<IconButton
				glyph={ArrowDownIcon}
				ariaLabel="Прокрутить журнал вниз"
				size="sm"
				class="absolute right-2 bottom-2"
				onclick={() => {
					if (region) region.scrollTop = region.scrollHeight;
					pinned = true;
					showDown = false;
				}}
			/>
		{/if}
	</div>
</Panel>
