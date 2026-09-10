export function commentLinks(text:string):{text:string;href?:string}[]{
 const parts:{text:string;href?:string}[]=[];
 let end=0;
 for(const match of text.matchAll(/https?:\/\/[^\s<>"']+/gi)){
  let address=match[0].replace(/[.,!?:;]+$/,'');
  while(/[)\]}]$/.test(address)){
   const close=address.at(-1)!,open=({')':'(',']':'[','}':'{'})[close]!;
   if(address.split(close).length<=address.split(open).length)break;
   address=address.slice(0,-1);
  }
  try{const url=new URL(address);if(!['http:','https:'].includes(url.protocol)||!url.hostname)continue}catch{continue}
  parts.push({text:text.slice(end,match.index)},{text:address,href:address});end=match.index+address.length;
 }
 parts.push({text:text.slice(end)});return parts;
}
