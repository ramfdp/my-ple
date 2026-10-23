export type OvertimeStatus='DRAFT'|'SUBMITTED'|'APPROVED'|'REJECTED';
export type OvertimeDayType='WORKDAY'|'HOLIDAY';
export interface OvertimeEntry {
 workDate:string;startTime:string;endTime:string;dayType:OvertimeDayType;isOverride:boolean;overrideReason:string;
 project:string;description:string;actualMinutes:number;weightedMinutes:number;
}
export interface OvertimeSheet {
 id:string;userId:string;employeeName:string;employeeRole:string;period:string;status:OvertimeStatus;version:number;
 entries:OvertimeEntry[];actualMinutes:number;weightedMinutes:number;submittedAt:string|null;reviewedBy:string|null;
 reviewerName:string;reviewedAt:string|null;reviewerNote:string;createdAt:string;updatedAt:string;
}
export interface OvertimeRosterItem {
 sheetId:string;userId:string;employeeName:string;employeeRole:string;status:OvertimeStatus|'NO_ENTRY';submittedAt:string|null;reviewedAt:string|null;
}
export const overtimeStatuses:Record<string,string>={NO_ENTRY:'Belum mengisi',DRAFT:'Draf',SUBMITTED:'Menunggu persetujuan',APPROVED:'Disetujui',REJECTED:'Perlu diperbaiki'};
export const jakartaToday=()=>new Intl.DateTimeFormat('en-CA',{timeZone:'Asia/Jakarta',year:'numeric',month:'2-digit',day:'2-digit'}).format(new Date());
export function defaultOvertimePeriod(){const today=jakartaToday();const [year,month,day]=today.split('-').map(Number);const date=new Date(Date.UTC(year,month-1-(day<=5?1:0),1));return date.toISOString().slice(0,7)}
export function calendarDates(period:string){const [year,month]=period.split('-').map(Number);const count=new Date(Date.UTC(year,month,0)).getUTCDate();return Array.from({length:count},(_,index)=>`${period}-${String(index+1).padStart(2,'0')}`)}
export const automaticDayType=(date:string):OvertimeDayType=>{const day=new Date(`${date}T00:00:00Z`).getUTCDay();return day===0||day===6?'HOLIDAY':'WORKDAY'};
export function calculateOvertime(start:string,end:string,dayType:OvertimeDayType){
 if(!/^\d{2}:\d{2}$/.test(start)||!/^\d{2}:\d{2}$/.test(end)||start===end)return null;
 const minutes=(value:string)=>{const [hour,minute]=value.split(':').map(Number);return hour*60+minute};
 const startMinute=minutes(start);let endMinute=minutes(end);if(endMinute<startMinute)endMinute+=24*60;
 let actualMinutes=dayType==='HOLIDAY'?endMinute-startMinute:Math.max(0,Math.min(endMinute,8*60)-startMinute)+Math.max(0,endMinute-Math.max(startMinute,17*60));
 actualMinutes=Math.floor((actualMinutes+15)/30)*30;
 return{actualMinutes,weightedMinutes:dayType==='HOLIDAY'?actualMinutes*2:actualMinutes};
}
type DraftStorage=Pick<Storage,'getItem'|'setItem'|'removeItem'>;
export function saveOvertimeDraft(storage:DraftStorage,key:string,rows:Record<string,unknown>){storage.setItem(key,JSON.stringify(rows))}
export function loadOvertimeDraft(storage:DraftStorage,key:string){
 try{const value=JSON.parse(storage.getItem(key)??'null');return value&&typeof value==='object'&&!Array.isArray(value)?value as Record<string,unknown>:null}catch{return null}
}
export const hours=(minutes:number)=>(minutes/60).toLocaleString('id-ID',{minimumFractionDigits:0,maximumFractionDigits:1});
