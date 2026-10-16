export function planPath(workspaceId: number, id?: string, date?: string): string {
 const base = `/ai-plans/workspaces/${workspaceId}`;
 return id ? `${base}/${id}` : date ? `${base}?date=${encodeURIComponent(date)}` : base;
}

export async function saveReviewAndRefresh(write: () => Promise<unknown>, refresh: () => Promise<boolean>): Promise<'saved' | 'refresh-failed'> {
 await write();
 return await refresh() ? 'saved' : 'refresh-failed';
}
