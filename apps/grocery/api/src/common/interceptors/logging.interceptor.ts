import {
	CallHandler,
	ExecutionContext,
	Injectable,
	Logger,
	NestInterceptor,
} from '@nestjs/common';
import { Reflector } from '@nestjs/core';
import { Request, Response } from 'express';
import { Observable } from 'rxjs';
import { tap } from 'rxjs/operators';
import { SKIP_LOGGING_KEY } from '../decorators/skip-logging.decorator';

function isDebugLogging(): boolean {
	return process.env.DEBUG === 'true' || process.env.DEBUG === '1';
}

@Injectable()
export class LoggingInterceptor implements NestInterceptor {
	private readonly logger = new Logger('HTTP');
	private readonly isDevelopment = isDebugLogging();

	constructor(private readonly reflector: Reflector) {}

	intercept(context: ExecutionContext, next: CallHandler): Observable<unknown> {
		const skipLogging = this.reflector.get<boolean>(
			SKIP_LOGGING_KEY,
			context.getHandler(),
		);
		if (skipLogging) {
			return next.handle();
		}

		const request = context.switchToHttp().getRequest<Request>();
		const response = context.switchToHttp().getResponse<Response>();
		const { method, url, query, body, params, ip } = request;
		const startTime = Date.now();

		if (url.includes('/health')) {
			return next.handle();
		}

		const logData: Record<string, unknown> = { method, url };

		if (Object.keys(query).length > 0) {
			logData.query = query;
		}
		if (Object.keys(params).length > 0) {
			logData.params = params;
		}
		if (['POST', 'PUT', 'PATCH'].includes(method) && body) {
			const bodyStr = JSON.stringify(body);
			logData.body =
				bodyStr.length > 1000
					? bodyStr.substring(0, 1000) + '... (truncated)'
					: body;
		}

		return next.handle().pipe(
			tap({
				next: () => {
					const duration = Date.now() - startTime;
					const statusCode = response.statusCode;
					const contentLength = response.get('content-length');
					const userAgent = request.get('User-Agent') || '';

					if (this.isDevelopment) {
						logData.duration = `${duration}ms`;
						logData.statusCode = statusCode;
						const line = `${method} ${url} ${statusCode} - ${duration}ms`;
						const payload = JSON.stringify(logData, null, 2);
						if (statusCode >= 500) {
							this.logger.error(line, payload);
						} else if (statusCode >= 400) {
							this.logger.warn(line, payload);
						} else {
							this.logger.debug(line, payload);
						}
					} else {
						this.logger.log(
							`${method} ${url} ${statusCode} ${contentLength || 0}b - ${duration}ms - ${ip || '-'} ${userAgent}`,
						);
					}
				},
				error: (error: { message?: string; status?: number }) => {
					const duration = Date.now() - startTime;
					const status = error.status || 500;
					if (this.isDevelopment) {
						logData.duration = `${duration}ms`;
						logData.error = error.message;
						logData.statusCode = status;
						this.logger.error(
							`${method} ${url} ${status} - ${duration}ms - ERROR: ${error.message}`,
							JSON.stringify(logData, null, 2),
						);
					} else {
						this.logger.error(
							`${method} ${url} ${status} - ${duration}ms - ERROR: ${error.message}`,
						);
					}
				},
			}),
		);
	}
}
