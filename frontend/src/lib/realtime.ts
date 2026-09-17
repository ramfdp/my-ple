export interface Change { workspaceId?:number; entityId?:string; revision?:number; event?:string }

// All views share the socket owned by the authenticated layout.
export function subscribeChanges(events:string[], refresh:(change:Change)=>void|Promise<void>, accepts:(change:Change)=>boolean=()=>true) {
 let active=true, running=false;
 const pending=new Map<string,Change>();
 const seen=new Set<string>();
 const run=async(change:Change)=>{
  if(change.event==='realtime.resync')pending.clear();
  pending.set(JSON.stringify([change.event,change.workspaceId,change.entityId]),change);
  if(running)return;
  running=true;
  try { while(active&&pending.size){const [key,next]=pending.entries().next().value!;pending.delete(key);try{await refresh(next)}catch{/* Retry on the next event; continue other pending updates. */}} }
  finally {running=false}
 };
 const listener=(e:Event)=>{const change={...((e as CustomEvent<Change>).detail??{}),event:e.type};if(change.event!=='realtime.resync'&&!accepts(change))return;if(change.revision){const key=JSON.stringify(change);if(seen.has(key))return;seen.add(key);if(seen.size>100)seen.delete(seen.values().next().value!)}void run(change)};
 for(const name of [...events,'realtime.resync'])window.addEventListener(name,listener);
 return()=>{active=false;pending.clear();for(const name of [...events,'realtime.resync'])window.removeEventListener(name,listener)};
}
