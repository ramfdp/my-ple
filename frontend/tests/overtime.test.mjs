import test from 'node:test';
import assert from 'node:assert/strict';
import {automaticDayType,calendarDates,hours} from '../src/lib/overtime.ts';

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
