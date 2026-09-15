/**
 * Переключение темы из любого места: кнопка в сайдбаре и горячая клавиша `t`
 * ведут себя одинаково.
 */
import { mode, resetMode, setMode } from "mode-watcher";

const NEXT = { dark: "light", light: "system", system: "dark" } as const;

export const THEME_LABELS = {
	dark: "тёмная",
	light: "светлая",
	system: "системная",
} as const;

export function cycleTheme(): void {
	const current = mode.current ?? "system";
	const next = NEXT[current];
	if (next === "system") resetMode();
	else setMode(next);
}
