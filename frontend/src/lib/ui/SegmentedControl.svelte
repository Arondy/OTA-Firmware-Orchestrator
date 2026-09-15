<script lang="ts">
	import { cn } from "./cn";

	/**
	 * Сегментированный переключатель на нативных radio: клавиатура и скринридеры
	 * работают из коробки. Выделение выбора «перетекает» между сегментами:
	 * подсветка - отдельный индикатор, который анимированно сдвигается и меняет
	 * ширину (длительность из токена `--dur-2`, при prefers-reduced-motion
	 * токен обнуляется и перетекание становится мгновенным).
	 *
	 * Геометрия повторяет сегментированный блок эталонной панели: трек - это
	 * полупрозрачная подложка `bg-bg-pressed` с внутренним отступом 3px, а не
	 * отдельный тёмный оттенок поверхности; бегунок залит градиентом акцента и
	 * несёт мягкую нейтральную тень; активный сегмент подписан цветом заливки
	 * (`accent-ink`), неактивный - вторичным текстом. Оттенки и тень приходят
	 * из токенов, поэтому тема меняет блок целиком.
	 *
	 * Высоту задаёт трек (`h-10 md:h-9` - та же шкала, что у полей ввода),
	 * а сегменты растягиваются по нему через `h-full`. Раньше высоту задавал
	 * сегмент, и рамка с внутренним отступом добавлялись к ней сверху: блок
	 * выходил на 8px толще соседних полей и кнопки «Сбросить». Внутренняя
	 * высота теперь считается от трека, а не наоборот.
	 *
	 * Обе границы бегунка несёт один `transform`, а не пара `left`/`width`.
	 * `left` и `width` - свойства раскладки: каждый кадр анимации стоит браузеру
	 * пересчёта раскладки и перерисовки. Замер на странице «Устройства» (250
	 * строк в таблице) даёт за 180ms перетекания около 4ms раскладки и 38ms
	 * отрисовки. Оба шага идут на главном потоке, поэтому перетекание дёргается
	 * ровно тогда, когда рядом что-то считается, а смена фильтра как раз
	 * перерисовывает таблицу. `transform` браузер уводит на композитор: на тех же
	 * замерах - 0ms раскладки и 0ms отрисовки, движение не зависит от того, чем
	 * занят главный поток.
	 *
	 * Ширину бегунка тоже приходится нести `transform`: сегменты разной ширины,
	 * а анимация `width` - та же раскладка (замер даёт те же 38ms отрисовки даже
	 * когда позицию двигает `transform`). `scaleX` сжимает заодно градиент и
	 * скругление, поэтому раскладочная ширина бегунка приколота к самому широкому
	 * сегменту, а `scaleX` показывает долю от неё. Сплющенное по горизонтали
	 * скругление делится на тот же коэффициент прямо в `border-radius`, и на
	 * экране угол остаётся круглым. Совпадение с прежним вариантом на
	 * `left`/`width` сверено попиксельно.
	 *
	 * `w-fit` в корне обязателен: в грид-ячейке inline-flex блокфицируется
	 * и без него растянулся бы на всю ширину колонки.
	 */
	let {
		name,
		value,
		items,
		ariaLabel,
		onchange,
		class: className,
	}: {
		name: string;
		value: string;
		items: { value: string; label: string; disabled?: boolean }[];
		ariaLabel: string;
		onchange: (value: string) => void;
		class?: string;
	} = $props();

	let fieldset: HTMLFieldSetElement | undefined = $state(undefined);
	let labels: (HTMLLabelElement | undefined)[] = $state([]);
	let indicator = $state({ x: 0, w: 0, max: 0 });

	// Доля от ширины самого широкого сегмента: её и показывает `scaleX`.
	let scale = $derived(indicator.max > 0 ? indicator.w / indicator.max : 1);

	function measure(): void {
		const index = items.findIndex((item) => item.value === value);
		const label = index >= 0 ? labels[index] : undefined;
		if (!fieldset || !label || label.offsetWidth === 0) {
			indicator = { x: 0, w: 0, max: 0 };
			return;
		}
		const fieldsetRect = fieldset.getBoundingClientRect();
		const labelRect = label.getBoundingClientRect();
		const borderLeft = parseFloat(getComputedStyle(fieldset).borderLeftWidth) || 0;
		// Ширину самого широкого сегмента читаем в том же проходе: она нужна как
		// раскладочная ширина бегунка, а отдельный замер - это лишний пересчёт
		// раскладки на каждое переключение.
		let max = 0;
		for (const item of labels) {
			const rect = item?.getBoundingClientRect();
			if (rect && rect.width > max) max = rect.width;
		}
		indicator = {
			x: labelRect.left - fieldsetRect.left - borderLeft,
			w: labelRect.width,
			max,
		};
	}

	// Замер после каждой смены выбора/набора сегментов и при изменении
	// размеров контейнера (шрифт догрузился, окно повернули).
	$effect(() => {
		void value;
		void items;
		measure();
	});

	$effect(() => {
		if (!fieldset || typeof ResizeObserver === "undefined") return;
		const observer = new ResizeObserver(() => measure());
		observer.observe(fieldset);
		return () => observer.disconnect();
	});
</script>

<fieldset
	bind:this={fieldset}
	role="radiogroup"
	aria-label={ariaLabel}
	class={cn(
		"relative inline-flex h-10 w-fit rounded-panel border border-border-subtle bg-bg-pressed p-[3px] md:h-9",
		className,
	)}
>
	{#if indicator.max > 0}
		<!--
			Ширина в разметке - ширина самого широкого сегмента, а не выбранного:
			анимировать её нельзя, поэтому бегунок всегда занимает её целиком, а
			видимую долю задаёт `scaleX`. Скругление при этом сплющивается по
			горизонтали, поэтому в `border-radius` оно делится на тот же
			коэффициент. `rounded-control` остаётся страховкой на случай нулевой
			доли: тогда `calc` отбросится целиком и радиус возьмётся из класса.
		-->
		<span
			aria-hidden="true"
			class="pointer-events-none absolute top-[3px] bottom-[3px] left-0 origin-left rounded-control bg-accent-gradient shadow-raised transition-transform duration-(--dur-2) ease-(--ease-standard)"
			style="width: {indicator.max}px; transform: translateX({indicator.x}px) scaleX({scale}); border-radius: calc(var(--radius-control) / {scale}) / var(--radius-control)"
		></span>
	{/if}
	{#each items as item, index (item.value)}
		<label class="h-full cursor-pointer" bind:this={labels[index]}>
			<input
				type="radio"
				class="peer sr-only"
				{name}
				value={item.value}
				checked={value === item.value}
				disabled={item.disabled ?? false}
				onchange={() => onchange(item.value)}
			/>
			<span
				class={cn(
					"relative flex h-full items-center whitespace-nowrap rounded-control px-3.5 text-dense font-semibold transition-colors",
					"peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-[var(--ring)]",
					"peer-disabled:cursor-not-allowed peer-disabled:opacity-55",
					value === item.value ? "text-accent-ink" : "text-fg-secondary hover:text-fg-primary",
				)}
			>
				{item.label}
			</span>
		</label>
	{/each}
</fieldset>
