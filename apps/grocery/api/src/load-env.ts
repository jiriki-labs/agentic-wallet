import { config } from 'dotenv';
import { existsSync } from 'fs';
import { resolve } from 'path';

/**
 * Load Grocery402 env files before Nest bootstrap.
 * Looks for apps/grocery/.env (workspace root) and apps/grocery/api/.env (overrides).
 */
export function loadGroceryEnv(): void {
	const apiRoot = resolve(__dirname, '..');
	const groceryRoot = resolve(apiRoot, '..');

	const paths = [
		resolve(groceryRoot, '.env'),
		resolve(apiRoot, '.env'),
	];
	for (const path of paths) {
		if (existsSync(path)) {
			config({ path, override: true });
		}
	}
}
