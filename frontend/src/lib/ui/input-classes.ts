import { cn } from "./cn";

/** Общая геометрия контролов: 36px высота, радиус из токена --radius-control (00-CONTEXT §8.4). */
export const INPUT_BASE =
	"h-10 w-full rounded-control md:h-9 border bg-bg-raised px-3 text-ui text-fg-primary transition placeholder:text-fg-muted disabled:cursor-not-allowed disabled:opacity-55";

export function inputClasses(invalid: boolean, ...extra: (string | false | undefined)[]): string {
	return cn(
		INPUT_BASE,
		invalid
			? "border-state-danger/60 hover:border-state-danger"
			: "border-border-strong hover:border-border-strong/80",
		...extra,
	);
}
