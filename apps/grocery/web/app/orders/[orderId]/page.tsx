"use client";

import Link from "next/link";
import { useParams, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";

const API =
  process.env.NEXT_PUBLIC_GROCERY_API_URL ?? "http://127.0.0.1:4402";

type ConfirmedOrder = {
  orderId: string;
  status: string;
  dish: string;
  servings: number;
  items: { name: string; amount: string; priceUsdc: string }[];
  totalUsdc: string;
  eta: string;
};

function txExplorerUrl(chain: string, txHash: string): string {
  const tx = txHash.trim();
  if (!tx) return "";
  switch (chain) {
    case "base":
      return `https://basescan.org/tx/${tx}`;
    case "base-sepolia":
    default:
      return `https://sepolia.basescan.org/tx/${tx}`;
  }
}

function orderNotFoundMessage(orderId: string, hasTx: boolean): string {
  if (hasTx) {
    return `Order ${orderId} is not in the shop database. Your payment may appear on BaseScan, but the grocery API never confirmed this order—often because payment verification failed or the server restarted after checkout. Place the order again through your agent to get a new confirmation link.`;
  }
  return `Order ${orderId} does not exist. Check the link or place a new order from the recipes page.`;
}

function loadFailureMessage(body: string, status: number): string {
  try {
    const parsed = JSON.parse(body) as { message?: string };
    if (typeof parsed.message === "string" && parsed.message.length > 0) {
      return parsed.message;
    }
  } catch {
    /* plain text */
  }
  if (body.trim()) return body.trim();
  return status > 0 ? `Request failed (${status})` : "Could not load order";
}

export default function OrderConfirmationPage() {
  const params = useParams<{ orderId: string }>();
  const searchParams = useSearchParams();
  const orderId = params.orderId;
  const txHash = searchParams.get("tx") ?? "";
  const chain = searchParams.get("chain") ?? "base-sepolia";

  const [order, setOrder] = useState<ConfirmedOrder | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setNotFound(false);
    try {
      const res = await fetch(`${API}/orders/${encodeURIComponent(orderId)}`);
      if (res.status === 404) {
        setOrder(null);
        setNotFound(true);
        return;
      }
      if (!res.ok) {
        const body = await res.text();
        setOrder(null);
        setError(loadFailureMessage(body, res.status));
        return;
      }
      setOrder(await res.json());
    } catch {
      setOrder(null);
      setError("Could not reach the grocery API. Is it running on port 4402?");
    } finally {
      setLoading(false);
    }
  }, [orderId]);

  useEffect(() => {
    void load();
  }, [load]);

  const explorer = txExplorerUrl(chain, txHash);

  let pageTitle = "Order details";
  if (order) pageTitle = "Order confirmed";
  else if (!loading) pageTitle = "Order not found";

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <main className="mx-auto max-w-2xl px-6 py-14">
        <p className="text-xs font-medium uppercase tracking-[0.2em] text-emerald-400/90">
          Grocery402
        </p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">
          {pageTitle}
        </h1>
        <p className="mt-2 font-mono text-sm text-emerald-300/90">{orderId}</p>

        {loading && (
          <p className="mt-8 text-sm text-zinc-400">Loading order…</p>
        )}
        {notFound && (
          <p className="mt-8 text-sm leading-relaxed text-zinc-300">
            {orderNotFoundMessage(orderId, Boolean(txHash))}
          </p>
        )}
        {error && (
          <p className="mt-8 text-sm text-rose-400">{error}</p>
        )}
        {order && (
          <section className="mt-8 rounded-2xl border border-emerald-900/50 bg-emerald-950/20 p-6">
            <h2 className="text-lg font-medium capitalize text-white">
              {order.dish}{" "}
              <span className="text-zinc-500">· {order.servings} servings</span>
            </h2>
            <p className="mt-2 text-sm text-emerald-200/80">{order.eta}</p>
            <ul className="mt-4 divide-y divide-zinc-800">
              {order.items.map((i) => (
                <li
                  key={i.name}
                  className="flex justify-between gap-4 py-2 text-sm"
                >
                  <span>
                    <span className="font-medium text-zinc-200">{i.name}</span>
                    <span className="text-zinc-500"> — {i.amount}</span>
                  </span>
                  <span className="tabular-nums text-emerald-300/90">
                    {i.priceUsdc} USDC
                  </span>
                </li>
              ))}
            </ul>
            <p className="mt-4 text-right text-base font-semibold text-white">
              Total{" "}
              <span className="tabular-nums text-emerald-400">
                {order.totalUsdc} USDC
              </span>
            </p>
          </section>
        )}

        <section className="mt-8 flex flex-col gap-3 text-sm">
          {explorer && (
            <a
              href={explorer}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 text-emerald-400 underline-offset-2 hover:underline"
            >
              View transaction on BaseScan ({chain})
            </a>
          )}
          <Link
            href="/"
            className="text-zinc-400 underline-offset-2 hover:text-zinc-200 hover:underline"
          >
            ← Back to recipes
          </Link>
        </section>
      </main>
    </div>
  );
}

