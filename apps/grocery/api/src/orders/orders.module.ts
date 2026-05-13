import { Module } from '@nestjs/common';
import { OrdersController } from './orders.controller';
import { OrdersService } from './orders.service';
import { RecipesModule } from '../recipes/recipes.module';
import { X402Module } from '../x402/x402.module';

@Module({
	imports: [RecipesModule, X402Module],
	controllers: [OrdersController],
	providers: [OrdersService],
})
export class OrdersModule {}
