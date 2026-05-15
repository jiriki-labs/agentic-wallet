import { loadGroceryEnv } from './load-env';

loadGroceryEnv();

import { Logger, ValidationPipe, type LogLevel } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { LoggingInterceptor } from './common/interceptors/logging.interceptor';

async function bootstrap() {
	const debugEnabled =
		process.env.DEBUG === 'true' || process.env.DEBUG === '1';
	const loggerLevels: LogLevel[] = debugEnabled
		? ['log', 'error', 'warn', 'debug', 'verbose']
		: ['log', 'error', 'warn'];

	const app = await NestFactory.create(AppModule, { logger: loggerLevels });
	app.enableCors({
		origin: [
			'http://localhost:3000',
			'http://127.0.0.1:3000',
			'http://localhost:3020',
			'http://127.0.0.1:3020',
		],
		exposedHeaders: ['X-Payment-Requirements'],
	});
	app.useGlobalPipes(
		new ValidationPipe({
			whitelist: true,
			transform: true,
			forbidNonWhitelisted: true,
		}),
	);
	app.useGlobalInterceptors(app.get(LoggingInterceptor));

	const port = parseInt(process.env.PORT ?? '4402', 10);
	await app.listen(port, '0.0.0.0');
	const logger = new Logger('Bootstrap');
	logger.log(`Grocery402 API listening on http://0.0.0.0:${port}`);
}
bootstrap();
