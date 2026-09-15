import { describe, expect, it } from "vitest";
import { flushSync, mount, unmount } from "svelte";
import ModalHarness from "./ModalHarness.svelte";

/**
 * У управляемого модального окна нет Dialog.Trigger, которому bits-ui вернул бы
 * фокус сам, поэтому закрытие обязано вернуть фокус на элемент, открывший диалог.
 */

function mountHarness(): { component: ReturnType<typeof mount>; dispose: () => void } {
	const component = mount(ModalHarness, { target: document.body });
	return { component, dispose: () => unmount(component) };
}

describe("Modal: возврат фокуса", () => {
	it("после закрытия фокус возвращается на элемент, бывший до открытия", () => {
		const { component, dispose } = mountHarness();
		const trigger = document.createElement("button");
		trigger.type = "button";
		trigger.textContent = "Открыть";
		document.body.append(trigger);
		trigger.focus();
		expect(document.activeElement).toBe(trigger);

		component.openModal();
		flushSync();

		component.closeModal();
		flushSync();

		expect(document.activeElement).toBe(trigger);
		dispose();
	});

	it("не бросает исключение, если элемент за жизнь диалога исчез из DOM", () => {
		const { component, dispose } = mountHarness();
		const trigger = document.createElement("button");
		trigger.type = "button";
		document.body.append(trigger);
		trigger.focus();

		component.openModal();
		flushSync();
		trigger.remove();
		expect(document.activeElement).toBe(document.body);

		expect(() => {
			component.closeModal();
			flushSync();
		}).not.toThrow();

		expect(document.activeElement).toBe(document.body);
		dispose();
	});
});
