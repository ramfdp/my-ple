import { translate, type Language } from './i18n';
import type { Activity } from './workspace';

const statusLabels: Record<string, string> = {
 TODO: 'Belum Dimulai', IN_PROGRESS: 'Sedang Dikerjakan', NEED_REVISION: 'Perlu Revisi', DONE: 'Selesai'
};
const actionLabels: Record<string, string> = {
 CREATE: 'membuat pekerjaan', STATUS_CHANGE: 'mengubah status pekerjaan',
 ASSIGN: 'menetapkan penanggung jawab', UNASSIGN: 'menghapus penanggung jawab',
 UPDATE: 'memperbarui pekerjaan', COMMENT: 'menambahkan komentar', ARCHIVE: 'mengarsipkan pekerjaan', RESTORE: 'memulihkan pekerjaan',
 PRIORITY_CHANGE: 'mengubah prioritas pekerjaan', DATE_CHANGE: 'mengubah tanggal pekerjaan'
};

export function formatActivity(activity: Activity, language: Language = 'id') {
 let message = translate(Object.hasOwn(actionLabels, activity.action) ? actionLabels[activity.action] : 'memperbarui pekerjaan', language);
 if (activity.action === 'CREATE' && typeof activity.newValue === 'string' && activity.newValue.trim()) {
  message += ` "${activity.newValue}"`;
 }
 if (activity.action === 'STATUS_CHANGE' && typeof activity.oldValue === 'string' && typeof activity.newValue === 'string'
  && Object.hasOwn(statusLabels, activity.oldValue) && Object.hasOwn(statusLabels, activity.newValue)) {
  message = translate('mengubah status dari {old} menjadi {new}',language,{old:translate(statusLabels[activity.oldValue],language),new:translate(statusLabels[activity.newValue],language)});
 }
 if (activity.field === 'revisionCount' && typeof activity.newValue === 'number') {
  message = translate('meminta revisi {revision}', language, {revision: `revisi-${activity.newValue}`});
 }
 return { title: `${activity.actorName || translate('Pengguna',language)} ${message}`, timestamp: new Date(activity.createdAt).toLocaleString(language==='en'?'en-GB':'id-ID') };
}
