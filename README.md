# SKTTS-Go

API de Text-to-Speech (TTS) desenvolvida em Go utilizando o Piper.

## Recursos

- Text-to-Speech via API REST
- 14 vozes Piper
- Português Brasileiro, Inglês e Espanhol
- Seleção de idioma e voz
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

## Idiomas e vozes

### Português Brasileiro — pt-BR

Cadu, Dii, Edresson, Faber, Jeff, Miro, Razo, Wesley

### Inglês — en-US

hfc_female, hfc_male, Joe, John

### Espanhol — es-ES

CarlFM, Sharvard

## Endpoints

### GET /health

Verifica se a API está funcionando.

### GET /info

Retorna informações da API, incluindo idiomas, vozes, vozes agrupadas por idioma e formatos disponíveis.

### POST /tts

Gera áudio a partir de um texto.

Parâmetros:

- text — texto que será convertido em áudio
- language — idioma da voz
- voice — voz utilizada
- format — formato de saída

## Exemplo PT-BR

curl -X POST -H "X-API-Key: SUA_API_KEY" --data-urlencode "text=Olá, este é um teste." --data-urlencode "language=pt-BR" --data-urlencode "voice=Cadu" --data-urlencode "format=ogg" http://localhost:8000/tts --output audio.ogg

## Exemplo EN-US

curl -X POST -H "X-API-Key: SUA_API_KEY" --data-urlencode "text=Hello, this is a test." --data-urlencode "language=en-US" --data-urlencode "voice=Joe" --data-urlencode "format=ogg" http://localhost:8000/tts --output audio.ogg

## Exemplo ES-ES

curl -X POST -H "X-API-Key: SUA_API_KEY" --data-urlencode "text=Hola, esta es una prueba." --data-urlencode "language=es-ES" --data-urlencode "voice=Sharvard" --data-urlencode "format=ogg" http://localhost:8000/tts --output audio.ogg

## Compatibilidade

Clientes antigos podem continuar utilizando o parâmetro voice sem informar language.

Exemplo:

curl -X POST -H "X-API-Key: SUA_API_KEY" --data-urlencode "text=Olá, este é um teste." --data-urlencode "voice=Wesley" --data-urlencode "format=ogg" http://localhost:8000/tts --output audio.ogg

Quando language é informado, a voz é resolvida dentro do idioma solicitado.

## Formatos

- OGG
- OPUS
- MP3
- WAV
- FLAC
- M4A
- WebM

## Documentação

A documentação integrada está disponível em:

GET /docs

## Configuração

XTTS_API_KEY=
SKTTS_ADMIN_KEY=
SKTTS_LINKEDIN_FREE_KEY=
SKTTS_DB=/opt/xtts/data/sktss.db
SKTTS_ADDR=:8000

Consulte o arquivo .env.example.

## Execução

go build -o sktts-go .

./sktts-go

## Estrutura

sktts-go/
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
