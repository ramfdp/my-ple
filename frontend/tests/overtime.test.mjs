import test from 'node:test';
import assert from 'node:assert/strict';
import {automaticDayType,calculateOvertime,calendarDates,hours,loadOvertimeDraft,saveOvertimeDraft} from '../src/lib/overtime.ts';

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
});

test('overtime preview calculates actual and weighted minutes before save',()=>{
 assert.deepEqual(calculateOvertime('17:00','19:30','WORKDAY'),{actualMinutes:150,weightedMinutes:150});
 assert.deepEqual(calculateOvertime('08:00','10:00','HOLIDAY'),{actualMinutes:120,weightedMinutes:240});
 assert.equal(calculateOvertime('','','WORKDAY'),null);
});

test('unsaved overtime survives page recreation',()=>{
 const values=new Map();
 const storage={getItem:key=>values.get(key)??null,setItem:(key,value)=>values.set(key,value),removeItem:key=>values.delete(key)};
 const rows={'2026-10-27':{workDate:'2026-10-27',startTime:'17:00',endTime:'19:00',dayType:'WORKDAY',project:'PLE',description:'Testing'}};
 saveOvertimeDraft(storage,'overtime:u1:2026-10',rows);
 assert.deepEqual(loadOvertimeDraft(storage,'overtime:u1:2026-10'),rows);
});
