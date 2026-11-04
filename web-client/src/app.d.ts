/// <reference types="vite/client" />
declare global {
	const __CRAFTSKY_WEB_RELEASE__: string;
	namespace App {
		interface PageState {
			navigationIndex?: number;
			projectDetail?: string;
			photoDetails?: string;
		}
	}
}
export {};
