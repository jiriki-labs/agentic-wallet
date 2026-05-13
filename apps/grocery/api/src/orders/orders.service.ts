import { Injectable } from '@nestjs/common';
import type { ScaledRecipe } from '../recipes/recipes.service';

@Injectable()
export class OrdersService {
	private seq = 1;

	confirmOrder(preview: ScaledRecipe) {
		const id = `GRC-${String(this.seq++).padStart(3, '0')}`;
		return {
			orderId: id,
			status: 'confirmed' as const,
			items: preview.ingredients,
			totalUsdc: preview.totalUsdc,
			eta: '2h delivery',
		};
	}
}
