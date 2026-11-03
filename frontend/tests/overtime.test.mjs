import test from 'node:test';
import {translate} from '../src/lib/i18n.ts';
import assert from 'node:assert/strict';
import {automaticDayType,calculateOvertime,calendarDates,defaultOvertimePeriod,watchOvertimePeriod,hours,loadOvertimeDraft,saveOvertimeDraft} from '../src/lib/overtime.ts';

test('default period follows the Jakarta month including days 1 to 5 and new year',t=>{
 t.mock.timers.enable({apis:['Date'],now:0});
 for(const [instant,period] of [
  ['2026-10-31T16:59:59Z','2026-10'],
  ['2026-10-31T17:00:00Z','2026-11'],
  ['2026-11-02T03:00:00Z','2026-11'],
  ['2026-11-05T16:59:59Z','2026-11'],
  ['2026-11-06T00:00:00Z','2026-11'],
  ['2026-12-31T17:00:00Z','2027-01']
 ]){
  t.mock.timers.setTime(Date.parse(instant));
  assert.equal(defaultOvertimePeriod(),period,instant);
 }
});

test('period watcher refreshes across midnight and on focus and cleans up',t=>{
 t.mock.timers.enable({apis:['Date','setInterval'],now:Date.parse('2026-10-31T16:59:30Z')});
 const originalWindow=globalThis.window;
 const target=new EventTarget();
 globalThis.window=target;
 t.after(()=>{if(originalWindow===undefined)delete globalThis.window;else globalThis.window=originalWindow});
 const periods=[];
 const stop=watchOvertimePeriod(period=>periods.push(period));
 assert.equal(periods.at(-1),'2026-10');
 t.mock.timers.tick(60_000);
 assert.equal(periods.at(-1),'2026-11');
 t.mock.timers.setTime(Date.parse('2026-11-30T17:00:00Z'));
 target.dispatchEvent(new Event('focus'));
 assert.equal(periods.at(-1),'2026-12');
 stop();
 const count=periods.length;
 t.mock.timers.tick(60_000);
 target.dispatchEvent(new Event('focus'));
 assert.equal(periods.length,count);
});

test('calendarDates covers leap years and short months',()=>{
 assert.equal(calendarDates('2024-02').length,29);
 assert.equal(calendarDates('2025-02').length,28);
 assert.equal(calendarDates('2026-04').at(-1),'2026-04-30');
 assert.equal(calendarDates('2026-10').at(-1),'2026-10-31');
});

test('automaticDayType marks weekends only',()=>{
 assert.equal(automaticDayType('2026-10-10'),'HOLIDAY');
 assert.equal(automaticDayType('2026-10-12'),'WORKDAY');
});

test('hours formats half hours',()=>{
 assert.equal(hours(90),'1,5');
 assert.equal(hours(90,'en-GB'),'1.5');
});

test('overtime preview calculates actual and weighted minutes before save',()=>{
 assert.deepEqual(calculateOvertime('17:00','19:30','WORKDAY'),{actualMinutes:90,weightedMinutes:90});
 assert.deepEqual(calculateOvertime('08:00','10:00','HOLIDAY'),{actualMinutes:120,weightedMinutes:240});
 assert.equal(calculateOvertime('','','WORKDAY'),null);
});

test('workday preview excludes 08:00 to 18:00 on both sides of midnight',()=>{
 for(const [start,end,expected] of [
  ['17:00','18:00',0],['18:00','19:00',60],['06:00','09:00',120],
  ['08:00','17:00',0],['22:00','09:00',600],['22:00','02:00',240],
  ['19:00','18:30',810],['18:00','18:15',30],['18:00','18:14',0]
 ])assert.deepEqual(calculateOvertime(start,end,'WORKDAY'),{actualMinutes:expected,weightedMinutes:expected},`${start}-${end}`);
 assert.deepEqual(calculateOvertime('22:00','02:00','HOLIDAY'),{actualMinutes:240,weightedMinutes:480});
});

test('unsaved overtime survives page recreation',()=>{
 const values=new Map();
 const storage={getItem:key=>values.get(key)??null,setItem:(key,value)=>values.set(key,value),removeItem:key=>values.delete(key)};
 const rows={'2026-10-27':{workDate:'2026-10-27',startTime:'17:00',endTime:'19:00',dayType:'WORKDAY',project:'PLE',description:'Testing'}};
 saveOvertimeDraft(storage,'overtime:u1:2026-10',rows);
 assert.deepEqual(loadOvertimeDraft(storage,'overtime:u1:2026-10'),rows);
});

test('overtime messages translate statuses, errors and parameters in both languages',()=>{
 assert.equal(translate('Menunggu persetujuan','en'),'Awaiting approval');
 assert.equal(translate('Lengkapi data lembur {date}.','en',{date:'2026-10-27'}),'Complete the overtime data for 2026-10-27.');
 assert.equal(translate('Draf lembur tersimpan.','id'),'Draf lembur tersimpan.');
 assert.equal(translate('Draf lembur tersimpan.','en'),'Overtime draft saved.');
 assert.equal(translate('{employee} mengajukan lembur periode {period}','en',{employee:'Ayu',period:'2026-10'}),'Ayu submitted overtime for 2026-10');
 assert.equal(translate('Data lembur tidak valid.','en'),'Invalid overtime data.');
});
