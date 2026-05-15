import { Injectable, NotFoundException } from '@nestjs/common';
import type { ScaledRecipe } from '../recipes/recipes.service';

export type ConfirmedOrder = {
	orderId: string;
	status: 'confirmed';
	dish: string;
	servings: number;
	items: ScaledRecipe['ingredients'];
	totalUsdc: string;
	eta: string;
};

@Injectable()
export class OrdersService {
	private seq = 1;
	private readonly orders = new Map<string, ConfirmedOrder>();

	confirmOrder(preview: ScaledRecipe): ConfirmedOrder {
		const id = `GRC-${String(this.seq++).padStart(3, '0')}`;
		const order: ConfirmedOrder = {
			orderId: id,
			status: 'confirmed',
			dish: preview.dish,
			servings: preview.servings,
			items: preview.ingredients,
			totalUsdc: preview.totalUsdc,
			eta: '2h delivery',
		};
		this.orders.set(id, order);
		return order;
	}

	getOrder(orderId: string): ConfirmedOrder {
		const order = this.orders.get(orderId);
		if (!order) {
			throw new NotFoundException(`order ${orderId} not found`);
		}
		return order;
	}
}
