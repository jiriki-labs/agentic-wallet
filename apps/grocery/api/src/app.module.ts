import { Module } from '@nestjs/common';
import { LoggingInterceptor } from './common/interceptors/logging.interceptor';
import { OrdersModule } from './orders/orders.module';
import { RecipesModule } from './recipes/recipes.module';

@Module({
	imports: [RecipesModule, OrdersModule],
	providers: [LoggingInterceptor],
})
export class AppModule {}
