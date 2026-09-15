import { getContext, setContext } from "svelte";

export interface FieldContext {
	hintId?: string;
	errorId?: string;
	invalid: boolean;
}

const KEY = Symbol("field");

export function setFieldContext(context: FieldContext): void {
	setContext(KEY, context);
}

export function getFieldContext(): FieldContext | undefined {
	return getContext<FieldContext | undefined>(KEY);
}
