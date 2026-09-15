<script lang="ts">
	import CaretLeftIcon from "phosphor-svelte/lib/CaretLeftIcon";
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon";
	import Button from "./Button.svelte";
	import SegmentedControl from "./SegmentedControl.svelte";
	import { int } from "$lib/format/number";

	/**
	 * Пагинация без общего числа: сервер его не отдаёт, поэтому только
	 * «Назад / Дальше», номер текущей страницы и размер (00-CONTEXT §5.2).
	 */
	let {
		page,
		limit,
		hasMore,
		onPage,
		onLimit,
		class: className,
	}: {
		page: number;
		limit: number;
		hasMore: boolean;
		onPage: (page: number) => void;
		onLimit: (limit: number) => void;
		class?: string;
	} = $props();

	const SIZES = [25, 50, 100].map((size) => ({ value: String(size), label: String(size) }));
</script>

<div class={className}>
	<div class="flex flex-wrap items-center justify-between gap-3">
		<div class="flex items-center gap-2">
			<Button
				variant="secondary"
				size="sm"
				icon={CaretLeftIcon}
				disabled={page <= 1}
				onclick={() => onPage(page - 1)}
			>
				Назад
			</Button>
			<Button
				variant="secondary"
				size="sm"
				icon={CaretRightIcon}
				iconPosition="after"
				disabled={!hasMore}
				onclick={() => onPage(page + 1)}
			>
				Дальше
			</Button>
		</div>

		<div class="flex items-center gap-3 text-dense text-fg-secondary">
			<span>страница <span class="tabular-nums text-fg-primary">{int(page)}</span></span>
			<SegmentedControl
				name="page-size"
				value={String(limit)}
				items={SIZES}
				ariaLabel="Размер страницы"
				onchange={(value) => onLimit(Number(value))}
			/>
		</div>
	</div>
</div>
