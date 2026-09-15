<script lang="ts">
	/** Хлебные крошки: последние в списке - текущая страница, без ссылки. */
	let { items }: { items: { href?: string; label: string }[] } = $props();
</script>

<nav aria-label="Хлебные крошки" class="flex flex-wrap items-center gap-1.5 text-dense">
	{#each items as item, index (item.label + index)}
		{#if index > 0}
			<span aria-hidden="true" class="text-fg-muted">/</span>
		{/if}
		{#if item.href && index < items.length - 1}
			<a
				href={item.href}
				class="text-fg-secondary underline-offset-4 hover:text-accent-text hover:underline"
			>
				{item.label}
			</a>
		{:else}
			<span aria-current="page" class="text-fg-secondary">{item.label}</span>
		{/if}
	{/each}
</nav>
