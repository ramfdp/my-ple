export function planPath(workspaceId: number, id?: string, date?: string): string {
 const base = `/ai-plans/workspaces/${workspaceId}`;
 return id ? `${base}/${id}` : date ? `${base}?date=${encodeURIComponent(date)}` : base;
}

export function isOneDriveLink(raw: string | undefined): boolean {
 if (!raw) return false;
 try {
  const url = new URL(raw);
  const host = url.hostname.toLowerCase();
  return url.protocol === 'https:' && !url.username && !url.password && !url.port &&
   (host === '1drv.ms' || host === 'onedrive.live.com' || host.endsWith('.sharepoint.com'));
 } catch { return false; }
}

export async function saveReviewAndRefresh(write: () => Promise<unknown>, refresh: () => Promise<boolean>): Promise<'saved' | 'refresh-failed'> {
 await write();
 return await refresh() ? 'saved' : 'refresh-failed';
}
