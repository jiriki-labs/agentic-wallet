import { Injectable, Logger } from '@nestjs/common';
import {
	HTTPFacilitatorClient,
	decodePaymentSignatureHeader,
} from '@x402/core/http';
import {
	VerifyError,
	type Network,
	type PaymentPayload,
	type PaymentRequirements as CorePaymentRequirements,
} from '@x402/core/types';

/** USDC on Base Sepolia (policy / daemon examples). */
export const BASE_SEPOLIA_USDC = '0x036CbD53842c5426634e7929541eC2318f3dCF7e';

/**
 * 402 `accepts` row shape aligned with Jiriki Go `x402.PaymentRequirements` (v1 wire format).
 * Kept as a local interface so Nest `declaration` emit stays portable (no SDK path in `.d.ts`).
 */
export interface PaymentRequirements {
	scheme: string;
	network: string;
	maxAmountRequired: string;
	resource: string;
	description: string;
	mimeType: string;
	payTo: string;
	maxTimeoutSeconds: number;
	asset: string;
	outputSchema?: Record<string, unknown>;
	extra?: Record<string, unknown>;
}

@Injectable()
export class X402Service {
	private readonly log = new Logger(X402Service.name);

	private facilitatorClient(): HTTPFacilitatorClient {
		const url = (
			process.env.X402_FACILITATOR_URL ?? 'https://x402.org/facilitator'
		).replace(/\/$/, '');
		return new HTTPFacilitatorClient({ url });
	}

	buildRequirements(params: {
		publicBaseUrl: string;
		merchantPayTo: string;
		maxAmountRequired: string;
		description: string;
	}): PaymentRequirements {
		const resource = `${params.publicBaseUrl.replace(/\/$/, '')}/orders`;
		return {
			scheme: 'exact',
			network: 'base-sepolia',
			maxAmountRequired: params.maxAmountRequired,
			resource,
			description: params.description,
			mimeType: 'application/json',
			outputSchema: {},
			payTo: params.merchantPayTo,
			maxTimeoutSeconds: 600,
			asset: BASE_SEPOLIA_USDC,
			extra: {},
		};
	}

	/**
	 * Verifies X-Payment (base64 JSON payment payload from Jiriki / x402 client)
	 * against payment requirements via the public facilitator, using the official
	 * {@link HTTPFacilitatorClient} from `@x402/core/http`.
	 * Set GROCERY_SKIP_X402_VERIFY=1 for local demos without on-chain settlement.
	 */
	async verifyPaymentHeader(
		xPaymentB64: string,
		requirements: PaymentRequirements,
	): Promise<{ ok: boolean; reason?: string }> {
		if (process.env.GROCERY_SKIP_X402_VERIFY === '1') {
			return { ok: xPaymentB64.trim().length > 0 };
		}
		let paymentPayload: PaymentPayload;
		try {
			paymentPayload = decodePaymentSignatureHeader(
				xPaymentB64.trim(),
			) as PaymentPayload;
		} catch (e) {
			this.log.warn(`invalid X-Payment base64/json: ${e}`);
			return { ok: false, reason: 'invalid_x_payment_encoding' };
		}
		const client = this.facilitatorClient();
		const coreRequirements = {
			...requirements,
			outputSchema: requirements.outputSchema ?? {},
			extra: requirements.extra ?? {},
			network: requirements.network as Network,
		} as unknown as CorePaymentRequirements;
		try {
			const data = await client.verify(paymentPayload, coreRequirements);
			if (data.isValid === true) {
				return { ok: true };
			}
			const reason =
				data.invalidMessage ??
				data.invalidReason ??
				'facilitator_rejected';
			this.log.warn(`facilitator verify failed: ${reason}`);
			return { ok: false, reason };
		} catch (e) {
			if (e instanceof VerifyError) {
				const reason =
					e.invalidMessage ??
					e.invalidReason ??
					'facilitator_rejected';
				this.log.warn(`facilitator verify failed: ${reason}`);
				return { ok: false, reason };
			}
			this.log.error(`facilitator verify request error: ${e}`);
			return { ok: false, reason: 'facilitator_unreachable' };
		}
	}
}
