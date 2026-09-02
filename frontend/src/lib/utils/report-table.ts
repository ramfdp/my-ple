export interface GroupableReport {
	primaryDivision: string;
	workDivision: string;
	role: string;
	work: string;
	number: number;
}

export type TableReport<T extends GroupableReport> =
	| { type: 'work'; work: string; category?: string; role?: string; categoryRowSpan?: number; roleRowSpan?: number }
	| { type: 'report'; report: T };

function compareReports(a: GroupableReport, b: GroupableReport) {
	for (const field of ['primaryDivision', 'workDivision'] as const) {
		const comparison = a[field].localeCompare(b[field], 'id');
		if (comparison !== 0) return comparison;
	}
	return a.number - b.number;
}

function renderedSpan<T extends GroupableReport>(reports: T[], start: number, sameGroup: (report: T) => boolean) {
	let span = 0;
	for (let index = start; index < reports.length && sameGroup(reports[index]); index += 1) {
		span += 1;
		if (index === start || reports[index].role !== reports[index - 1].role || reports[index].work !== reports[index - 1].work) span += 1;
	}
	return span;
}

export function groupReports<T extends GroupableReport>(reports: T[]): TableReport<T>[] {
	const sortedReports = [...reports].sort(compareReports);
	const rows: TableReport<T>[] = [];

	for (let index = 0; index < sortedReports.length; index += 1) {
		const report = sortedReports[index];
		const previous = sortedReports[index - 1];
		const categoryStart = !previous || report.primaryDivision !== previous.primaryDivision || report.workDivision !== previous.workDivision;
		const roleStart = categoryStart || report.role !== previous.role;
		const workStart = roleStart || report.work !== previous.work;

		if (workStart) rows.push({
			type: 'work',
			work: report.work || '—',
			...(categoryStart ? { category: report.workDivision, categoryRowSpan: renderedSpan(sortedReports, index, (item) => item.primaryDivision === report.primaryDivision && item.workDivision === report.workDivision) } : {}),
			...(roleStart ? { role: report.role || '—', roleRowSpan: renderedSpan(sortedReports, index, (item) => item.primaryDivision === report.primaryDivision && item.workDivision === report.workDivision && item.role === report.role) } : {})
		});
		rows.push({ type: 'report', report });
	}

	return rows;
}
