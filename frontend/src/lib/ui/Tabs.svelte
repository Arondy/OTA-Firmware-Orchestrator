<script lang="ts">
	import type { Snippet } from "svelte";
	import { Tabs } from "bits-ui";
	import { cn } from "./cn";

	let {
		items,
		value = $bindable(""),
		panel,
		ariaLabel,
		class: className,
	}: {
		items: { value: string; label: string }[];
		value?: string;
		panel: Snippet<[string]>;
		ariaLabel: string;
		class?: string;
	} = $props();
</script>

<Tabs.Root bind:value class={cn("grid gap-3", className)}>
	<Tabs.List aria-label={ariaLabel} class="flex gap-1 border-b border-border-subtle">
		{#each items as item (item.value)}
			<Tabs.Trigger
				value={item.value}
				class={cn(
					"-mb-px border-b-2 px-3 py-1.5 text-table font-medium transition",
					value === item.value
						? "border-accent text-fg-primary"
						: "border-transparent text-fg-secondary hover:text-fg-primary",
				)}
			>
				{item.label}
			</Tabs.Trigger>
		{/each}
	</Tabs.List>

	{#each items as item (item.value)}
		<Tabs.Content value={item.value} class="outline-none">
			{@render panel(item.value)}
		</Tabs.Content>
	{/each}
</Tabs.Root>
