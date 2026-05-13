import { NestFactory } from '@nestjs/core';
import { ValidationPipe } from '@nestjs/common';
import { AppModule } from './app.module';

async function bootstrap() {
	const app = await NestFactory.create(AppModule);
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
	const port = parseInt(process.env.PORT ?? '4402', 10);
	await app.listen(port, '0.0.0.0');
	// eslint-disable-next-line no-console
	console.log(`Grocery402 API listening on http://0.0.0.0:${port}`);
}
bootstrap();
