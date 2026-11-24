import { craftChoice, label, projectMethodValues } from './project-options';
import { str } from './model';

export function extraDetails(
	craft: string,
	detail: Record<string, unknown>
): [string, string][] {
	const methods = () =>
		projectMethodValues(craft, detail).map(label).join(', ');
	switch (craftChoice(craft)?.profileId) {
		case 'knitting':
			return [['Knitting methods', methods()]];
		case 'crochet':
			return [['Crochet methods', methods()]];
		case 'embroidery':
			return [
				['Method', label(str(detail.method).trim())],
				['Style', label(str(detail.style).trim())]
			];
		case 'weaving':
			return [['Loom type', label(str(detail.loomType).trim())]];
		case 'felting':
			return [['Felting methods', methods()]];
		default:
			return [];
	}
}
