import type {
	Reporter,
	TestCase,
	TestResult,
	TestStep,
	FullResult,
	FullConfig,
	Suite
} from '@playwright/test/reporter';
/** Never render Playwright error attachments, OAuth URLs, request bodies or credential call logs. */
export default class LiveReporter implements Reporter {
	onBegin(_config: FullConfig, suite: Suite) {
		console.log(`Selected ${suite.allTests().length} live smoke test.`);
	}
	onStdOut(chunk: string | Buffer) {
		process.stdout.write(chunk);
	}
	onTestBegin() {
		console.log('Live smoke started.');
	}
	onStepBegin(_test: TestCase, _result: TestResult, step: TestStep) {
		if (step.category === 'test.step') console.log(step.title);
	}
	onTestEnd(_test: TestCase, result: TestResult) {
		console.log(`Live smoke: ${result.status}.`);
		for (const error of result.errors) {
			// Single-line cause only; no call logs, snapshots, URLs or attachments.
			const summary = error.message
				?.split('\n')[0]
				.replace(/https?:\/\/[^\s)]+/g, '[URL]');
			if (summary) console.log(summary.slice(0, 300));
			const location = error.stack?.match(
				/(?:smoke\.spec|api)\.ts:\d+:\d+/
			)?.[0];
			if (location) console.log(`Failed check at ${location}.`);
		}
		if (result.status !== 'passed')
			console.log(
				'Check the last named stage. Private browser artifacts are disabled.'
			);
	}
	onError() {
		console.log(
			'Live runner failed before completion; check local server availability and browser installation.'
		);
	}
	onEnd(result: FullResult) {
		console.log(`Live suite: ${result.status}.`);
	}
}
