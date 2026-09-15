<script lang="ts">
	import Modal from "./Modal.svelte";

	/**
	 * Обёртка только для юнит-тестов: открытое состояние живёт здесь, тест
	 * дёргает его через экспортированные функции. В прод-код не импортируется.
	 */
	let open = $state(false);

	export function openModal(): void {
		open = true;
	}

	export function closeModal(): void {
		open = false;
	}
</script>

<Modal
	{open}
	onOpenChange={(value) => {
		if (!value) open = false;
	}}
	title="Тестовый диалог"
>
	содержимое диалога
</Modal>
