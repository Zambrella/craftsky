import { readdirSync, readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import type { Page } from '@playwright/test';
export async function mockRevenueCat(page: Page) {
	// Replace the dynamically loaded provider SDK, not the CraftSky UI/coordinator.
	// Live sandbox renders are checked separately; no network purchase is made here.
	const chunks = fileURLToPath(
		new URL(
			'../.svelte-kit/output/client/_app/immutable/chunks/',
			import.meta.url
		)
	);
	const sdk = readdirSync(chunks).find(
		(name) =>
			name.endsWith('.js') &&
			readFileSync(chunks + name, 'utf8').includes(' as Purchases,')
	);
	if (!sdk) throw new Error('RevenueCat SDK bundle missing');
	await page.route(`**/_app/immutable/chunks/${sdk}`, async (route) => {
		return route.fulfill({
			contentType: 'text/javascript',
			body: `
   export const LogLevel={Silent:'silent'};
   export class Purchases {
    static setLogLevel(){} static configure(options){window.fixtureIdentity=options;return new Purchases();}
    async getOfferings(){return {all:{plus:{identifier:'plus',availablePackages:[{identifier:'$rc_monthly'}]},business:{identifier:'business',availablePackages:[{identifier:'$rc_monthly'}]}}};}
    close(){this.target?.replaceChildren();}
    async purchase(params){return new Promise((resolve,reject)=>{
     const title=document.createElement('h3');title.textContent='Stripe checkout';
     const complete=document.createElement('button');complete.textContent='Complete fixture payment';complete.onclick=()=>{params.htmlTarget.replaceChildren();resolve({customerInfo:{}});};
     const cancel=document.createElement('button');cancel.textContent='Back to plans';cancel.onclick=()=>{params.htmlTarget.replaceChildren();reject({errorCode:1});};
     params.htmlTarget.replaceChildren(title,complete,cancel);
    });}
    presentPaywall(options){this.target=options.htmlTarget;return new Promise((resolve,reject)=>{
     const h=document.createElement('h3');h.textContent='RevenueCat '+options.offering.identifier+' paywall';
     const button=document.createElement('button');button.textContent='Continue to checkout';
     button.onclick=async()=>{try{const r=await this.purchase({htmlTarget:options.purchaseHtmlTarget});resolve(r);}catch(error){options.listener?.onPurchaseError?.(error);}};
     this.target.replaceChildren(h,button);
    });}
   }`
		});
	});
}
