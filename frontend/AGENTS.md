# Console HadesGate (`frontend/`): guia para agentes

Painel administrativo e Sandbox em tempo real construído em **Next.js 16** (App Router, React 19, TypeScript 5, Tailwind CSS) com exportação estática (`output: 'export'` em `next.config.js`), embutido no binário Go em produção.

## Identidade Visual: Industrial Brutalism (HadesGate Edition)

Consulte também `.agents/skills/industrial-brutalist-ui/SKILL.md`, respeitando sempre os tokens oficiais do HadesGate:

- **Geometria rígida (`0px` radius):** `border-radius` é proibido em toda a interface. Cantos de `90°`, bordas sólidas visíveis (`1px` e `2px`) e divisórias estruturais.
- **Tipografia bimodal:**
  - Macro/Display (`font-display` / `Archivo Black`): caixa alta, tracking comprimido, peso pesado para cabeçalhos e KPIs (`<data>`).
  - Micro/Telemetria (`font-mono` / `JetBrains Mono`): caixa alta para labels técnicos, tags `[ ZERO_PII_LEAK ]`, timestamps, hashes e métricas.
- **Tema e Paleta:**
  - **Dark Mode é o padrão** (`#121216` / `#181820`), com suporte completo ao seletor **Light Mode** (`#F4F4F0`). Nunca quebre o contraste em nenhum dos dois modos.
  - **Cor de destaque primária:** Roxo HadesGate (`#9333EA` / `purple-600` e `#A855F7` / `purple-400` no escuro; `purple-700` no claro). Vermelho (`#FF2A2A`) é reservado para bloqueios/alertas críticos e âmbar/verde apenas para indicadores semânticos pontuais.
- **Logo oficial:** use sempre o componente `<HadesGateLogo />` (`frontend/components/hadesgate-logo.tsx`) com `currentColor` (`text-zinc-950 dark:text-white`) — nunca recrie quadrados com letras `"HG"`.

## Internacionalização (`pt-BR` padrão + `en`)

- `pt-BR` é o idioma padrão de inicialização (`frontend/app/page.tsx`), com alternância imediata para `en` persistida em `localStorage` (`hadesgate_lang`).
- Todo texto novo visível ao usuário deve ser adicionado à interface `Translations` em `frontend/lib/i18n.ts` com tradução completa nos blocos `"pt-BR"` e `"en"`.

## Contratos com o Backend Go

- Em `next dev` (`:3000`), `app/page.tsx` aponta automaticamente para `http://localhost:8080`. Em produção (servido pelo binário Go), usa `window.location.origin`.
- A aba principal de Telemetria (`components/observability.tsx`) consome `/api/traces?page=<n>&limit=<n>` com paginação no SQLite — nunca remova a paginação nem busque listas ilimitadas sem controle de página.
- A tela de login (`components/login.tsx`) nunca deve exibir nomes de variáveis de ambiente de segurança do servidor (como `HADESGATE_ADMIN_KEY`).

## Validação obrigatória

Ao tocar em qualquer arquivo de `frontend/`:

```bash
cd frontend && npx tsc --noEmit
cd frontend && npm run build
```

Ambos devem terminar sem erros antes de considerar a tarefa concluída.

