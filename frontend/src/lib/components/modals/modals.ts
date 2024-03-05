export interface ModalOptions {
	open: boolean;
	onConfirm: () => Promise<void>;
}

export function timeout(ms: number) {
	return new Promise((resolve) => setTimeout(resolve, ms));
}
