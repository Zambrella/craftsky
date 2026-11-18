import assert from 'node:assert/strict';
import test from 'node:test';
import { developmentSubscriptions } from './env-files.mjs';

test('only a development server enables billing by default', () => {
	assert.equal(developmentSubscriptions('serve', 'development'), 'true');
	assert.equal(developmentSubscriptions('serve', 'production'), undefined);
	assert.equal(developmentSubscriptions('build', 'production'), undefined);
	assert.equal(developmentSubscriptions('build', 'development'), undefined);
	assert.equal(developmentSubscriptions('serve', 'test'), undefined);
});

test('explicit shell and profile choices override local billing defaults', () => {
	assert.equal(
		developmentSubscriptions('serve', 'development', 'false', 'true'),
		'false'
	);
	assert.equal(
		developmentSubscriptions('serve', 'development', undefined, 'false'),
		'false'
	);
	assert.equal(
		developmentSubscriptions('serve', 'development', '', 'true'),
		''
	);
	assert.equal(
		developmentSubscriptions('build', 'production', 'false', 'true'),
		'false'
	);
});
