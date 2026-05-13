import { Controller, Get, Query } from '@nestjs/common';
import { RecipesService } from './recipes.service';

@Controller()
export class RecipesController {
	constructor(private readonly recipes: RecipesService) {}

	/** Free endpoint: recipe + ingredient prices in USDC (for base servings scaling). */
	@Get('recipes')
	getRecipe(
		@Query('dish') dish: string,
		@Query('servings') servingsRaw?: string,
	) {
		const servings = Math.max(
			1,
			Number.parseInt(servingsRaw ?? '2', 10) || 2,
		);
		return this.recipes.getRecipe(dish, servings);
	}
}
