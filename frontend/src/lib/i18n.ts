import { derived, writable } from 'svelte/store';
import { messages, type MessageKey } from './messages.ts';
export type Language = 'id' | 'en';
export const language = writable<Language>('id');
export const locale = derived(language, value => value === 'en' ? 'en-GB' : 'id-ID');
export function setLanguage(value: string) {
 const next = value === 'en' ? 'en' : 'id';
 language.set(next);
 if(typeof document !== 'undefined') document.documentElement.lang = next;
 try { if(typeof localStorage !== 'undefined') localStorage.setItem('language',next); } catch { /* Storage may be disabled. */ }
}
export function restoreLanguage() {
 try { setLanguage(localStorage.getItem('language') ?? 'id'); } catch { setLanguage('id'); }
}
const enumLabels: Record<string,string> = {LOW:'Low',MEDIUM:'Medium',HIGH:'High',URGENT:'Urgent'};
export function translate(key: string, lang: Language, params: Record<string,string|number> = {}): string {
 const source = Object.hasOwn(enumLabels,key) ? enumLabels[key] : key;
 const pair = Object.hasOwn(messages,source) ? messages[source as MessageKey] : undefined;
 const text = pair ? pair[lang === 'en' ? 1 : 0] : key;
 return text.replace(/\{(\w+)\}/g, (placeholder, name) => Object.hasOwn(params,name) ? String(params[name]) : placeholder);
}
export const t = derived(language, value => (key: string, params?: Record<string,string|number>) => translate(key,value,params));
export const hasMessage = (key: string) => Object.hasOwn(messages,key);
