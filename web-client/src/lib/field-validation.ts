import { APIError } from './api/transport';

type Field = HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement;
const errors = new WeakMap<HTMLElement, HTMLElement>();
let serial = 0;
export class FieldValidationError extends Error {
	constructor(
		readonly field: string,
		message: string
	) {
		super(message);
	}
}
const normalise = (value: string) =>
	value.toLowerCase().replace(/[^a-z0-9]/g, '');
function scope(): ParentNode {
	return (
		document.querySelector('dialog[open]') ||
		document.querySelector('.app-main') ||
		document
	);
}
export function findField(
	label: string,
	root: ParentNode = scope()
): HTMLElement | undefined {
	const key = normalise(label);
	if (!key) return undefined;
	return (
		[
			...root.querySelectorAll<Field>(
				'input:not([type=hidden]),textarea,select'
			)
		].find((node) => {
			const names = [
				node.name,
				node.id,
				node.getAttribute('aria-label') || '',
				...[...(node.labels || [])].map(
					(label) => label.firstChild?.textContent || label.textContent || ''
				)
			];
			return names.some(
				(name) => normalise(name) === key || normalise(name).startsWith(key)
			);
		}) ||
		[...root.querySelectorAll<HTMLElement>('[data-field],[aria-label]')].find(
			(node) =>
				normalise(
					node.dataset.field || node.getAttribute('aria-label') || ''
				) === key
		)
	);
}
export function clearFieldError(field: HTMLElement) {
	const error = errors.get(field);
	if (!error) return;
	field.removeAttribute('aria-invalid');
	const described = (field.getAttribute('aria-describedby') || '')
		.split(' ')
		.filter((id) => id && id !== error.id);
	if (described.length)
		field.setAttribute('aria-describedby', described.join(' '));
	else field.removeAttribute('aria-describedby');
	error.remove();
	errors.delete(field);
}
export function showFieldError(
	field: HTMLElement | undefined,
	message: string
): boolean {
	if (!field) return false;
	clearFieldError(field);
	const error = document.createElement('p');
	error.id = `field-error-${++serial}`;
	error.className = 'field-error';
	error.setAttribute('role', 'alert');
	error.textContent = message;
	field.setAttribute('aria-invalid', 'true');
	field.setAttribute(
		'aria-describedby',
		[field.getAttribute('aria-describedby'), error.id].filter(Boolean).join(' ')
	);
	const container =
		field.closest(
			'.native-single-select,.native-multi-select,.facet-text-input,label'
		) || field;
	container.after(error);
	errors.set(field, error);
	return true;
}
export function focusField(field: HTMLElement | undefined) {
	requestAnimationFrame(() => focusNow(field));
}
function focusNow(field: HTMLElement | undefined) {
	if (!field?.isConnected) return;
	const controls = field.matches('input,textarea,select,button')
		? [field]
		: [...field.querySelectorAll<HTMLElement>('input,textarea,select,button')];
	const target = controls.find(
		(node) =>
			!node.matches(':disabled') &&
			node.getClientRects().length &&
			getComputedStyle(node).visibility !== 'hidden'
	);
	if (target) target.focus({ preventScroll: true });
	else {
		field.tabIndex = -1;
		field.focus({ preventScroll: true });
	}
	field.scrollIntoView({ block: 'nearest', behavior: 'instant' });
}
export function fieldError(
	label: string,
	message: string,
	root?: ParentNode
): boolean {
	const field = findField(label, root);
	const shown = showFieldError(field, message);
	if (shown) focusField(field);
	return shown;
}
/** Only classified field errors are consumed; service failures retain their normal feedback. */
export function presentFieldErrors(error: unknown): boolean {
	if (typeof document === 'undefined') return false;
	if (error instanceof FieldValidationError)
		return fieldError(error.field, error.message);
	if (
		!(error instanceof APIError) ||
		!error.fields ||
		error.status < 400 ||
		error.status >= 500
	)
		return false;
	let first: HTMLElement | undefined;
	let allMapped = true;
	for (const [key, message] of Object.entries(error.fields)) {
		const field = findField(key) || findField(key.split('.').at(-1)!);
		if (!showFieldError(field, message)) allMapped = false;
		else first ||= field;
	}
	focusField(first);
	return !!first && allMapped;
}
export function validateFields(root: ParentNode): boolean {
	let first: HTMLElement | undefined;
	for (const field of root.querySelectorAll<Field>('input,textarea,select')) {
		if (field.disabled || field.type === 'hidden') continue;
		let message =
			field.willValidate && !field.validity.valid
				? field.validationMessage
				: '';
		if (
			(field.required || field.getAttribute('aria-required') === 'true') &&
			!(field.dataset.value ?? field.value).trim()
		)
			message = 'This field is required.';
		if (message) {
			showFieldError(field, message);
			first ||= field;
		}
	}
	focusField(first);
	return !first;
}
export function installFieldValidation() {
	const invalid = (event: Event) => {
		const field = event.target;
		if (!(
			field instanceof HTMLInputElement ||
			field instanceof HTMLTextAreaElement ||
			field instanceof HTMLSelectElement
		))
			return;
		event.preventDefault();
		showFieldError(field, field.validationMessage);
		queueMicrotask(() =>
			focusField(
				field.form?.querySelector<HTMLElement>('[aria-invalid=true]') || field
			)
		);
	};
	const edit = (event: Event) => {
		if (event.target instanceof HTMLElement) {
			const target = event.target;
			if (
				(target instanceof HTMLInputElement ||
					target instanceof HTMLTextAreaElement ||
					target instanceof HTMLSelectElement) &&
				errors.has(target) &&
				(((target.required ||
					target.getAttribute('aria-required') === 'true') &&
					!(target.dataset.value ?? target.value).trim()) ||
					(target.willValidate && !target.validity.valid))
			)
				return;
			clearFieldError(event.target);
			const widget = event.target.closest(
				'.native-single-select,.native-multi-select'
			);
			for (const field of widget?.querySelectorAll<HTMLElement>(
				'[aria-invalid=true]'
			) || [])
				clearFieldError(field);
			const group = event.target.closest<HTMLElement>('[aria-invalid=true]');
			if (group) clearFieldError(group);
		}
	};
	const close = (event: Event) => {
		if (event.target instanceof HTMLDialogElement) {
			for (const field of event.target.querySelectorAll<HTMLElement>(
				'[aria-invalid=true]'
			))
				clearFieldError(field);
		}
	};
	const submit = (event: Event) => {
		if (
			event.target instanceof HTMLFormElement &&
			!validateFields(event.target)
		) {
			event.preventDefault();
			event.stopImmediatePropagation();
		}
	};
	document.addEventListener('close', close, true);
	document.addEventListener('invalid', invalid, true);
	document.addEventListener('input', edit);
	document.addEventListener('change', edit);
	document.addEventListener('submit', submit, true);
	return () => {
		document.removeEventListener('close', close, true);
		document.removeEventListener('invalid', invalid, true);
		document.removeEventListener('input', edit);
		document.removeEventListener('change', edit);
		document.removeEventListener('submit', submit, true);
	};
}
