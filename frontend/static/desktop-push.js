function desktopMessage({kind, entity, workspace, language}) {
 const en = language === 'en';
 if (kind === 'attendance') return [en ? 'Attendance reminder' : 'Pengingat absensi', '/attendance'];
 if (kind === 'zoom_booking_created') return [en ? 'Zoom booking update' : 'Pembaruan booking Zoom', '/booking-zoom'];
 if (kind.startsWith('leave_request_')) {
  const path = kind === 'leave_request_created' ? '/leave/approvals' : '/leave';
  return [en ? 'Leave update' : 'Pembaruan cuti', path + '?request=' + encodeURIComponent(entity)];
 }
 if (kind.startsWith('overtime_')) return [en ? 'Overtime update' : 'Pembaruan lembur', kind === 'overtime_submitted' ? '/overtime/approvals' : '/overtime'];
 if (kind === 'work_item_assigned' || kind === 'workspace_task') return [en ? 'Work update' : 'Pembaruan pekerjaan', /^\d+$/.test(workspace) ? '/workspace/' + workspace : '/workspace'];
 return [en ? 'New notification' : 'Notifikasi baru', '/dashboard'];
}

self.addEventListener('push', event => {
 if (!event.data) return;
 const payload = event.data.json();
 const [body, path] = desktopMessage(payload);
 event.waitUntil(self.registration.showNotification('My PLE', {body, tag:payload.id, data:{path}}));
});

self.addEventListener('notificationclick', event => {
 event.notification.close();
 const target = new URL(event.notification.data.path, self.location.origin);
 if (target.origin !== self.location.origin) return;
 event.waitUntil((async () => {
  const windows = await self.clients.matchAll({type:'window',includeUncontrolled:true});
  const existing = windows.find(client => client.url === target.href);
  if (existing) return existing.focus();
  return self.clients.openWindow(target.href);
 })());
});

