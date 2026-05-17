# scraper
[![CI](https://github.com/siluk00/scraper/actions/workflows/ci.yml/badge.svg)](https://github.com/siluk00/scraper/actions/workflows/ci.yml)

API REST em Go que coleta notebooks Lenovo do site [webscraper.io](https://webscraper.io/test-sites/e-commerce/static/computers/laptops), ordena por preço e expõe os dados via JSON.

## Como rodar

```bash
git clone https://github.com/siluk00/scraper.git
cd scraper
go mod download
go run ./cmd/server
```

A API estará disponível em `http://localhost:8080`.

Para mudar a porta:
```bash
PORT=9090 go run ./cmd/server
```

## Testes

```bash
go test ./internal/scraper/...
```

Os testes usam `httptest.NewServer` para simular o site alvo localmente — nenhuma requisição real é feita para a internet. Cobrem:

- Parsing básico de produto (título, preço, URL)
- Cache: segunda chamada retorna resultado em memória sem hit no servidor
- Expiração do cache: após o TTL, o scraper busca dados frescos
- Retry em 429: verifica que o backoff realmente retenta e tem sucesso
- Ordenação: produtos sempre retornam do mais barato ao mais caro
- `brandMatch`: filtro de marca incluindo sub-marcas Lenovo (ThinkPad, IdeaPad)

## Endpoints

### `GET /lenovo`

Retorna todos os notebooks Lenovo encontrados, ordenados do mais barato ao mais caro. O resultado é servido do cache em memória se a última coleta tiver menos de 5 minutos.

```json
[
  {
    "title": "Lenovo IdeaPad 320-15IAP",
    "price": 299.99,
    "description": "...",
    "rating": 4,
    "reviews": 3,
    "url": "https://webscraper.io/..."
  }
]
```

### `GET /healthz`

Verifica se o servidor está vivo e se consegue alcançar o site alvo.

**200 OK:**
```json
{"status":"ok","target_reachable":true,"latency_ms":142}
```

**503 Service Unavailable** (servidor vivo, site alvo fora):
```json
{"status":"degraded","target_reachable":false,"error":"...","latency_ms":5001}
```

## Estrutura do projeto

```
cmd/server/           → entrypoint, inicializa o servidor
internal/models/      → struct Product compartilhado entre pacotes
internal/scraper/     → lógica de scraping, cache e retry
internal/api/         → servidor HTTP e handlers REST
internal/client/      → http.Client com TLS fingerprinting e headers de browser
```

## Decisões técnicas

### Por que `net/http` + `golang.org/x/net/html` e não colly ou goquery?

Neste projeto eu quis demonstrar domínio da stdlib Go sem depender de frameworks. O `golang.org/x/net/html` é o parser HTML oficial do projeto Go. A biblioteca colly seria mais conveniente para esse projeto, mas esconderia toda a mecânica de atravessar a árvore DOM que implementei manualmente.

### Por que `models.Product` em um pacote separado?

Por razões de escalabilidade, caso haja necessidade de expandir a API e algum outro pacote necessite utilizar o modelo, já temos um modelo que pode ser alterado sem necessidade de alterar o código em pontos duplicados.

### Por que TLS com `utls` e JA3 fingerprinting?

O handshake TLS do Go stdlib gera um JA3 fingerprint fixo e publicamente conhecido como "cliente Go". Sites protegidos por Cloudflare ou Akamai usam isso para bloquear bots. O `utls` permite controlar exatamente o `ClientHello` enviado, imitando o fingerprint do Chrome. Combinado com headers de browser reais (User-Agent, Accept-Language), o cliente se torna indistinguível de um browser para a maioria dos sistemas anti-bot.

### Por que `sort.Slice` após coletar todas as páginas e não durante?

Ordenar durante o scraping não faria sentido porque os dados chegam por página. Inserção ordenada seria O(n²). Coletar tudo e ordenar uma vez é O(n log n) e mais legível.

### Por que `slog` e não `log` ou `zerolog`?

`slog` é o logger estruturado oficial do Go desde 1.21. Logging estruturado (key-value) é essencial para observabilidade em produção — facilita filtros em ferramentas como Datadog, Loki ou CloudWatch. Escolher a stdlib evita dependência extra mantendo a qualidade de um logger estruturado.

### Por que cache em memória com TTL de 5 minutos?

O scraping completo leva alguns segundos e envolve múltiplas requisições HTTP. Servir o mesmo resultado para todos os requests dentro de uma janela de tempo evita dois problemas: latência alta para o cliente e risco de ban por rate limit no site alvo. O TTL de 5 minutos é um trade-off entre frescor dos dados e custo de coleta — facilmente ajustável via constante `cacheTTL`.

O cache é protegido por `sync.RWMutex`: múltiplas goroutines podem ler simultaneamente, e a escrita (atualização após scraping) bloqueia exclusivamente. O resultado retornado é uma cópia via `copy()` — o caller não pode mutar o slice interno.

### Por que backoff exponencial com jitter no retry?

Quando uma requisição falha por rate limit (429) ou erro de servidor (5xx), retomar imediatamente provavelmente vai falhar de novo pelo mesmo motivo. O backoff exponencial (1s, 2s, 4s…) dá tempo para o servidor se recuperar.

O jitter (variação aleatória de ±30%) é essencial quando há concorrência: sem ele, todas as goroutines que falham ao mesmo tempo vão retomar ao mesmo tempo — o "thundering herd" — piorando o problema. O jitter espalha os retries no tempo. A estratégia foi descrita originalmente pelo time da AWS em ["Exponential Backoff And Jitter"](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/).

### Por que `ScrapeProducts` recebe `*http.Client` como parâmetro?

Injeção de dependência. O `scraper` não decide qual client usar — quem chama decide. Isso tem duas consequências práticas: em produção, passa o `BrowserClient` com TLS fingerprinting; em testes, passa um client apontando para um `httptest.Server` mockado sem tocar na internet.

## Limitações conhecidas

- **Cache por processo:** o cache vive na memória do processo. Em um deploy com múltiplas instâncias, cada instância tem seu próprio cache. 
- **Connection pool do utls:** o `roundTripper` atual abre um novo handshake TLS a cada request porque o `http.Transport` é recriado após cada conexão estabelecida.
- **Site estático:** o site alvo usa HTML estático. Sites com conteúdo renderizado via JavaScript precisariam de uma abordagem diferente.
- **Apenas Lenovo:** a arquitetura já suporta múltiplas marcas via `ScrapeProducts(client, url, brand)`, mas apenas o endpoint `/lenovo` está exposto.

## Dependências

| Pacote | Motivo |
|--------|--------|
| `golang.org/x/net/html` | Parser HTML oficial do projeto Go |
| `github.com/refraction-networking/utls` | TLS fingerprinting (JA3 imitando Chrome) |

## Referências técnicas

- [RFC 8446 §4.1.2](https://www.rfc-editor.org/rfc/rfc8446#section-4.1.2) — estrutura do ClientHello no TLS 1.3
- [JA3 TLS Fingerprinting](https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967) — artigo original da Salesforce Engineering
- [Exponential Backoff And Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/) — AWS Architecture Blog
- [refraction-networking/utls](https://github.com/refraction-networking/utls) — documentação dos perfis ClientHello disponíveis
- [pkg.go.dev/net/http](https://pkg.go.dev/net/http#Transport) — documentação do Transport e connection pool
- [pkg.go.dev/golang.org/x/net/html](https://pkg.go.dev/golang.org/x/net/html) — documentação do parser HTML
