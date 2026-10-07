import assert from 'node:assert/strict';
import test from 'node:test';

import { goBack } from '../src/lib/navigation.ts';

test('returns to the previous page when browser history exists', () => {
	let prevented = false;
	let backed = false;
	globalThis.history = { length: 2, back: () => (backed = true) };

	goBack({ preventDefault: () => (prevented = true) });

	assert.equal(prevented, true);
	assert.equal(backed, true);
});

test('allows the dashboard link when there is no previous page', () => {
	let prevented = false;
	let backed = false;
	globalThis.history = { length: 1, back: () => (backed = true) };

	goBack({ preventDefault: () => (prevented = true) });

	assert.equal(prevented, false);
	assert.equal(backed, false);
});
