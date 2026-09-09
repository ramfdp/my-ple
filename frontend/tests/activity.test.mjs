import assert from 'node:assert/strict';
import {formatActivity} from '../src/lib/activity.ts';
const activity={id:'a',taskId:'t',actorName:'mingse',action:'',field:'internal_field',oldValue:null,newValue:null,createdAt:'2026-09-09T10:00:00Z'};
for(const [action,label] of Object.entries({CREATE:'membuat pekerjaan',STATUS_CHANGE:'mengubah status pekerjaan',ASSIGN:'menetapkan penanggung jawab',UNASSIGN:'menghapus penanggung jawab',UPDATE:'memperbarui pekerjaan',COMMENT:'menambahkan komentar',ARCHIVE:'mengarsipkan pekerjaan',PRIORITY_CHANGE:'mengubah prioritas pekerjaan',DATE_CHANGE:'mengubah tanggal pekerjaan',UNKNOWN:'memperbarui pekerjaan',constructor:'memperbarui pekerjaan'})) {
 assert.equal(formatActivity({...activity,action,newValue:{internal:'secret'}}).title,`mingse ${label}`);
}
assert.equal(formatActivity({...activity,action:'CREATE',newValue:'MoM asistensi 20'}).title,'mingse membuat pekerjaan "MoM asistensi 20"');
for(const [value,label] of Object.entries({TODO:'Belum Dimulai',IN_PROGRESS:'Sedang Dikerjakan',NEED_REVISION:'Perlu Revisi',DONE:'Selesai'})) {
 assert.equal(formatActivity({...activity,action:'STATUS_CHANGE',oldValue:'TODO',newValue:value}).title,`mingse mengubah status dari Belum Dimulai menjadi ${label}`);
}
assert.equal(formatActivity({...activity,action:'STATUS_CHANGE',oldValue:'TODO',newValue:'UNKNOWN'}).title,'mingse mengubah status pekerjaan');
console.log('PASS: all actual event/status labels, create title, malformed payload and unknown/prototype-key fallback');
