# Dataset DLP para Laya

Conjunto inicial sintético para treinar **presença contextual de categorias**, no
formato nativo `{state, questions, expected}` de `laya.train`. Cada categoria é
uma pergunta `noul` (sim/não), permitindo múltiplas categorias na mesma entrada.
As perguntas rotuladas variam por linha para reduzir negativos triviais. Não se
trata de um dataset de chat ou de geração de texto mascarado.

## Arquivos e reprodução

- `train.jsonl`: entrada para fine-tuning.
- `questions.json`: perguntas para inferência, com os mesmos critérios do treino.
- `spans.jsonl`: anotações auxiliares correspondentes às linhas do treino;
  `start` inclusivo e `end` exclusivo em **bytes UTF-8**, como no sidecar Go.
  Essas anotações não são alvos de extração aprendidos pelo Laya.
- `seeds.json`: valores e contextos sintéticos editáveis.
- `manifest.json`: contagens positivas/negativas por categoria e limitações.

Na raiz do repositório, com Node.js:

```sh
node datasets/laya-dlp/generate.mjs
node datasets/laya-dlp/generate.mjs --check
```

O gerador verifica cobertura de todas as categorias declaradas no código Go,
presença de positivos e negativos, duplicatas, sobreposições, limites e conteúdo
dos spans. Não lê `.env`, logs, banco de dados nem o corpus do gate de acurácia.

## Treinamento

O contrato foi conferido em [laya/train.py](https://github.com/NandhaKishorM/laya/blob/main/laya/train.py)
em 08/11/2026, em particular `items_from_rows` e `target_from_expected`.
Use uma versão do Laya que exponha essas APIs; a branch upstream é mutável.
Exemplo para executar em ambiente com Laya e recursos de treinamento instalados:

```python
from laya.train import TrainConfig, finetune

finetune(
    "datasets/laya-dlp/train.jsonl",
    "convaiinnovations/laya-multilingual",
    "./laya-hadesgate-dlp",
    TrainConfig(loss="soft-ce", seed=42),
)
```

Não foi executado treinamento ou download de pesos na criação deste conjunto.
Valide a API na versão instalada antes de iniciar uma execução longa.

## Escopo e limites

As 33 categorias declaradas estão representadas: CPF, CNPJ, PHONE, EMAIL,
CREDIT_CARD, PERSON, CEP, PIX, RG, CNH, BANK_ACCOUNT, IBAN, PASSPORT, IP_ADDRESS,
API_KEY, SECRET, CUSTOM, SSN, EIN, NINO, NHS, DNI, NIR, CODICE_FISCALE, MEDICAL_ID,
SUS_CARD, VEHICLE, LEGAL, DEVICE_ID, GOVERNMENT_ID, CID, MEDICATION e CORPORATE.
Há variações de segredos (senha, JWT, URI, PEM, SSH), documentos, contas,
dispositivos e veículos, contextos mistos, menções sem valores, placeholders já
mascarados e instruções adversariais. A cobertura é de categorias e alguns
subtipos; não esgota padrões, provedores, medicamentos ou idiomas aceitos.

`CUSTOM` usa a regra fictícia `PRIV-ORION-\d{2}`. `CORPORATE` usa nomes de
projetos explicitamente confidenciais; é uma categoria declarada sem detector
nativo dedicado. Nenhuma delas implica conhecer regras privadas dos tenants.
Adapte critérios e exemplos às regras efetivamente configuradas antes de usar.

São valores inventados ou fixtures públicas de teste (ex.: cartão de sandbox e
IBAN ilustrativo), sem vínculo intencional com pessoas reais. Identificadores
inventados podem coincidir com registros existentes. A tarefa é reconhecer o
tipo no contexto: **não há garantia de checksum válido**, existência do documento
ou credencial funcional. Não use estes rótulos como testes de paridade dos
validadores determinísticos. Credenciais e material criptográfico são fixtures
sem uso operacional.

Este é um **bootstrap pequeno e baseado em templates**, com valores repetidos em
contextos diferentes. Não há holdout independente nem evidência de precisão ou
recall do modelo. Não divida estas linhas aleatoriamente para publicar métricas:
isso vazaria valores e templates. Crie avaliação separada por identidade, origem
e família de contexto, com revisão humana, antes de uso em produção. O corpus
`benchmarks/pii-accuracy/corpus.json` foi preservado e não foi usado no gerador.

O Laya retorna decisões, não os offsets necessários para mascarar. A integração
futura deve combinar as decisões com extratores de spans, validar limites UTF-8
e conservar os detectores determinísticos, o vault efêmero e a detokenização
existentes. As anotações não devem virar logs de produção. Esta entrega não
altera o gateway nem implementa o sidecar.
