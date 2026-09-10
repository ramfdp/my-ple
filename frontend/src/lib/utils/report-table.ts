import type { JobPIC } from '$lib/api';
export interface GroupableReport {
 id?:string;
 pics?:JobPIC[];
	primaryDivision: string;
	workDivision: string;
	role: string;
	work: string;
	number: number;
	completedAt?: string;
}

export type TableReport<T extends GroupableReport> =
	| { type: 'work'; key:string; work: string; pics?:JobPIC[]; category?: string; role?: string; categoryRowSpan?: number; roleRowSpan?: number }
	| { type: 'report'; report: T; displayNumber: number };

export const canModifyReport = (user: { id: string; role: string } | null | undefined, _ownerID: string) => !!user;

const projectEngineerCategoryOrder = ['MK supervisi', 'Non-MK supervisi', 'QA1', 'QA2', 'JAKG', 'Manggarai Hydrant', 'Jember'];

const canonicalQA = (value: string) => ['QA 1','QA 2','QA1','QA2'].includes(value) ? 'QA' : value;

const groupKey = (report: GroupableReport) => `${report.primaryDivision}\u0000${report.workDivision}\u0000${report.role}\u0000${report.work}`;

function compareReports(a: GroupableReport, b: GroupableReport) {
	const compare=(a:string,b:string)=>a<b?-1:a>b?1:0;
	const divisionComparison = compare(a.primaryDivision,b.primaryDivision);
	if (divisionComparison !== 0) return divisionComparison;
	const rank:Record<string,number>={PE:1,QA:2,CIHPK:3};
	const roleComparison=(rank[a.role]??0)-(rank[b.role]??0);
	if(roleComparison)return roleComparison;

	const categoryRank = (category: string) => {
		const rank = projectEngineerCategoryOrder.indexOf(category);
		return rank === -1 ? -1 : rank;
	};
	const categoryComparison = categoryRank(a.workDivision) - categoryRank(b.workDivision) || compare(a.workDivision,b.workDivision);
	if (categoryComparison !== 0) return categoryComparison;

	const workComparison = compare(a.work,b.work);
	if (workComparison !== 0) return workComparison;
	return a.number - b.number || compare(a.completedAt??'',b.completedAt??'') || compare(a.id??'',b.id??'');
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
	const normalized = reports.map((report) => ({ ...report, role: canonicalQA(report.role), workDivision: report.workDivision==='QA'?'QA1':report.workDivision.replace(/^QA ([12])$/,'QA$1') }));
	const sortedReports = normalized.sort(compareReports);
	const rows: TableReport<T>[] = [];
	let displayNumber=0;

	for (let index = 0; index < sortedReports.length; index += 1) {
		const report = sortedReports[index];
		const previous = sortedReports[index - 1];
		const categoryStart = !previous || report.primaryDivision !== previous.primaryDivision || report.workDivision !== previous.workDivision;
		const roleStart = categoryStart || report.role !== previous.role;
		const workStart = roleStart || report.work !== previous.work;
		displayNumber=workStart?1:displayNumber+1;

		if (workStart) rows.push({
			type: 'work',
			key: groupKey(report),
            ...(report.pics?.length ? {pics:report.pics} : {}),
			work: report.work || '—',
			...(categoryStart ? { category: report.workDivision, categoryRowSpan: renderedSpan(sortedReports, index, (item) => item.primaryDivision === report.primaryDivision && item.workDivision === report.workDivision) } : {}),
			...(roleStart ? { role: report.role || '—', roleRowSpan: renderedSpan(sortedReports, index, (item) => item.primaryDivision === report.primaryDivision && item.workDivision === report.workDivision && item.role === report.role) } : {})
		});
		rows.push({ type: 'report', report, displayNumber });
	}

	return rows;
}
