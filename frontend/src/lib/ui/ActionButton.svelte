<script lang="ts">
	import type { Snippet } from "svelte";
	import Button from "./Button.svelte";

	/**
	 * Кнопка основного действия экрана: «Новая кампания», «Зарегистрировать
	 * прошивку», «Зарегистрировать устройство». Один компонент на все экраны,
	 * потому что кнопки одного смысла не должны выглядеть по-разному: раньше
	 * «Прошивки» заливались акцентом, а «Устройства» оставались нейтральными, и
	 * разница читалась как случайность, а не как иерархия.
	 *
	 * Тон и размер зафиксированы здесь. `Button` остаётся конструктором для
	 * локальных случаев - диалогов, баннеров, фильтров; вход в основной сценарий
	 * собирается только отсюда, поэтому вид кнопки меняется в одном месте.
	 */
	let {
		href,
		onclick,
		loading = false,
		disabled = false,
		children,
	}: {
		/** Навигация вместо действия: тот же вид, но ссылкой. */
		href?: string;
		onclick?: (event: MouseEvent) => void;
		loading?: boolean;
		disabled?: boolean;
		children: Snippet;
	} = $props();
</script>

<Button variant="secondary" {href} {onclick} {loading} {disabled}>
	{@render children()}
</Button>
