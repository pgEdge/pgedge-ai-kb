# Configuring Embedding Providers

The builder supports four embedding providers: OpenAI, Voyage AI,
Ollama, and Gemini. You enable each provider independently in the
`embeddings` section of the configuration file; the build calls every
enabled provider for every chunk and stores the resulting vectors in
separate database columns.

At least one provider must be enabled.

## Selecting Providers

Choose providers based on cost, privacy, and downstream consumer
requirements. The following guidance applies in most deployments:

- OpenAI offers strong general-purpose embeddings and competitive
  pricing. Use OpenAI when you want the broadest model ecosystem.

- Voyage AI specialises in retrieval-tuned embeddings and supports
  multiple model sizes. Use Voyage when you want a retrieval-focused
  provider.

- Ollama runs locally with no API key. Use Ollama for strict-privacy
  builds, air-gapped environments, or to avoid per-token costs.

- Gemini offers Google's embedding models through the Gemini API.
  Use Gemini when you want a Google-ecosystem provider or access to
  the `gemini-embedding-001` model.

Enabling multiple providers in one build is supported and stores all
vectors in the output database. Downstream consumers can then pick a
provider at query time without rebuilding the database.

## OpenAI

The OpenAI provider calls the public embeddings API. Supply the API
key in a file with mode `0600`:

```bash
echo "sk-..." > ~/.openai-api-key
chmod 600 ~/.openai-api-key
```

Enable the provider in your configuration file:

```yaml
embeddings:
    openai:
        enabled: true
        api_key_file: "~/.openai-api-key"
        model: "text-embedding-3-small"
        dimensions: 1536
```

The `model` field defaults to `text-embedding-3-small`. The
`dimensions` field is optional; supply it only for models that support
variable dimensions.

## Voyage AI

The Voyage AI provider works the same way. Store the API key:

```bash
echo "pa-..." > ~/.voyage-api-key
chmod 600 ~/.voyage-api-key
```

Then enable the provider:

```yaml
embeddings:
    voyage:
        enabled: true
        api_key_file: "~/.voyage-api-key"
        model: "voyage-3"
```

The `model` field defaults to `voyage-3`.

## Ollama

The Ollama provider talks to a local or remote Ollama instance over
HTTP. The builder posts to the `/api/embeddings` endpoint. Confirm the
instance is running and the desired model is pulled:

```bash
ollama serve &
ollama pull nomic-embed-text
```

Enable the provider in your configuration:

```yaml
embeddings:
    ollama:
        enabled: true
        endpoint: "http://localhost:11434"
        model: "nomic-embed-text"
```

The `endpoint` defaults to `http://localhost:11434`. The `model`
defaults to `nomic-embed-text`. Use `https://ollama.com` and supply
`api_key_file` if you target Ollama Cloud instead of a self-hosted
instance.

## Gemini

The Gemini provider calls the Google Gemini embeddings API. Store
the API key in a file with mode `0600`:

```bash
echo "AIza..." > ~/.gemini-api-key
chmod 600 ~/.gemini-api-key
```

Then enable the provider:

```yaml
embeddings:
    gemini:
        enabled: true
        api_key_file: "~/.gemini-api-key"
        model: "gemini-embedding-001"
```

The `model` field defaults to `gemini-embedding-001`.

## Retry Behaviour

The builder retries transient embedding API errors with exponential
backoff. The default retry limit is 5; control it with `--max-retries`:

```bash
# Five retries (default)
./bin/pgedge-ai-kb-builder --config build.yaml

# Higher retry budget for CI environments
./bin/pgedge-ai-kb-builder --config build.yaml --max-retries 50

# Retry indefinitely; rely on an external timeout
./bin/pgedge-ai-kb-builder --config build.yaml --max-retries 0
```

Context-length errors from Ollama are detected immediately and never
retried; the builder truncates oversized text progressively or skips
the offending chunk.

## Request Timeouts

The builder bounds embedding requests with two timeouts under the
`embeddings` section. The `request_timeout` value is the overall
wall-clock ceiling for one request, including every retry; the
`per_attempt_timeout` value bounds each individual HTTP attempt. A
stalled attempt that exceeds `per_attempt_timeout` is retried rather
than cancelling the whole request, so a single slow batch no longer
exhausts the retry budget. This matters for providers that embed large
batches in one call, such as Gemini.

In the following example, the `embeddings` section raises the overall
ceiling and bounds each attempt; both values accept Go duration strings
such as `90s` or `10m`:

```yaml
embeddings:
    request_timeout: "10m"
    per_attempt_timeout: "90s"
    openai:
        enabled: true
```

The `request_timeout` value defaults to `10m` and `per_attempt_timeout`
defaults to `90s`. The `per_attempt_timeout` value must remain below
`request_timeout` so retries have room to run.

## Managing Existing Embeddings

The builder includes two flags for working with embeddings on an
existing database.

### Adding Missing Embeddings

Enable a new provider after an initial build, then use
`--add-missing-embeddings` to generate vectors only for chunks that
lack them:

```bash
./bin/pgedge-ai-kb-builder --config build.yaml --add-missing-embeddings
```

The build skips chunks that already have embeddings for every enabled
provider.

### Clearing Embeddings

Use `--clear-embeddings <provider>` to remove vectors for one
provider. The flag accepts `openai`, `voyage`, `ollama`, or `gemini`:

```bash
./bin/pgedge-ai-kb-builder --config build.yaml --clear-embeddings openai
```

Run with `--add-missing-embeddings` afterwards to repopulate the
cleared column.
