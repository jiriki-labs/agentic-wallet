import { Module } from '@nestjs/common';
import { RecipesModule } from './recipes/recipes.module';
import { OrdersModule } from './orders/orders.module';

@Module({
	imports: [RecipesModule, OrdersModule],
})
export class AppModule {}
