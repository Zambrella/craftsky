import { publicReadPath } from './public-path';

/** Reporting is the sole anonymous write; no other member mutation is mapped. */
export function publicReportPath(
	path: string,
	query = new URLSearchParams()
): string {
	if (
		query.size ||
		!/^(posts\/[^/]+\/[^/]+|profiles\/[^/]+)\/reports$/.test(path)
	)
		throw new TypeError('Invalid public report request');
	return publicReadPath(path.slice(0, -'/reports'.length)) + '/reports';
}
