<script lang="ts">
	import Kbd from "$lib/ui/Kbd.svelte";
	import Modal from "$lib/ui/Modal.svelte";

	/**
	 * Справка по горячим клавишам: список фиксированный и короткий, это
	 * удобство оператора, а не продукт.
	 */
	let { open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void } = $props();

	const ROWS: { keys: string[]; note: string }[] = [
		{ keys: ["/"], note: "фокус на поле поиска (если оно есть на экране)" },
		{ keys: ["g", "o"], note: "перейти: Обзор" },
		{ keys: ["g", "c"], note: "перейти: Кампании" },
		{ keys: ["g", "d"], note: "перейти: Устройства" },
		{ keys: ["g", "f"], note: "перейти: Прошивки" },
		{ keys: ["g", "l"], note: "перейти: Песочница" },
		{ keys: ["r"], note: "обновить данные экрана" },
		{ keys: ["t"], note: "сменить тему" },
		{ keys: ["?"], note: "эта справка" },
		{ keys: ["Esc"], note: "закрыть оверлей" },
	];
</script>

<Modal {open} {onOpenChange} title="Горячие клавиши">
	<ul class="grid gap-2">
		{#each ROWS as row (row.note)}
			<li class="flex items-center justify-between gap-4">
				<span class="flex items-center gap-1">
					{#each row.keys as key (key)}
						<Kbd>{key}</Kbd>
					{/each}
				</span>
				<span class="text-table text-fg-secondary">{row.note}</span>
			</li>
		{/each}
	</ul>
</Modal>
