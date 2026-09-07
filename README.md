# SKTTS-Go

API de Text-to-Speech (TTS) desenvolvida em Go utilizando o Piper.

## Recursos

- Text-to-Speech via API REST
- Múltiplas vozes
- OGG, OPUS, MP3, WAV, FLAC, M4A e WebM
- API Keys
- Autenticação
- Planos e limites dinâmicos
- Limites por minuto e por dia
- Limites de caracteres
- Controle de concorrência
- Histórico de áudios
- Bloqueio de IP
- Documentação integrada
- Health Check

## Endpoints

### Health Check

GET /health

### Informações da API

GET /info

### Gerar áudio

POST /tts

Exemplo:

curl -X POST \
  -H "X-API-Key: SUA_API_KEY" \
  --data-urlencode "text=Olá, este é um teste." \
  --data-urlencode "format=ogg" \
  --data-urlencode "voice=Wesley" \
  http://localhost:8000/tts \
  --output audio.ogg

## Vozes

As vozes disponíveis podem ser consultadas através de:

GET /info

## Formatos

- OGG
- OPUS
- MP3
- WAV
- FLAC
- M4A
- WebM

## Documentação

A documentação está disponível em:

GET /docs

Exemplo:

http://localhost:8000/docs

## Configuração

Variáveis de ambiente:

XTTS_API_KEY=
SKTTS_ADMIN_KEY=
SKTTS_LINKEDIN_FREE_KEY=
SKTTS_DB=/opt/xtts/data/sktss.db
SKTTS_ADDR=:8000

Consulte o arquivo .env.example.

## Execução

Compilar:

go build -o sktts-go .

Executar:

./sktts-go

## Estrutura

sktts-go/
├── config/
├── database/
├── docs/
├── middleware/
├── models/
├── services/
├── docs.go
├── main.go
├── go.mod
├── go.sum
├── .env.example
└── .gitignore

## Status

Projeto em desenvolvimento.

## Licença

Projeto de uso privado.
