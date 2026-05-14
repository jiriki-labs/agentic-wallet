---
name: jiriki-github-issues
description: >-
  Defines how to write GitHub issues (tasks) for jiriki-labs/agentic-wallet:
  Polish body structure matching issues #1 and #2, epic-level granularity
  (roughly 3–5 issues unless the user asks for a fine-grained breakdown), and
  required sections. Use when creating, splitting, or rewriting GitHub issues,
  task lists for the org project board, or “zrób issue” / “add tasks” requests
  for this repository.
---

# Jiriki — format issue na GitHubie

## Kiedy stosować

Przy tworzeniu lub edycji issue w **`jiriki-labs/agentic-wallet`** (oraz gdy użytkownik prosi o taski pod Projects / Issues).

## Granularność

- **Domyślnie:** kilka **epików** (około **3–5** otwartych issue), każdy z pełnym opisem i checklistą — w stylu [#1](https://github.com/jiriki-labs/agentic-wallet/issues/1) i [#2](https://github.com/jiriki-labs/agentic-wallet/issues/2), nie dziesiątki mikro-issue, chyba że użytkownik wyraźnie chce rozbicie.
- **Wyjątek:** rozbij na więcej issue, gdy użytkownik poprosi o listę pod sprinty albo gdy jeden epik blokuje wiele zespołów i musi być śledzony osobno.

## Język i tytuł

- **Tytuł i treść issue:** po **polsku** (spójnie z istniejącymi „Flow release”, „Git flow” i zagregowanymi #25–#28).
- **Nazwy pól JSON / API / kodu** w opisie: po **angielsku**, dosłownie jak w repo (np. `txHash`, `allowedMerchants`).

## Obowiązkowa struktura treści (`body`)

Kolejność sekcji — **zachowaj nagłówki** (poziom `##` jak poniżej):

```markdown
# [Krótki tytuł zadania — może powielić temat z pola title]

## Opis

[1–3 akapity: kontekst, cel, dlaczego to robimy. Dla Jiriki: daemon lokalny, YAML, socket/TCP, brak udostępniania klucza agentom — tylko jeśli istotne dla zadania.]

---

## Zakres zadania

[Wypunktowana lub numerowana lista konkretnych dostaw — co jest IN.]

---

## Acceptance criteria

- [ ] …
- [ ] …

---

## Poza zakresem

[Co świadomie nie robimy w tym issue — żeby nie rozjeżdżać zakresu.]
```

### Zasady jakości

1. **`## Opis`** — zawsze; wyjaśnij „done” na poziomie produktu/procesu.
2. **`## Zakres zadania`** — numerowana lub lista z podrzędnymi nagłówkami `###` jeśli epik jest duży (jak w #2).
3. **`## Acceptance criteria`** — **checkboxy** `- [ ]`; muszą być **weryfikowalne** (da się odhaczyć po review).
4. **`## Poza zakresem`** — zawsze; redukuje scope creep.
5. Separator `---` między głównymi blokami — jak w #1 / #2.
6. **Fragmenty techniczne** (komendy, ścieżki plików, przykładowy YAML) w fenced code blocks.
7. **Bezpieczeństwo:** nigdy nie wklejaj w issue prawdziwych kluczy, seedów, bearer tokenów, haseł keystore; używaj placeholderów.

## Label i metadane

- Jeśli repo ma label **`enhancement`** dla feature’ów — użyj przy taskach produktowych / procesowych.
- **Assignee** tylko jeśli użytkownik lub maintainer wskaże osobę.

## Odwołania w repo

- Przy zmianach w polityce / daemonie wskazuj ścieżki pakietów (`internal/`, `cmd/jiriki/`, `configs/`) zgodnie z `AGENTS.md` (minimalny diff, bez refaktoru `internal/` bez potrzeby).

## Szybki self-check przed utworzeniem issue

- [ ] Czy to jeden logiczny epik (albo użytkownik chciał świadomie drobnych issue)?
- [ ] Czy są wszystkie cztery sekcje: Opis, Zakres, Acceptance criteria, Poza zakresem?
- [ ] Czy każde acceptance criterion da się zweryfikować?
- [ ] Czy tytuł jest po polsku i jednoznaczny?
