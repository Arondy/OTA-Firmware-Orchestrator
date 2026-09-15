<script lang="ts">
	import CaretLeftIcon from "phosphor-svelte/lib/CaretLeftIcon";
	import CaretRightIcon from "phosphor-svelte/lib/CaretRightIcon";
	import GaugeIcon from "phosphor-svelte/lib/GaugeIcon";
	import RocketIcon from "phosphor-svelte/lib/RocketIcon";
	import CpuIcon from "phosphor-svelte/lib/CpuIcon";
	import BinaryIcon from "phosphor-svelte/lib/BinaryIcon";
	import FlaskIcon from "phosphor-svelte/lib/FlaskIcon";
	import { page } from "$app/state";
	import { absoluteDateTime } from "$lib/format/datetime";
	import Icon from "$lib/ui/Icon.svelte";
	import Kbd from "$lib/ui/Kbd.svelte";
	import Tooltip from "$lib/ui/Tooltip.svelte";
	import { cn } from "$lib/ui/cn";

	/**
	 * Боковая панель разделов.
	 *
	 * Сворачивание анимируется шириной: тексты пунктов не удаляются
	 * из DOM, а схлопываются по max-width с прозрачностью. Геометрия
	 * строки постоянна (gap/padding не меняются), иконка прижата влево
	 * и стоит на месте всю анимацию. Заливка активного пункта живёт на
	 * отдельном абсолютном слое и морфится той же длительностью
	 * и кривой (240ms), что и ширина панели: в свёрнутом виде это
	 * квадрат 36px со сдвигом 1px от левого края, в развёрнутом -
	 * вся строка. При сдвиге 1px иконка (отступ 10px) сидит в центре
	 * квадрата с полями 9/9 без единого движения: на иконке больше
	 * нет условных классов. Полоса лежит внутри подложки у её левого
	 * края - левые границы совпадают, на размер она не влияет.
	 * Геометрия строки при этом не меняется. Навигация режет
	 * переполнение по горизонтали, чтобы во время анимации
	 * не возникал скроллбар.
	 * Футер в свёрнутом виде не резервирует место под скрытые подписи:
	 * разделитель стоит прямо над кнопкой разворачивания.
	 *
	 * Кнопка сворачивания живёт внизу панели - единственная стабильная позиция
	 * в обоих состояниях. Футер: сборка слева и версия справа на одной строке.
	 */
	let {
		collapsed = false,
		onToggle,
	}: {
		collapsed?: boolean;
		onToggle?: () => void;
	} = $props();

	const ITEMS = [
		{ href: "/", label: "Обзор", icon: GaugeIcon },
		{ href: "/campaigns", label: "Кампании", icon: RocketIcon },
		{ href: "/devices", label: "Устройства", icon: CpuIcon },
		{ href: "/firmware", label: "Прошивки", icon: BinaryIcon },
		{ href: "/lab", label: "Песочница", icon: FlaskIcon },
	] as const;

	/** Общий класс схлопывания текста при сворачивании панели. */
	const FADE = $derived(
		cn(
			"min-w-0 overflow-hidden whitespace-nowrap transition-[max-width,opacity]",
			"duration-(--dur-3) ease-(--ease-standard)",
			collapsed ? "max-w-0 opacity-0" : "max-w-52 opacity-100",
		),
	);

	function isActive(href: string): boolean {
		const path = page.url.pathname;
		return href === "/" ? path === "/" : path === href || path.startsWith(`${href}/`);
	}
</script>

<aside
	class={cn(
		"row-span-2 flex flex-col border-r border-border-subtle bg-bg-surface transition-[width] duration-240",
		"sticky top-0 h-[100dvh]",
		collapsed ? "w-14" : "w-58",
	)}
>
	<!-- overflow-hidden: в свёрнутом виде схлопнутый заголовок не должен
	     выталкивать горизонтальную прокрутку; логотип держит явный размер
	     (атрибуты + shrink-0) и в обоих состояниях рисуется 1:1. -->
	<div
		class="flex h-14 shrink-0 items-center gap-2 overflow-hidden border-b border-border-subtle px-4"
	>
		<img src="/favicon.svg" alt="" width="24" height="24" class="size-6 min-w-6 shrink-0" />
		<span class={cn("min-w-0 flex-1 text-ui leading-tight font-semibold text-fg-primary", FADE)}>
			OTA Orchestrator
		</span>
	</div>

	<nav aria-label="Основная" class="flex-1 overflow-x-hidden overflow-y-auto p-2">
		<ul class="grid gap-1">
			{#each ITEMS as item (item.href)}
				<li>
					<a
						href={item.href}
						aria-current={isActive(item.href) ? "page" : undefined}
						title={collapsed ? item.label : undefined}
						class={cn(
							"group relative flex h-9 items-center gap-2.5 overflow-hidden rounded-control px-2.5 text-table transition-colors",
							isActive(item.href)
								? "font-medium text-fg-primary"
								: "text-fg-secondary hover:text-fg-primary",
						)}
					>
						<!-- Подложка заливки: абсолютный слой, геометрию строки
						     не трогает, поэтому иконка не двигается. Позиция
						     и ширина морфятся синхронно с панелью (240ms,
						     та же кривая): квадрат 36px со сдвигом 1px
						     в свёрнутом виде, вся строка в развёрнутом.
						     Полоса внутри подложки у левого края: левые
						     границы совпадают. Ховер красится
						     через group-hover. -->
						<span
							aria-hidden="true"
							class={cn(
								"absolute inset-y-0 rounded-control transition-[left,width,background-color] duration-240 ease-(--ease-standard)",
								collapsed ? "left-px w-9" : "left-0 w-full",
								isActive(item.href) ? "bg-accent-tint" : "bg-transparent group-hover:bg-bg-raised",
							)}
						>
							{#if isActive(item.href)}
								<span class="absolute inset-y-1 left-0 w-0.5 rounded-full bg-accent"></span>
							{/if}
						</span>
						<Icon glyph={item.icon} size={18} class="relative shrink-0" />
						<span class={cn("relative", FADE)}>{item.label}</span>
					</a>
				</li>
			{/each}
		</ul>
	</nav>

	<div class="shrink-0 border-t border-border-subtle p-2">
		{#if !collapsed}
			<div class="flex items-baseline justify-between gap-2 leading-tight">
				<span class="truncate text-micro text-fg-muted">
					сборка {absoluteDateTime(__BUILD_TIME__)}
				</span>
				<span class="shrink-0 font-mono text-micro text-fg-muted">v{__APP_VERSION__}</span>
			</div>
		{/if}
		<div
			class={cn("flex items-center gap-2", collapsed ? "justify-center" : "mt-1 justify-between")}
		>
			{#if !collapsed}
				<span class="flex items-center gap-1 text-micro text-fg-muted">
					<Kbd>?</Kbd> клавиши
				</span>
			{/if}
			{#if onToggle}
				<Tooltip label={collapsed ? "Развернуть панель" : "Свернуть панель"}>
					<button
						type="button"
						onclick={onToggle}
						aria-label={collapsed ? "Развернуть панель" : "Свернуть панель"}
						class="hidden size-6 shrink-0 items-center justify-center rounded-chip text-fg-muted transition hover:bg-bg-raised hover:text-fg-primary md:flex"
					>
						<Icon glyph={collapsed ? CaretRightIcon : CaretLeftIcon} size={16} />
					</button>
				</Tooltip>
			{/if}
		</div>
	</div>
</aside>
