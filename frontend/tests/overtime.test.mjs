import test from 'node:test';
import {translate} from '../src/lib/i18n.ts';
import assert from 'node:assert/strict';
import {automaticDayType,calculateOvertime,calendarDates,hours,workdayPayment,loadOvertimeDraft,saveOvertimeDraft} from '../src/lib/overtime.ts';

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

test('hours always formats two decimal places without rounding durations to whole hours',()=>{
 assert.equal(hours(0),'0,00');
 assert.equal(hours(60),'1,00');
 assert.equal(hours(30),'0,50');
 assert.equal(hours(90),'1,50');
 assert.equal(hours(90,'en-GB'),'1.50');
});

test('overtime preview calculates actual and weighted minutes before save',()=>{
 assert.deepEqual(calculateOvertime('17:00','19:30','WORKDAY'),{actualMinutes:180,weightedMinutes:180});
 assert.deepEqual(calculateOvertime('08:00','10:00','HOLIDAY'),{actualMinutes:120,weightedMinutes:240});
 assert.equal(calculateOvertime('','','WORKDAY'),null);
});

test('workday preview excludes 08:00 to 17:00 on both sides of midnight',()=>{
 for(const [start,end,expected] of [
  ['08:00','21:00',240],['08:00','18:00',60],['18:00','21:00',180],
  ['17:00','18:00',60],['18:00','19:00',60],['06:00','09:00',120],
  ['08:00','17:00',0],['22:00','09:00',600],['22:00','02:00',240],
  ['19:00','18:30',900],['17:00','17:14',0],['08:00','00:00',420]
 ])assert.deepEqual(calculateOvertime(start,end,'WORKDAY'),{actualMinutes:expected,weightedMinutes:expected},`${start}-${end}`);
 assert.deepEqual(calculateOvertime('22:00','02:00','HOLIDAY'),{actualMinutes:240,weightedMinutes:480});
});

test('workday payment splits first hour and remaining hours using Excel rules',()=>{
 for(const [actualMinutes,firstMinutes,remainingMinutes] of [
  [0,0,0],[30,60,0],[60,60,0],[90,60,30],[120,60,60],[240,60,180]
 ]){
  assert.deepEqual(workdayPayment(actualMinutes,'WORKDAY'),{firstMinutes,remainingMinutes});
  assert.deepEqual(workdayPayment(actualMinutes,'HOLIDAY'),{firstMinutes:0,remainingMinutes:0});
 }
});

test('actual overtime rounds once to whole hours before payment for either day type',()=>{
 for(const dayType of ['WORKDAY','HOLIDAY']){
  for(const [end,actualMinutes] of [
   ['17:15',0],['17:29',0],['17:30',60],['17:59',60],
   ['18:00',60],['18:29',60],['18:30',120],['18:59',120]
  ])assert.deepEqual(calculateOvertime('17:00',end,dayType),{actualMinutes,weightedMinutes:actualMinutes*(dayType==='HOLIDAY'?2:1)});
 }
 assert.deepEqual(calculateOvertime('18:29','19:30','WORKDAY'),{actualMinutes:60,weightedMinutes:60});
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
