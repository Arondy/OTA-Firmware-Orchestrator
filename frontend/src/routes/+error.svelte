<script lang="ts">
	import { page } from "$app/state";
	import Button from "$lib/ui/Button.svelte";
	import Panel from "$lib/ui/Panel.svelte";
</script>

<svelte:head>
	<title>Ошибка - OTA Firmware Orchestrator</title>
</svelte:head>

<Panel title="Страница не открылась" subtitle={`Код ошибки: ${page.status}`}>
	<p class="max-w-[60ch] text-table text-fg-secondary">
		{page.error?.message ??
			"Маршрут не найден или экран упал при отображении. Разделы перечислены в боковой панели."}
	</p>
	<div class="mt-4">
		<Button variant="primary" onclick={() => (window.location.href = "/")}>На главную</Button>
	</div>
</Panel>
