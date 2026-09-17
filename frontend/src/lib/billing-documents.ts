export const billingStatuses = ['BELUM ADA','PEMBUATAN','KURANG LENGKAP','SIRKULIR TTD','OK'] as const;
export const documentSuggestions = ['Lembar Pengesahan','Daftar Hadir','Daftar Kegiatan','RPHP/BA Pemeriksaan','DPHP/DPL/DPB','BA Pengujian','BAPP','BAST','RAB Tagihan'];
export const canUseBilling = (role = '') => ['admin','finance','project engineer','management'].includes(role.toLowerCase());
export type Readiness = 'SIAP' | 'BELUM SIAP';
export interface BillingDocument {id:number;name:string;status:typeof billingStatuses[number];updatedAt:string}
export interface BillingMilestone {id:number;name:string;notes:string;updatedAt:string;documents:BillingDocument[];readiness:Readiness}
export interface BillingStage {id:number;jobId:number;name:string;periodLabel:string;notes:string;updatedAt:string;version:number;milestones:BillingMilestone[];readiness:Readiness}
export interface BillingConfiguration {jobId:number;name:string;periodLabel:string;version:number;milestones:{id:number;name:string;documents:{id:number;name:string}[]}[]}
