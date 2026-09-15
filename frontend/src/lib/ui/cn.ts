type ClassValue = string | false | null | undefined;

/** Склейка классов без зависимости: ложные значения просто отбрасываются. */
export function cn(...values: ClassValue[]): string {
	return values.filter(Boolean).join(" ");
}
