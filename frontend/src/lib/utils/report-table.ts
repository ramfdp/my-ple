import type { JobPIC, Report, WeeklyRoleGroup } from '$lib/api';

export type TableReport =
	| { type: 'work'; key:string; work:string; pics?:JobPIC[]; category?:string; role?:string; categoryRowSpan?:number; roleRowSpan?:number }
	| { type: 'report'; report:Report; displayNumber:number };

export const canModifyReport = (user: { id:string; role:string } | null | undefined, _ownerID:string) => !!user;

const categorySpan = (category:WeeklyRoleGroup['categories'][number]) => category.jobs.reduce((rows, job) => rows + 1 + job.progress.length, 0);

export function groupReports(groups:WeeklyRoleGroup[]):TableReport[] {
	const rows:TableReport[]=[];
	for(const role of groups){
		let roleFirst=true;
		const roleRowSpan=role.categories.reduce((total,category)=>total+categorySpan(category),0);
		for(const category of role.categories){
			let categoryFirst=true;
			const categoryRowSpan=categorySpan(category);
			for(const job of category.jobs){
				rows.push({type:'work',key:`${role.name}\u0000${category.name}\u0000${job.key}`,work:job.name,...(job.pics?.length?{pics:job.pics}:{}),...(roleFirst?{role:role.name,roleRowSpan}:{}),...(categoryFirst?{category:category.name,categoryRowSpan}:{})});
				roleFirst=false;categoryFirst=false;
				for(const progress of job.progress)rows.push({type:'report',report:progress.report,displayNumber:progress.number});
			}
		}
	}
	return rows;
}
