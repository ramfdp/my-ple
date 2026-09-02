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

test('keeps a category sequence ordered and resets it for a new category', () => {
	const rows = groupReports([
		report('4', 'MK supervisi', 'PE', 'Penataan Kawasan Stasiun Jember', 4),
		report('3', 'MK supervisi', 'PE', 'Pengawasan Renovasi Gudang Besar JAKG', 3),
		report('1', 'MK supervisi', 'PE', 'Penataan Kawasan Stasiun Jember', 1),
		report('2', 'MK supervisi', 'PE', 'Penataan Kawasan Stasiun Jember', 2),
		report('5', 'Non-MK supervisi', 'PE', 'Studi Daya Beli Gas Bumi', 1)
	]);

	assert.deepEqual(rows.filter((row) => row.type === 'report').map((row) => row.report.number), [1, 2, 3, 4, 1]);
});
