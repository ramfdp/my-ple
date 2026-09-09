import type { Activity } from './workspace';

const statusLabels: Record<string, string> = {
 TODO: 'Belum Dimulai', IN_PROGRESS: 'Sedang Dikerjakan', NEED_REVISION: 'Perlu Revisi', DONE: 'Selesai'
};
const actionLabels: Record<string, string> = {
 CREATE: 'membuat pekerjaan', STATUS_CHANGE: 'mengubah status pekerjaan',
 ASSIGN: 'menetapkan penanggung jawab', UNASSIGN: 'menghapus penanggung jawab',
 UPDATE: 'memperbarui pekerjaan', COMMENT: 'menambahkan komentar', ARCHIVE: 'mengarsipkan pekerjaan',
 PRIORITY_CHANGE: 'mengubah prioritas pekerjaan', DATE_CHANGE: 'mengubah tanggal pekerjaan'
};

export function formatActivity(activity: Activity) {
 let message = Object.hasOwn(actionLabels, activity.action) ? actionLabels[activity.action] : 'memperbarui pekerjaan';
 if (activity.action === 'CREATE' && typeof activity.newValue === 'string' && activity.newValue.trim()) {
  message += ` "${activity.newValue}"`;
 }
 if (activity.action === 'STATUS_CHANGE' && typeof activity.oldValue === 'string' && typeof activity.newValue === 'string'
  && Object.hasOwn(statusLabels, activity.oldValue) && Object.hasOwn(statusLabels, activity.newValue)) {
  message = `mengubah status dari ${statusLabels[activity.oldValue]} menjadi ${statusLabels[activity.newValue]}`;
 }
 return { title: `${activity.actorName || 'Pengguna'} ${message}`, timestamp: new Date(activity.createdAt).toLocaleString('id-ID') };
}
