export const leaveTypes = {CUTI:'Cuti',IZIN:'Izin'} as const;
export const leaveStatuses = {PENDING:'Menunggu Persetujuan',APPROVED:'Disetujui',REJECTED:'Ditolak'} as const;
export interface LeaveRequest {
 id:string;userId:string;employeeName:string;division:string;requestType:keyof typeof leaveTypes;
 startDate:string;endDate:string;reason:string;notes:string;status:keyof typeof leaveStatuses;
 reviewedBy:string|null;reviewerName:string;reviewedAt:string|null;reviewerNote:string;createdAt:string;updatedAt:string;
}
export interface LeaveInput {requestId:string;requestType:keyof typeof leaveTypes;startDate:string;endDate:string;reason:string;notes:string}
