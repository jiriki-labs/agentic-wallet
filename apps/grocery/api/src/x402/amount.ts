/**
 * Converts decimal USDC (e.g. "8.50") to 6-decimal atomic units for x402 verify/settle.
 * If the string has no decimal point, it is treated as already atomic.
 */
export function decimalUsdcToAtomic(amount: string): string {
	const trimmed = amount.trim();
	if (!trimmed) {
		throw new Error('empty USDC amount');
	}
	if (!trimmed.includes('.')) {
		if (!/^\d+$/.test(trimmed)) {
			throw new Error(`invalid USDC amount ${amount}`);
		}
		return trimmed;
	}
	const [wholePart, fracPart = ''] = trimmed.split('.');
	const whole = BigInt(wholePart || '0');
	const frac = (fracPart + '000000').slice(0, 6);
	const fracBig = BigInt(frac || '0');
	return (whole * 1_000_000n + fracBig).toString();
}
