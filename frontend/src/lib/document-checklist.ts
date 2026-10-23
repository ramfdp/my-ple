export const canUseDocumentChecklist = (role = '') => ['project engineer', 'finance', 'admin', 'management'].includes(role.toLowerCase());
export interface ChecklistSigner { id: number; type: 'internal' | 'external'; name: string; position: string; displayOrder: number; archived: boolean }
export interface ChecklistItem { id: number; parentId: number | null; number: string; documentName: string; digitalSigned: boolean; documentPosition: string; sentDate: string | null; notes: string; createdAt: string; updatedAt: string; signatures: Record<number, boolean> }
export interface ChecklistNewItem { clientId: string; parentId: number; parentClientId: string; documentName: string }
export interface DocumentChecklist { title: string; headerLabels: Record<string, string>; id: number; jobId: number; requireDigitalSignature: boolean; version: number; signers: ChecklistSigner[]; items: ChecklistItem[] }
