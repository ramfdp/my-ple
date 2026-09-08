import type { JobPIC } from '$lib/api';
export interface GroupableReport {
 pics?:JobPIC[];
	primaryDivision: string;
	workDivision: string;
	role: string;
	work: string;
	number: number;
	completedAt?: string;
}

export type TableReport<T extends GroupableReport> =
	| { type: 'work'; work: string; pics?:JobPIC[]; category?: string; role?: string; categoryRowSpan?: number; roleRowSpan?: number }
	| { type: 'report'; report: T; displayNumber: number };

export const canModifyReport = (user: { id: string; role: string } | null | undefined, _ownerID: string) => !!user;

const projectEngineerCategoryOrder = ['MK supervisi', 'Non-MK supervisi', 'QA'];

const canonicalQA = (value: string) => value === 'QA 1' || value === 'QA 2' ? 'QA' : value;

const groupKey = (report: GroupableReport) => `${report.primaryDivision}\u0000${report.workDivision}\u0000${report.role}\u0000${report.work}`;

function compareReports(a: GroupableReport, b: GroupableReport, groupOrder: Map<string, number>) {
	const divisionComparison = a.primaryDivision.localeCompare(b.primaryDivision, 'id');
	if (divisionComparison !== 0) return divisionComparison;

	const categoryRank = (category: string) => {
		const rank = projectEngineerCategoryOrder.indexOf(category);
		return rank === -1 ? projectEngineerCategoryOrder.length : rank;
	};
	const categoryComparison = categoryRank(a.workDivision) - categoryRank(b.workDivision) || a.workDivision.localeCompare(b.workDivision, 'id');
	if (categoryComparison !== 0) return categoryComparison;

	const roleComparison = a.role.localeCompare(b.role, 'id');
	if (roleComparison !== 0) return roleComparison;
	const workComparison = (groupOrder.get(groupKey(a)) ?? 0) - (groupOrder.get(groupKey(b)) ?? 0) || a.work.localeCompare(b.work, 'id');
	if (workComparison !== 0) return workComparison;
	return a.number - b.number || (a.completedAt ?? '').localeCompare(b.completedAt ?? '');
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
	const normalized = reports.map((report) => ({ ...report, role: canonicalQA(report.role), workDivision: canonicalQA(report.workDivision) }));
	const groupOrder = new Map<string, number>();
	for (const report of normalized) groupOrder.set(groupKey(report), Math.min(groupOrder.get(groupKey(report)) ?? Infinity, report.number));
	const sortedReports = normalized.sort((a, b) => compareReports(a, b, groupOrder));
	const rows: TableReport<T>[] = [];

	for (let index = 0; index < sortedReports.length; index += 1) {
		const report = sortedReports[index];
		const previous = sortedReports[index - 1];
		const displayNumber = report.number;
		const categoryStart = !previous || report.primaryDivision !== previous.primaryDivision || report.workDivision !== previous.workDivision;
		const roleStart = categoryStart || report.role !== previous.role;
		const workStart = roleStart || report.work !== previous.work;

		if (workStart) rows.push({
			type: 'work',
            ...(report.pics?.length ? {pics:report.pics} : {}),
			work: report.work || '—',
			...(categoryStart ? { category: report.workDivision, categoryRowSpan: renderedSpan(sortedReports, index, (item) => item.primaryDivision === report.primaryDivision && item.workDivision === report.workDivision) } : {}),
			...(roleStart ? { role: report.role || '—', roleRowSpan: renderedSpan(sortedReports, index, (item) => item.primaryDivision === report.primaryDivision && item.workDivision === report.workDivision && item.role === report.role) } : {})
		});
		rows.push({ type: 'report', report, displayNumber });
	}

	return rows;
}
