import assert from 'node:assert/strict';
import test from 'node:test';
import { groupReports } from '../src/lib/utils/report-table.ts';

const report = (id: string, category: string, role: string, work: string, number: number) => ({
	id,
	primaryDivision: 'Project Engineer',
	workDivision: category,
	role,
	work,
	number
});

test('groups interleaved projects once and restarts numbering for each project', () => {
	const rows = groupReports([
		report('1', 'MK supervisi', 'PE', 'Project A', 1),
		report('2', 'MK supervisi', 'PE', 'Project B', 2),
		report('3', 'MK supervisi', 'PE', 'Project A', 3),
		report('4', 'MK supervisi', 'PE', 'Project C', 4)
	]);

	assert.deepEqual(rows.map((row) => row.type === 'work'
		? ['project', row.work]
		: ['item', row.report.id, row.displayNumber]), [
		['project', 'Project A'], ['item', '1', 1], ['item', '3', 2],
		['project', 'Project B'], ['item', '2', 1],
		['project', 'Project C'], ['item', '4', 1]
	]);
});

test('creates work title rows while merging the surrounding category and role cells', () => {
	const rows = groupReports([
		report('1', 'MK supervisi', 'PE', 'Penataan Kawasan Stasiun Jember', 1),
		report('2', 'MK supervisi', 'PE', 'Penataan Kawasan Stasiun Jember', 2),
		report('3', 'MK supervisi', 'PE', 'Pengawasan Renovasi Gudang Besar JAKG', 3)
	]);

	assert.deepEqual(rows.map((row) => row.type === 'work'
		? [row.type, row.work, row.categoryRowSpan, row.roleRowSpan]
		: [row.type, row.report.number]), [
		['work', 'Penataan Kawasan Stasiun Jember', 5, 5],
		['report', 1],
		['report', 2],
		['work', 'Pengawasan Renovasi Gudang Besar JAKG', undefined, undefined],
		['report', 3]
	]);
});

test('normalizes legacy QA categories before grouping', () => {
	const rows = groupReports([
		report('1', 'JAKG', 'CIHPK', 'JAKG', 1),
		report('2', 'QA 2', 'QA 2', 'MPS SGR', 1),
		report('3', 'Non-MK supervisi', 'PE', 'MPS SGR', 1),
		report('4', 'MK supervisi', 'PE', 'Penataan Kawasan Stasiun Jember', 1),
		report('5', 'QA 1', 'QA 1', 'MPS PIT', 1),
		report('6', 'Manggarai Hydrant', 'CIHPK', 'Hydrant', 1)
	]);

	assert.deepEqual(rows.filter((row) => row.type === 'work' && row.category).map((row) => row.category), [
		'MK supervisi', 'Non-MK supervisi', 'QA', 'JAKG', 'Manggarai Hydrant'
	]);
	assert.deepEqual(rows.filter((row) => row.type === 'work' && row.category === 'QA').map((row) => [row.category, row.role, row.categoryRowSpan]), [['QA', 'QA', 4]]);
});

test('displays CIHPK category reports from one even when stored numbers are stale', () => {
	const rows = groupReports([
		report('5', 'JAKG', 'CIHPK', 'JAKG Supervising Consultant', 5),
		report('6', 'JAKG', 'CIHPK', 'JAKG Supervising Consultant', 6)
	]);

	assert.deepEqual(rows.filter((row) => row.type === 'report').map((row) => row.displayNumber), [1, 2]);
});

test('restarts visible numbering for every work while keeping its later reports in sequence', () => {
	const rows = groupReports([
		report('4', 'MK supervisi', 'PE', 'Penataan Kawasan Stasiun Jember', 4),
		report('3', 'MK supervisi', 'PE', 'Pengawasan Renovasi Gudang Besar JAKG', 3),
		report('1', 'MK supervisi', 'PE', 'Penataan Kawasan Stasiun Jember', 1),
		report('2', 'MK supervisi', 'PE', 'Penataan Kawasan Stasiun Jember', 2),
		report('5', 'Non-MK supervisi', 'PE', 'Studi Daya Beli Gas Bumi', 1)
	]);

	assert.deepEqual(rows.filter((row) => row.type === 'report').map((row) => row.displayNumber), [1, 2, 3, 1, 1]);
});

test('renumbers a project after deleting an item and appending another', () => {
	const remaining = [
		report('1', 'MK supervisi', 'PE', 'Project A', 1),
		report('3', 'MK supervisi', 'PE', 'Project A', 3)
	];
	assert.deepEqual(groupReports(remaining).filter((row) => row.type === 'report').map((row) => row.displayNumber), [1, 2]);
	assert.deepEqual(groupReports([...remaining, report('4', 'MK supervisi', 'PE', 'Project A', 4)]).filter((row) => row.type === 'report').map((row) => row.displayNumber), [1, 2, 3]);
});
