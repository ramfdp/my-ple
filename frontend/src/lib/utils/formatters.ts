// ponytail: use native standard Intl.NumberFormat and lightweight regex checks
export function formatRupiah(amount: number, locale = 'id-ID'): string {
	return new Intl.NumberFormat(locale, {
		style: 'currency',
		currency: 'IDR',
		maximumFractionDigits: 0
	}).format(amount);
}

export function validateIndonesianPhone(phone: string): boolean {
	// Matches: +62 812-3456-7890, +6281234567890, 081234567890, 6281234567890
	const clean = phone.replace(/[\s-]/g, '');
	return /^(?:\+62|62|0)8[1-9]\d{7,10}$/.test(clean);
}
