"use client";

import { useCallback, useState } from "react";

const API =
  process.env.NEXT_PUBLIC_GROCERY_API_URL ?? "http://127.0.0.1:4402";

type Recipe = {
  dish: string;
  servings: number;
  ingredients: { name: string; amount: string; priceUsdc: string }[];
  totalUsdc: string;
};

const DISHES = [
  { key: "carbonara", label: "Carbonara" },
  { key: "bolognese", label: "Bolognese" },
  { key: "aglio e olio", label: "Aglio e olio" },
];

export default function Home() {
  const [dish, setDish] = useState("carbonara");
  const [servings, setServings] = useState(2);
  const [recipe, setRecipe] = useState<Recipe | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [orderStatus, setOrderStatus] = useState<string | null>(null);
  const [orderBody, setOrderBody] = useState<string | null>(null);
  const [loadingRecipe, setLoadingRecipe] = useState(false);
  const [loadingOrder, setLoadingOrder] = useState(false);

  const fetchRecipe = useCallback(async () => {
    setLoadingRecipe(true);
    setLoadError(null);
    setOrderStatus(null);
    setOrderBody(null);
    try {
      const q = new URLSearchParams({
        dish,
        servings: String(servings),
      });
      const res = await fetch(`${API}/recipes?${q}`);
      if (!res.ok) {
        const t = await res.text();
        throw new Error(t || res.statusText);
      }
      setRecipe(await res.json());
    } catch (e) {
      setRecipe(null);
      setLoadError(e instanceof Error ? e.message : "Failed to load recipe");
    } finally {
      setLoadingRecipe(false);
    }
  }, [dish, servings]);

  const tryOrder = useCallback(async () => {
    setLoadingOrder(true);
    setOrderStatus(null);
    setOrderBody(null);
    try {
      const res = await fetch(`${API}/orders`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ dish, servings }),
      });
      const text = await res.text();
      setOrderBody(text);
      setOrderStatus(`${res.status} ${res.statusText}`);
    } catch (e) {
      setOrderStatus(e instanceof Error ? e.message : "Request failed");
    } finally {
      setLoadingOrder(false);
    }
  }, [dish, servings]);

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <main className="mx-auto max-w-2xl px-6 py-14">
        <p className="text-xs font-medium uppercase tracking-[0.2em] text-emerald-400/90">
          Grocery402
        </p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">
          Recipe & ingredients (x402 demo)
        </h1>
        <p className="mt-3 text-sm leading-relaxed text-zinc-400">
          Free recipe lookup from the Nest API. Ordering is{" "}
          <code className="rounded bg-zinc-900 px-1 py-0.5 text-emerald-300/90">
            POST /orders
          </code>{" "}
          and returns{" "}
          <span className="text-amber-200/90">402 Payment Required</span> until
          the Jiriki wallet replays with{" "}
          <code className="rounded bg-zinc-900 px-1 py-0.5">X-Payment</code>.
        </p>

        <section className="mt-10 space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/40 p-6">
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="text-zinc-400">Dish</span>
              <select
                className="mt-1 w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm outline-none ring-emerald-500/0 transition focus:ring-2"
                value={dish}
                onChange={(e) => setDish(e.target.value)}
              >
                {DISHES.map((d) => (
                  <option key={d.key} value={d.key}>
                    {d.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm">
              <span className="text-zinc-400">Servings</span>
              <input
                type="number"
                min={1}
                max={99}
                className="mt-1 w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-emerald-500/60"
                value={servings}
                onChange={(e) =>
                  setServings(
                    Math.min(99, Math.max(1, Number(e.target.value) || 1)),
                  )
                }
              />
            </label>
          </div>
          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              onClick={() => void fetchRecipe()}
              disabled={loadingRecipe}
              className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-emerald-500 disabled:opacity-50"
            >
              {loadingRecipe ? "Loading…" : "Load recipe"}
            </button>
            <button
              type="button"
              onClick={() => void tryOrder()}
              disabled={loadingOrder}
              className="rounded-lg border border-zinc-600 px-4 py-2 text-sm font-medium text-zinc-100 transition hover:border-zinc-500 hover:bg-zinc-800 disabled:opacity-50"
            >
              {loadingOrder ? "Posting…" : "POST /orders (unpaid probe)"}
            </button>
          </div>
          {loadError && (
            <p className="text-sm text-rose-400">
              API: {API} — {loadError}
            </p>
          )}
        </section>

        {recipe && (
          <section className="mt-8 rounded-2xl border border-zinc-800 bg-zinc-900/30 p-6">
            <h2 className="text-lg font-medium capitalize text-white">
              {recipe.dish}{" "}
              <span className="text-zinc-500">
                · {recipe.servings} servings
              </span>
            </h2>
            <ul className="mt-4 divide-y divide-zinc-800">
              {recipe.ingredients.map((i) => (
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
                {recipe.totalUsdc} USDC
              </span>
            </p>
          </section>
        )}

        {(orderStatus || orderBody) && (
          <section className="mt-8 rounded-2xl border border-amber-900/40 bg-amber-950/20 p-6">
            <h3 className="text-sm font-medium text-amber-200/90">
              Unpaid order probe (
              {DISHES.find((d) => d.key === dish)?.label ?? dish})
            </h3>
            {orderStatus && (
              <p className="mt-2 font-mono text-sm text-amber-100/90">
                {orderStatus}
              </p>
            )}
            {orderBody && (
              <pre className="mt-3 max-h-64 overflow-auto rounded-lg bg-black/40 p-3 text-xs leading-relaxed text-zinc-300">
                {orderBody}
              </pre>
            )}
          </section>
        )}
      </main>
    </div>
  );
}
