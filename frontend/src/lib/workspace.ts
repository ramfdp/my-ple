import { api } from '$lib/api';

export const statuses = { TODO:'To Do', IN_PROGRESS:'In Progress', NEED_REVISION:'Need Revision', DONE:'Done' } as const;
export const priorities = { LOW:'Low', MEDIUM:'Medium', HIGH:'High', URGENT:'Urgent' } as const;
export type Status=keyof typeof statuses;
export type Priority=keyof typeof priorities;
export interface Member {id:string;name:string;avatarUrl:string}
export interface Workspace {id:number;name:string;canManage:boolean}
export interface WorkspaceJob {id:number;name:string;isActive:boolean}
export interface Task {
 id:string;workspaceId:number;jobId:number|null;jobName:string;title:string;description:string;relatedUrl:string;
 startDate:string;dueDate:string;status:Status;priority:Priority;createdBy:string;createdAt:string;updatedAt:string;version:number;
 assignees:Member[];commentCount:number;canEdit:boolean;canDelete:boolean;
}
export interface TaskInput {title:string;description:string;relatedUrl:string;jobId:number|null;startDate:string;dueDate:string;status:Status;priority:Priority;assigneeIds:string[];version:number}
export interface Comment {id:string;userId:string;name:string;content:string;createdAt:string}
export interface Activity {id:string;taskId:string;actorName:string;action:string;field:string;oldValue:unknown;newValue:unknown;createdAt:string}
export interface Workload {member:Member;activeTaskCount:number;highPriorityCount:number;dueSoonCount:number;overdueCount:number;overlapCount:number;workloadScore:number;workloadLevel:'AVAILABLE'|'NORMAL'|'BUSY'|'OVERLOADED';tasks:Task[]}
export interface PageLine {kind:'text'|'heading'|'bullet'|'check'|'link'|'bold'|'italic';text:string;checked:boolean;url?:string}
export interface Note {id:string;workspaceId:number;parentId:string|null;title:string;content:PageLine[];visibility:'PERSONAL'|'DIVISION';createdBy:string;updatedAt:string;version:number;canEdit:boolean}
export type NoteInput=Pick<Note,'parentId'|'title'|'content'|'visibility'|'version'>;

export const today=()=>new Intl.DateTimeFormat('en-CA',{timeZone:'Asia/Jakarta',year:'numeric',month:'2-digit',day:'2-digit'}).format(new Date());
export const dateValue=(s:string)=>new Date(s+'T00:00:00Z');
export const dateString=(d:Date)=>d.toISOString().slice(0,10);
export const addDays=(s:string,n:number)=>{const d=dateValue(s);d.setUTCDate(d.getUTCDate()+n);return dateString(d)};
export const labelDate=(s:string)=>s?dateValue(s).toLocaleDateString('id-ID',{day:'numeric',month:'short',year:'numeric',timeZone:'UTC'}):'Tanpa tanggal';
export function period(anchor:string,scale:'week'|'month'){
 const d=dateValue(anchor);
 const start=scale==='month'?dateString(new Date(Date.UTC(d.getUTCFullYear(),d.getUTCMonth(),1))):addDays(anchor,-((d.getUTCDay()+6)%7));
 const end=scale==='month'?dateString(new Date(Date.UTC(d.getUTCFullYear(),d.getUTCMonth()+1,0))):addDays(start,6);
 const days:string[]=[];for(let day=start;day<=end;day=addDays(day,1))days.push(day);
 return {start,end,days};
}
export function taskSpan(t:Pick<Task,'startDate'|'dueDate'>,start:string,end:string){
 const a=t.startDate||t.dueDate,b=t.dueDate||t.startDate;
 if(!a||!b||a>end||b<start)return null;
 return {start:a<start?start:a,end:b>end?end:b};
}
export function taskInput(t:Task):TaskInput{return {title:t.title,description:t.description,relatedUrl:t.relatedUrl,jobId:t.jobId,startDate:t.startDate,dueDate:t.dueDate,status:t.status,priority:t.priority,assigneeIds:t.assignees.map(a=>a.id),version:t.version}}
export const updateTask=(id:string,input:Partial<TaskInput>)=>api<{task:Task}>('/work-items/'+id,{method:'PATCH',body:JSON.stringify(input)});
