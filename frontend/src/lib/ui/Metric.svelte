<script lang="ts">
	import { cn } from "./cn";

	/**
	 * Ключевой показатель: микро-подпись и моноширинное значение с табличными
	 * цифрами. При смене значения - короткая подсветка фоном (§8.6): обратная
	 * связь «число обновилось», а не украшение.
	 */
	let {
		label,
		value,
		hint,
		class: className,
	}: {
		label: string;
		value: string;
		hint?: string;
		class?: string;
	} = $props();

	let flashing = $state(false);
	let previous: string | undefined = $state(undefined);

	$effect(() => {
		if (previous === undefined) {
			previous = value;
			return;
		}
		if (previous === value) return;

		previous = value;
		flashing = true;
		const timer = setTimeout(() => {
			flashing = false;
		}, 600);
		return () => clearTimeout(timer);
	});
</script>

<div
	class={cn(
		"grid gap-1 rounded-chip px-1 transition-colors",
		flashing && "metric-flash",
		className,
	)}
>
	<span class="text-micro font-medium text-fg-muted uppercase">{label}</span>
	<span class="font-mono text-metric font-semibold tabular-nums text-fg-primary">{value}</span>
	{#if hint}
		<span class="text-dense text-fg-secondary">{hint}</span>
	{/if}
</div>
