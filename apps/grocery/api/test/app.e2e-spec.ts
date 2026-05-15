import { Test, TestingModule } from '@nestjs/testing';
import { INestApplication, ValidationPipe } from '@nestjs/common';
import * as request from 'supertest';
import { AppModule } from './../src/app.module';

describe('Grocery402 API (e2e)', () => {
	let app: INestApplication;

	beforeEach(async () => {
		process.env.MERCHANT_ADDR =
			'0x1234567890123456789012345678901234567890';
		process.env.GROCERY_SKIP_X402_VERIFY = '1';
		const moduleFixture: TestingModule = await Test.createTestingModule({
			imports: [AppModule],
		}).compile();

		app = moduleFixture.createNestApplication();
		app.useGlobalPipes(
			new ValidationPipe({
				whitelist: true,
				transform: true,
				forbidNonWhitelisted: true,
			}),
		);
		await app.init();
	});

	afterEach(async () => {
		await app.close();
		delete process.env.GROCERY_SKIP_X402_VERIFY;
	});

	it('GET /recipes returns carbonara', () => {
		return request(app.getHttpServer())
			.get('/recipes?dish=carbonara&servings=2')
			.expect(200)
			.expect((res) => {
				expect(res.body.dish).toBe('carbonara');
				expect(res.body.totalUsdc).toBe('8.50');
			});
	});

	it('POST /orders without X-Payment returns 402', () => {
		return request(app.getHttpServer())
			.post('/orders')
			.send({ dish: 'carbonara', servings: 2 })
			.expect(402)
			.expect((res) => {
				expect(res.body.paymentRequirements).toBeDefined();
				expect(res.body.paymentRequirements.maxAmountRequired).toBe(
					'8.50',
				);
			});
	});

	it('POST /orders with X-Payment confirms when verify skipped', async () => {
		const res = await request(app.getHttpServer())
			.post('/orders')
			.set('X-Payment', Buffer.from('{}').toString('base64'))
			.send({ dish: 'carbonara', servings: 2 })
			.expect(201);
		expect(res.body.status).toBe('confirmed');
		expect(res.body.orderId).toMatch(/^GRC-/);

		await request(app.getHttpServer())
			.get(`/orders/${res.body.orderId}`)
			.expect(200)
			.expect((getRes) => {
				expect(getRes.body.orderId).toBe(res.body.orderId);
				expect(getRes.body.dish).toBe('carbonara');
			});
	});
});
