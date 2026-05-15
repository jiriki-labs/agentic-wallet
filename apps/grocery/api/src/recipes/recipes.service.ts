import { Injectable, NotFoundException } from '@nestjs/common';

export interface Ingredient {
	name: string;
	amount: string;
	priceUsdc: string;
}

export interface ScaledRecipe {
	dish: string;
	servings: number;
	ingredients: Ingredient[];
	totalUsdc: string;
}

export interface Recipe {
	dish: string;
	baseServings: number;
	ingredients: Ingredient[];
}

/** USDC prices are for baseServings (default 2). */
const CATALOG: Record<string, Recipe> = {
	carbonara: {
		dish: 'carbonara',
		baseServings: 2,
		ingredients: [
			{ name: 'pancetta', amount: '150g', priceUsdc: '0.10' },
			{ name: 'eggs', amount: '4', priceUsdc: '0.20' },
			{ name: 'pecorino', amount: '50g', priceUsdc: '0.30' },
			{ name: 'spaghetti', amount: '200g', priceUsdc: '0.50' },
			{ name: 'black pepper', amount: '5g', priceUsdc: '0.03' },
		],
	},
	bolognese: {
		dish: 'bolognese',
		baseServings: 2,
		ingredients: [
			{ name: 'ground beef', amount: '300g', priceUsdc: '0.42' },
			{ name: 'tomatoes', amount: '400g', priceUsdc: '0.18' },
			{ name: 'onion', amount: '1', priceUsdc: '0.40' },
			{ name: 'carrot', amount: '1', priceUsdc: '0.35' },
			{ name: 'pasta', amount: '250g', priceUsdc: '0.16' },
		],
	},
	'aglio e olio': {
		dish: 'aglio e olio',
		baseServings: 2,
		ingredients: [
			{ name: 'spaghetti', amount: '200g', priceUsdc: '1.50' },
			{ name: 'garlic', amount: '4 cloves', priceUsdc: '0.025' },
			{ name: 'olive oil', amount: '60ml', priceUsdc: '0.11' },
			{ name: 'chili flakes', amount: '5g', priceUsdc: '0.02' },
		],
	},
};

function scalePrice(price: string, factor: number): string {
	const n = Number.parseFloat(price);
	if (Number.isNaN(n)) {
		return '0.00';
	}
	return (n * factor).toFixed(2);
}

@Injectable()
export class RecipesService {
	getRecipe(dish: string, servings: number): ScaledRecipe {
		const key = dish.trim().toLowerCase();
		const recipe = CATALOG[key];
		if (!recipe) {
			throw new NotFoundException(`unknown dish: ${dish}`);
		}
		const factor = servings / recipe.baseServings;
		const ingredients = recipe.ingredients.map((i) => ({
			...i,
			priceUsdc: scalePrice(i.priceUsdc, factor),
		}));
		const total = ingredients
			.reduce((acc, i) => acc + Number.parseFloat(i.priceUsdc), 0)
			.toFixed(2);
		return {
			dish: recipe.dish,
			servings,
			ingredients,
			totalUsdc: total,
		};
	}
}
