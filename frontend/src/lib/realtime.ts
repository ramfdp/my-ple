export interface Change { workspaceId?:number; entityId?:string; revision?:number; event?:string }

// All views share the socket owned by the authenticated layout.
export function subscribeChanges(events:string[], refresh:(change:Change)=>void|Promise<void>) {
 let active=true, running=false, pending:Change|null=null;
 const seen=new Set<string>();
 const run=async(change:Change)=>{
  pending=change;
  if(running)return;
  running=true;
  try { while(active&&pending){const next=pending;pending=null;await refresh(next)} }
  catch { /* Views display their HTTP errors and may retry on the next event. */ }
  finally {running=false}
 };
 const listener=(e:Event)=>{const change={...((e as CustomEvent<Change>).detail??{}),event:e.type};if(change.revision){const key=JSON.stringify(change);if(seen.has(key))return;seen.add(key);if(seen.size>100)seen.delete(seen.values().next().value!)}void run(change)};
 for(const name of [...events,'realtime.resync'])window.addEventListener(name,listener);
 return()=>{active=false;pending=null;for(const name of [...events,'realtime.resync'])window.removeEventListener(name,listener)};
}
