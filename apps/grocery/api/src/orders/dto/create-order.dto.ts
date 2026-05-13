import { Type } from 'class-transformer';
import { IsInt, IsString, Max, Min, MinLength } from 'class-validator';

export class CreateOrderDto {
	@IsString()
	@MinLength(2)
	dish!: string;

	@Type(() => Number)
	@IsInt()
	@Min(1)
	@Max(99)
	servings!: number;
}
