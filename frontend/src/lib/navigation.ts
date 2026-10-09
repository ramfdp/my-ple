import { canUseDocumentChecklist } from './document-checklist.ts';
import { canUseBilling } from './billing-documents.ts';
export const features = [
 {key:'Dokumen Penagihan',href:'/billing-documents',icon:'file',description:'Rekap kesiapan dokumen penagihan per pekerjaan.'},
 {key:'Team Workspace',href:'/workspace',icon:'users',description:'Catatan, timeline, dan koordinasi pekerjaan tim.'},
 {key:'My Profile',href:'/my-profile',icon:'user',description:'Personal and employment information'},
 {key:'Absensi',href:'/attendance',icon:'calendar',description:'Catat kehadiran dan lihat riwayat kerja.'},
 {key:'Izin & Cuti',href:'/leave',icon:'clock',description:'Ajukan izin atau cuti dan pantau status pengajuan Anda.'},
 {key:'Weekly Progress',href:'/weekly-progress',icon:'file',description:'Ringkas progres dan prioritas mingguan.'},
 {key:'Booking Zoom',href:'/booking-zoom',icon:'calendar',description:'Jadwalkan ruang rapat virtual Anda.'},
 {key:'Pengaturan Pekerjaan',href:'/settings/jobs',icon:'settings',description:'Kelola pekerjaan per divisi, role, dan kategori.'},
 {key:'Employees',href:'/employees',icon:'users',description:'Kelola data karyawan.'},
 {key:'Absensi Karyawan',href:'/attendance/overview',icon:'calendar',description:'Ringkasan kehadiran'},
 {key:'Checklist Dokumen Pekerjaan',href:'/document-checklists',icon:'file',description:'Pantau dokumen dan kelengkapan tanda tangan setiap pekerjaan.'},
 {key:'Persetujuan Izin & Cuti',href:'/leave/approvals',icon:'clock',description:'Tinjau dan setujui atau tolak pengajuan izin dan cuti.'}
] as const;
export function goBack(event: Event) {
 if (history.length <= 1) return;
 event.preventDefault();
 history.back();
}
export function visibleFeatures(role = '') {
 role = role.toLowerCase();
 return features.filter(item => {
  if(['/employees','/attendance/overview'].includes(item.href))return ['admin','management','hr'].includes(role);
  if(item.href==='/document-checklists')return canUseDocumentChecklist(role);
  if(item.href==='/billing-documents')return canUseBilling(role);
  if(item.href==='/leave/approvals')return ['admin','management'].includes(role);
  return !!role;
 });
}
