<script lang="ts">
	import type { Glyph } from "./glyph";
	import { cn } from "./cn";

	/**
	 * Единственное место, где задаётся вес иконок: `regular` на всё приложение
	 * (00-CONTEXT §2), чтобы на местах вызова не плодить варианты.
	 * Без `label` иконка декоративная и скрыта от скринридеров.
	 */
	let {
		glyph,
		size = 16,
		label,
		class: className,
	}: {
		glyph: Glyph;
		size?: 16 | 18 | 20 | 24;
		label?: string;
		class?: string;
	} = $props();

	const GlyphComponent = $derived(glyph);
</script>

<GlyphComponent
	weight="regular"
	{size}
	aria-hidden={label ? undefined : true}
	role={label ? "img" : undefined}
	aria-label={label}
	class={cn("shrink-0", className)}
/>
