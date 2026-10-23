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
export const hours=(minutes:number)=>(minutes/60).toLocaleString('id-ID',{minimumFractionDigits:0,maximumFractionDigits:1});
