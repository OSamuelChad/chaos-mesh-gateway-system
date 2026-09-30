# Chaos Mesh Gateway & Resilience Visualizer

Malha de microsserviços em **Go** com um **API Gateway** (JWT, Swagger e HATEOAS dinâmico), um serviço de pagamentos com **caos programável** e um serviço de resiliência com **Circuit Breaker**. Um cliente **React + React Flow** mostra a malha em tempo real e monta os botões de ação só a partir do `_links` enviado pelo gateway.

```
            :3000                     :8080 (host)
┌──────────────────┐  /api   ┌────────────────────┐        ┌──────────────────────┐        ┌───────────────────┐
│   web-client     │───────▶ │    api-gateway     │──────▶ │  service-resilience  │──────▶ │  service-payment  │
│ React · Nginx    │         │ JWT · HATEOAS ·    │        │  sony/gobreaker      │        │  chaos middleware │
│ React Flow       │         │ Swagger · proxy    │───┐    │  :8082 (interno)     │        │  :8081 (interno)  │
└──────────────────┘         └────────────────────┘   └───────────────────────────────────▶ (status/caos)
                                     rede Docker privada: mesh-network
```

## Como executar

Pré-requisito: Docker com Docker Compose.

```bash
docker-compose up --build        # ou: docker compose up --build
```

| URL | O que é |
|---|---|
| http://localhost:3000 | Visualizador (login `admin` / `password`) |
| http://localhost:8080/swagger/index.html | Documentação Swagger do gateway |
| http://localhost:8080/api/v1/auth/login | Endpoint de login (JWT) |

Os serviços `service-payment` e `service-resilience` **não** são expostos ao host: só respondem dentro da `mesh-network`.

### Variáveis de ambiente (opcionais)

| Variável | Padrão | Uso |
|---|---|---|
| `JWT_SECRET` | aleatório a cada início | Segredo HS256 (mín. 32 bytes). Sem valor, os tokens deixam de valer quando o gateway reinicia. |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | `admin` / `password` | Credenciais do login (as padrão são só para demonstração). |

```bash
JWT_SECRET=$(openssl rand -hex 32) ADMIN_PASSWORD=troque-me docker-compose up --build
```

## Roteiro de demonstração

1. Entre com `admin` / `password`. Os três nós aparecem **verdes** (`HEALTHY`).
2. Clique em **Payment Service**. O painel mostra as ações recebidas em `_links`.
3. **Injetar 2s de Latência**: o nó fica **amarelo** (`DEGRADED`).
4. **Injetar 80% de Erros** e depois **Simular Tráfego**: o breaker passa de `CLOSED` para `OPEN`, Resilience e Payment ficam **vermelhos** (`CRITICAL`) e as chamadas passam a receber o fallback `503`.
5. Clique em **Resilience Service**. Com o circuito aberto, o gateway passa a oferecer **Fechar Circuit Breaker** e **Restaurar Payment (Heal)**; a ação "Processar 1 cobrança" deixa de ser oferecida.
6. **Restaurar / Heal**. Depois de 10 s o breaker vai para `HALF-OPEN` (amarelo); com tráfego saudável, volta a `CLOSED`.

## API do Gateway

Todas as rotas, exceto o login, exigem `Authorization: Bearer <token>`. **Toda resposta, inclusive as de erro, traz `_links`.**

| Método | Rota | Descrição |
|---|---|---|
| POST | `/api/v1/auth/login` | Emite o JWT (limite de 10 tentativas/min por IP) |
| GET | `/api/v1` | Índice navegável |
| GET | `/api/v1/mesh/topology` | Agregador: nós, arestas e saúde da malha |
| GET | `/api/v1/services/{gateway,payment,resilience}` | Detalhe de um nó e suas ações |
| PUT | `/api/v1/services/payment/chaos/latency` | `{"latency_ms": 0-10000}` |
| PUT | `/api/v1/services/payment/chaos/faults` | `{"rate": 0-100}` |
| POST | `/api/v1/services/payment/chaos/reset` | Restaura a operação normal |
| POST | `/api/v1/services/resilience/breaker/reset` | Força o breaker para `CLOSED` |
| POST | `/api/v1/payments` | Cobrança pelo fluxo Gateway → Resilience → Payment |
| POST | `/api/v1/traffic/simulate` | `{"requests": 1-100, "concurrency": 1-10}` |

### Motor HATEOAS

Cada link tem este formato:

```json
"inject-faults": {
  "href": "/api/v1/services/payment/chaos/faults",
  "method": "PUT",
  "title": "Injetar 80% de Erros",
  "body": { "rate": 80 }
}
```

As regras ficam em `api-gateway/hateoas/rules.go` e dependem da saúde dos serviços e do estado do breaker. Por exemplo:
- `inject-latency` e `inject-faults` só aparecem se aquele caos ainda não estiver ativo;
- `heal` só aparece com caos ativo;
- `process-payment` some quando o breaker está `OPEN`;
- `reset-breaker` aparece fora do estado `CLOSED`;
- serviço inacessível → só `self` e `topology`.

O React (`web-client/src/services/hypermedia.js`) apenas percorre o mapa `_links` e executa `method` + `href` + `body`. Nenhum botão é fixo no código.

### Saúde dos nós (cores)

| Nó | 🟢 HEALTHY | 🟡 DEGRADED | 🔴 CRITICAL |
|---|---|---|---|
| Payment | sem caos | latência ≥ 1 s ou erros entre 1% e 49% | erros ≥ 50%, breaker `OPEN` ou inacessível |
| Resilience | `CLOSED` | `HALF-OPEN` | `OPEN` ou inacessível |
| Gateway | tudo ok | algum serviço crítico | ambos inacessíveis |

## Serviços internos

**service-payment** (`:8081`)
- `POST /process`: cobrança fictícia, protegida pelo middleware de caos.
- `POST /chaos/latency`, `POST /chaos/faults`, `POST /chaos/reset`, `GET /chaos/status`.
- O estado do caos usa `sync/atomic` (`atomic.Int64` / `atomic.Uint64`): sem mutex e sem race conditions. O middleware aplica `time.Sleep` e/ou retorna `HTTP 500`.

**service-resilience** (`:8082`)
- `POST /resilience/process`: envia a cobrança ao payment através do `sony/gobreaker`.
- `GET /resilience/status`: estado (`CLOSED`/`OPEN`/`HALF-OPEN`), janela de amostragem, métricas e configuração.
- `POST /resilience/reset`.

Configuração do breaker:

| Parâmetro | Valor |
|---|---|
| Condição de abertura | janela de 30 s com ≥ 5 requisições e taxa de falhas **> 50%** |
| Tempo em `OPEN` | 10 s |
| Requisições de teste em `HALF-OPEN` | 3 |
| Timeout por chamada | 3 s |

Respostas 5xx, timeouts e erros de rede contam como falha; 4xx (validação) não contam. Com o circuito `OPEN`, o serviço responde na hora com **503** e um payload amigável (`fallback: true`, `retry_after_seconds`), sem chamar o payment.

## Estrutura

```
chaos-mesh-gateway/
├── docker-compose.yml
├── gorun.sh                    # roda comandos Go num container (Go não é necessário no host)
├── api-gateway/
│   ├── main.go                 # configuração, rotas, CORS, limiter, Swagger
│   ├── handlers/               # auth, topologia/nós, caos, pagamentos, tráfego, cliente upstream
│   ├── middleware/jwt.go       # emissão/validação JWT (HS256, exp, iss, bloqueio de alg "none")
│   ├── hateoas/                # Link/Links, regras dinâmicas, saúde, resposta de erro
│   ├── docs/                   # gerado por `swag init`
│   └── Dockerfile
├── service-payment/
│   ├── main.go · process.go
│   ├── chaos/                  # controller atômico, middleware e rotas de caos
│   └── Dockerfile
├── service-resilience/
│   ├── main.go
│   ├── breaker/                # engine com gobreaker, fallback e métricas
│   └── Dockerfile
└── web-client/
    ├── src/
    │   ├── components/         # LoginScreen, MeshGraph, ServiceNode, ActionPanel...
    │   ├── services/           # api (axios + JWT), session (estado global), hypermedia
    │   ├── hooks/useTopology.js
    │   └── App.jsx
    ├── nginx.conf              # serve o build e faz proxy de /api e /swagger
    ├── package.json · vite.config.js
    └── Dockerfile
```

## Desenvolvimento

Não é preciso ter Go instalado: o `gorun.sh` usa a imagem `golang:1.22-alpine`.

```bash
./gorun.sh service-payment    "go test ./..."
./gorun.sh service-resilience "go test ./..."
./gorun.sh api-gateway        "go test ./..."

# Regenerar o Swagger depois de alterar anotações
./gorun.sh api-gateway "go install github.com/swaggo/swag/cmd/swag@v1.16.4 && swag init -g main.go --output docs"

# Frontend com hot reload (com o gateway rodando em :8080)
cd web-client && npm install && npm run dev   # http://localhost:5173
npm test
```

## Stack

Go 1.22 · Fiber v2 · golang-jwt v5 · swaggo/swag + gofiber/swagger · sony/gobreaker · React 19 · Vite · Tailwind CSS 4 · React Flow (`@xyflow/react`) · Axios · Docker Compose.
