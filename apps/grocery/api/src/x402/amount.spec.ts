import { decimalUsdcToAtomic } from './amount';

describe('decimalUsdcToAtomic', () => {
	it('converts decimal USDC to atomic units', () => {
		expect(decimalUsdcToAtomic('8.50')).toBe('8500000');
		expect(decimalUsdcToAtomic('0.30')).toBe('300000');
	});

	it('passes through atomic strings', () => {
		expect(decimalUsdcToAtomic('8500000')).toBe('8500000');
	});
});
