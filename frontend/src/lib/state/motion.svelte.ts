/**
 * Реактивный флаг `prefers-reduced-motion`. Svelte-переходы (`transition:`)
 * не подчиняются CSS-медиазапросу сами по себе, поэтому читают этот флаг
 * (00-CONTEXT §8.6). CSS-часть гасится отдельно в app.css.
 */
class ReducedMotion {
	current = $state(false);

	#query: MediaQueryList | undefined;
	#onChange = (event: MediaQueryListEvent): void => {
		this.current = event.matches;
	};

	/** Возвращает функцию отписки; вызывается из `$effect` корневого макета. */
	track(): () => void {
		if (typeof window === "undefined") return () => {};

		this.#query = window.matchMedia("(prefers-reduced-motion: reduce)");
		this.current = this.#query.matches;
		this.#query.addEventListener("change", this.#onChange);
		return () => {
			this.#query?.removeEventListener("change", this.#onChange);
			this.#query = undefined;
		};
	}

	/** Длительность перехода с учётом предпочтения пользователя. */
	duration(ms: number): number {
		return this.current ? 0 : ms;
	}
}

export const reducedMotion = new ReducedMotion();
