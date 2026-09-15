<script lang="ts">
	import ListIcon from "phosphor-svelte/lib/ListIcon";
	import { page } from "$app/state";
	import HealthStrip from "./HealthStrip.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import ThemeToggle from "$lib/ui/ThemeToggle.svelte";
	import { shortId } from "$lib/format/id";

	/** Верхняя полоса: крошки, здоровье стека и переключатель темы. */
	let { onMenu }: { onMenu: () => void } = $props();

	const LABELS: Record<string, string> = {
		campaigns: "Кампании",
		devices: "Устройства",
		firmware: "Прошивки",
		lab: "Песочница",
		new: "новая кампания",
	};

	const crumbs = $derived.by(() => {
		const segments = page.url.pathname.split("/").filter(Boolean);
		if (segments.length === 0) return [{ label: "Обзор", href: "/" }];

		let href = "";
		return segments.map((segment, index) => {
			href += `/${segment}`;
			const known = LABELS[segment];
			return {
				label: known ?? shortId(segment),
				href: index === segments.length - 1 ? undefined : href,
			};
		});
	});
</script>

<header
	class="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-border-subtle bg-bg-surface px-4 md:px-6"
>
	<IconButton glyph={ListIcon} ariaLabel="Открыть меню" class="md:hidden" onclick={onMenu} />

	<nav aria-label="Хлебные крошки" class="min-w-0 flex-1 truncate text-table">
		<ol class="flex items-center gap-1.5">
			{#each crumbs as crumb, index (crumb.label)}
				<li class="flex items-center gap-1.5">
					{#if index > 0}
						<span class="text-fg-muted" aria-hidden="true">/</span>
					{/if}
					{#if crumb.href}
						<a href={crumb.href} class="text-fg-secondary hover:text-fg-primary">{crumb.label}</a>
					{:else}
						<span class="font-medium text-fg-primary" aria-current="page">{crumb.label}</span>
					{/if}
				</li>
			{/each}
		</ol>
	</nav>

	<div class="hidden min-w-64 justify-end lg:flex">
		<HealthStrip />
	</div>

	<ThemeToggle />
</header>
