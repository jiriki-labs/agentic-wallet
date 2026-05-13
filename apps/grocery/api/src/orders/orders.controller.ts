import {
	Body,
	Controller,
	Headers,
	Post,
	Req,
	Res,
	HttpStatus,
} from '@nestjs/common';
import type { Request, Response } from 'express';
import { CreateOrderDto } from './dto/create-order.dto';
import { OrdersService } from './orders.service';
import { RecipesService } from '../recipes/recipes.service';
import { X402Service, type PaymentRequirements } from '../x402/x402.service';

@Controller()
export class OrdersController {
	constructor(
		private readonly orders: OrdersService,
		private readonly recipes: RecipesService,
		private readonly x402: X402Service,
	) {}

	@Post('orders')
	async createOrder(
		@Req() req: Request,
		@Res({ passthrough: true }) res: Response,
		@Body() dto: CreateOrderDto,
		@Headers('x-payment') xPayment?: string,
	) {
		const preview = this.recipes.getRecipe(dto.dish, dto.servings);
		const merchantAddr =
			process.env.MERCHANT_ADDR ??
			process.env.GROCERY_MERCHANT_ADDR ??
			'0x0000000000000000000000000000000000000000';

		const publicBase =
			process.env.GROCERY_PUBLIC_URL ??
			`${req.protocol}://${req.get('host')}`;

		const requirements = this.x402.buildRequirements({
			publicBaseUrl: publicBase,
			merchantPayTo: merchantAddr,
			maxAmountRequired: preview.totalUsdc,
			description: `ingredients for ${preview.dish} (${dto.servings} servings)`,
		});

		if (!xPayment?.trim()) {
			return this.respond402(res, requirements, {
				error: 'Payment Required',
			});
		}

		const { ok, reason } = await this.x402.verifyPaymentHeader(
			xPayment,
			requirements,
		);
		if (!ok) {
			return this.respond402(res, requirements, {
				error: 'Payment verification failed',
				detail: reason,
			});
		}

		return this.orders.confirmOrder(preview);
	}

	private respond402(
		res: Response,
		requirements: PaymentRequirements,
		partial: { error: string; detail?: string },
	) {
		res.status(HttpStatus.PAYMENT_REQUIRED);
		res.setHeader('X-Payment-Requirements', JSON.stringify(requirements));
		return {
			error: partial.error,
			...(partial.detail === undefined ? {} : { detail: partial.detail }),
			paymentRequirements: requirements,
		};
	}
}
