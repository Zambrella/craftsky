import { describe, it, expect } from 'vitest';
import { reactionBounds } from './reaction-bounds';
const viewport = { left: 0, top: 0, right: 1000, bottom: 800 };
describe('visible post reaction bounds', () => {
	it('centres the native 160px reaction on a visible card', () => {
		expect(
			reactionBounds({ left: 200, top: 100, right: 800, bottom: 700 }, [
				viewport
			])
		).toEqual({ left: 420, top: 320, size: 160 });
	});
	it('centres within the visible portion of a tall, partially scrolled post', () => {
		expect(
			reactionBounds({ left: 200, top: -1500, right: 800, bottom: 400 }, [
				viewport
			])
		).toEqual({ left: 420, top: 120, size: 160 });
	});
	it('honours nested clips and shrinks to a narrow visible strip', () => {
		expect(
			reactionBounds({ left: 200, top: 100, right: 800, bottom: 700 }, [
				viewport,
				{ left: 400, top: 250, right: 480, bottom: 600 }
			])
		).toEqual({ left: 400, top: 385, size: 80 });
	});
	it('does not paint a fully clipped post', () => {
		expect(
			reactionBounds({ left: 200, top: 900, right: 800, bottom: 1400 }, [
				viewport
			])
		).toBeUndefined();
	});
});
