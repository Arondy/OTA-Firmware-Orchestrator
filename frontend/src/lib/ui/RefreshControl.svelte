<script lang="ts">
	import ArrowsClockwiseIcon from "phosphor-svelte/lib/ArrowsClockwiseIcon";
	import { refreshRegistry } from "$lib/state/refresh.svelte";
	import IconButton from "$lib/ui/IconButton.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";

	/**
	 * Экранный контрол обновления: подпись «обновлено ...», кнопка и регистрация
	 * в refreshRegistry (хоткей `r`) одним компонентом - на новой странице
	 * невозможно забыть половину связки.
	 *
	 * Спиннер крутится только при ручном обновлении (клик или `r`): фоновый
	 * поллинг тоже поднимает `refreshing`, но иконка при автоматическом
	 * обновлении обязана оставаться неподвижной.
	 */
	let {
		refreshing,
		refresh,
		updatedAt,
	}: {
		/** Идёт ли сейчас фоновая или ручная перезагрузка ресурсов экрана. */
		refreshing: boolean;
		/** Перезагрузить ресурсы экрана. */
		refresh: () => Promise<void>;
		/** Подпись «обновлено ...»; пустая - до первой загрузки. */
		updatedAt?: string;
	} = $props();

	let manual = $state(false);

	async function refreshManually(): Promise<void> {
		manual = true;
		try {
			await refresh();
		} finally {
			manual = false;
		}
	}

	$effect(() => refreshRegistry.register(refreshManually));
</script>

<div class="flex items-center gap-2">
	{#if updatedAt}
		<span class="text-dense text-fg-muted">обновлено {updatedAt}</span>
	{/if}
	<Tooltip label="Обновить данные экрана">
		<span class="inline-flex">
			<IconButton
				glyph={ArrowsClockwiseIcon}
				ariaLabel="Обновить"
				loading={manual && refreshing}
				disabled={manual && refreshing}
				onclick={() => void refreshManually()}
			/>
		</span>
	</Tooltip>
</div>
