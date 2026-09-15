export const canUseDocumentChecklist = (role = '') => ['project engineer', 'finance', 'admin', 'management'].includes(role.toLowerCase());
export interface ChecklistSigner { id: number; type: 'internal' | 'external'; name: string; position: string; displayOrder: number; archived: boolean }
export interface ChecklistItem { id: number; documentName: string; digitalSigned: boolean; documentPosition: string; sentDate: string | null; notes: string; createdAt: string; updatedAt: string; signatures: Record<number, boolean> }
export interface DocumentChecklist { id: number; jobId: number; requireDigitalSignature: boolean; version: number; signers: ChecklistSigner[]; items: ChecklistItem[] }
