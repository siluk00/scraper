# scraper
[![CI](https://github.com/siluk00/scraper/actions/workflows/ci.yml/badge.svg)](https://github.com/siluk00/scraper/actions/workflows/ci.yml)

# scraper

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

## Endpoints

### `GET /lenovo`
Retorna todos os notebooks Lenovo encontrados, ordenados do mais barato ao mais caro.

**Exemplo de resposta:**
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
internal/scraper/     → lógica de scraping e parsing HTML
internal/api/         → servidor HTTP e handlers REST
internal/client/      → cliente que consome a API (com TLS fingerprinting)
```

## Decisões técnicas

### Por que `net/http` + `golang.org/x/net/html` e não colly ou goquery?

Neste projeto eu quis demonstrar domínio da stdlib Go sem depender de frameworks.  O `golang.org/x/net.html`  é o parser HTML oficial do projeto go. A biblioteca colly seria mais conveniente pra esse projeto, mas esconderia toda a mecânica de atravessar a árvore DOM que implementei manualmente. 

### Por que `models.Product` em um pacote separado?

Por razões de escalabilidade, caso haja necessidade de expandir a api e algum outro pacote necessite utilizar o modelo, já temos um modelo que pode ser alterado sem necessidade de alterar o código em pontos duplicados.

### Por que TLS com `utls` e JA3 fingerprinting?

O handshake TLS do Go stdlib gera um JA3 fingerprint fixo e publicamente conhecido como "cliente Go". Sites protegidos por Cloudflare ou Akamai usam isso para bloquear bots. O `utls` permite controlar exatamente o `ClientHello` enviado, imitando o fingerprint do Chrome. Combinado com headers de browser reais (User-Agent, Accept-Language), o cliente se torna indistinguível de um browser para a maioria dos sistemas anti-bot. 

### Por que `sort.Slice` após coletar todas as páginas e não durante?

Ordenar durante o scraping não faria sentido porque os dados chegam por página. Inserção ordenada seria O(n²). Coletar tudo e ordenar uma vez é O(n log n) e mais legível.

### Por que `slog` e não `log` ou `zerolog`?

 `slog` é o logger estruturado oficial do Go desde 1.21. Logging estruturado (key-value) é essencial para observabilidade em produção — facilita filtros em ferramentas como Datadog, Loki ou CloudWatch. Escolher a stdlib evita dependência extra mantendo a qualidade de um logger estruturado.

## Limitações conhecidas

- O scraping é feito a cada request, sem cache. Em produção, isso seria mitigado com cache em memória (TTL de alguns minutos) ou scraping periódico em background.
- O site alvo é estático. Sites com JavaScript dinâmico precisariam de uma abordagem diferente.
- Apenas o endpoint `/lenovo` está implementado. A estrutura já está preparada para expandir para outras marcas sem mudança arquitetural.

## Dependências

| Pacote | Motivo |
|--------|--------|
| `golang.org/x/net/html` | Parser HTML oficial do projeto Go |
| `github.com/refraction-networking/utls` | TLS fingerprinting (JA3 imitando Chrome) |
