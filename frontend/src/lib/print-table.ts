// Print the rendered table so column order, merged cells and status colours stay in sync.
export function printTable(table: HTMLTableElement, title: string) {
 document.querySelector('iframe[data-table-print]')?.remove();
 const frame = document.createElement('iframe');
 frame.dataset.tablePrint = '';
 frame.title = title;
 frame.style.cssText = 'position:fixed;width:0;height:0;border:0;bottom:0;right:0';
 document.body.append(frame);
 const doc = frame.contentDocument!;
 doc.title = title;
 doc.documentElement.lang = document.documentElement.lang;
 const copy = table.cloneNode(true) as HTMLTableElement;
 const originals = [table, ...table.querySelectorAll<HTMLElement>('*')];
 const clones = [copy, ...copy.querySelectorAll<HTMLElement>('*')];
 originals.forEach((original, i) => {
  const clone = clones[i], style = getComputedStyle(original);
  clone.removeAttribute('id');
  clone.removeAttribute('style');
  for (const property of ['color', 'background-color', 'font-weight', 'text-align', 'vertical-align']) {
   clone.style.setProperty(property, style.getPropertyValue(property));
  }
  if (original instanceof HTMLInputElement || original instanceof HTMLTextAreaElement || original instanceof HTMLSelectElement) {
   const text = doc.createElement('span');
   text.style.cssText = clone.style.cssText;
   if (original instanceof HTMLInputElement && original.type === 'checkbox') {
    text.textContent = original.checked ? '☑' : '☐';
   } else {
    text.textContent = original instanceof HTMLSelectElement ? original.selectedOptions[0]?.textContent ?? '' : original.value;
   }
   clone.replaceWith(text);
  }
 });
 copy.querySelectorAll('button, caption, .sr-only').forEach(el => el.remove());
 const columns = Math.max(...Array.from(table.rows, row => Array.from(row.cells).reduce((n, cell) => n + cell.colSpan, 0)));
 const style = doc.createElement('style');
 style.textContent = `
  @page { size: ${Math.max(420, columns * 26 + 20)}mm 297mm; margin: 10mm; }
  * { box-sizing: border-box; print-color-adjust: exact; -webkit-print-color-adjust: exact; }
  body { margin: 0; font: 9pt Arial, sans-serif; color: #17202a; }
  h1 { text-align: center; font-size: 14pt; margin: 0 0 8mm; }
  table { width: 100%; table-layout: fixed; border-collapse: collapse; }
  th, td { border: 1px solid #64748b; padding: 2mm; overflow-wrap: anywhere; }
  thead { display: table-header-group; }
  tr { break-inside: avoid; }
  p { margin: 0; white-space: pre-wrap; }
  small, td span { display: block; white-space: pre-wrap; }
  small { font-size: 8pt; margin-top: 1mm; }
  td span { padding: 1mm; }
 `;
 doc.head.append(style);
 const heading = doc.createElement('h1');
 heading.textContent = title;
 doc.body.append(heading, copy);
 frame.contentWindow!.onafterprint = () => frame.remove();
 frame.contentWindow!.focus();
 frame.contentWindow!.print();
}
