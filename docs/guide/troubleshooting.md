# Troubleshooting

This page collects fixes for common errors that occur during a build.
If you encounter an issue not covered here, file a bug at
[GitHub Issues](https://github.com/pgEdge/pgedge-ai-kb/issues).

## Build Time Errors

### "failed to read config file"

The builder cannot open the file passed to `--config`. Verify the
file exists and that the path is correct. The builder expands a
leading `~` to the user's home directory.

### "at least one embedding provider must be enabled"

The configuration disables all three providers. Set `enabled: true`
under `embeddings.openai`, `embeddings.voyage`, or `embeddings.ollama`
(or any combination).

### "OpenAI API key file is empty" (or "Voyage")

The configured `api_key_file` exists but contains no key (or only
whitespace). Write the key with `echo "sk-..." > ~/.openai-api-key`
followed by `chmod 600 ~/.openai-api-key`.

### "source N: must specify either git_url or local_path"

The Nth source defines neither `git_url` nor `local_path`. Add the
missing field. A source cannot specify both.

### "source N: project_name is required"

Every source must have a `project_name`. Add the field.

## Git Errors

### "Repository not found"

The build's `git_url` is wrong, the repository is private, or the
host has no network access. Confirm the URL works with `git ls-remote
<url>`. For private repositories, configure HTTPS credentials or run
the builder with SSH keys configured.

### "Authentication failed"

The host has the wrong credentials for the repository. The release
pipeline workflow uses the `PGEDGE_BUILDER_TOKEN` secret to
authenticate; local builds should use your own GitHub credential
helper.

### Stale Clone Behaviour

The builder pulls every Git source on each run. Pass `--skip-updates`
when iterating on local content to skip the pull:

```bash
./bin/pgedge-ai-kb-builder --config build.yaml --skip-updates
```

## Embedding Errors

### "context length exceeded" (Ollama)

The chunk exceeds the embedding model's context window. The chunker
sizes chunks for compatibility with `nomic-embed-text` (8192 tokens),
but unusually dense content may still overflow. The builder
truncates progressively and finally skips the chunk; the run
continues.

### Repeated 429 Errors (OpenAI or Voyage)

The provider is rate-limiting the build. Use a higher retry budget
or run during off-peak hours:

```bash
./bin/pgedge-ai-kb-builder --config build.yaml --max-retries 50
```

### "no embeddings returned from Ollama"

The Ollama instance is reachable but returned an empty response.
Confirm the model is pulled (`ollama list`) and that the endpoint URL
matches the running instance.

## Database Errors

### "database is locked"

Another process holds an exclusive lock on the SQLite file. Stop the
other process before rebuilding. The builder uses a single
connection, so concurrent runs against the same file are not safe.

### Database File Grows Larger Than Expected

Incremental rebuilds reclaim space only after `VACUUM`. Run the
following command if size matters:

```bash
sqlite3 pgedge-ai-kb.db "VACUUM;"
```

## Performance Tips

The following techniques speed up builds:

- Run with `--skip-updates` when source repositories have not
  changed.

- Enable only the providers you use; every enabled provider triggers
  an API call per chunk.

- Use Ollama for large builds to avoid per-token API costs.

- Use the `--add-missing-embeddings` flag after toggling a new
  provider so the builder skips already-processed chunks.

## Reporting Bugs

Bugs and feature requests belong on
[GitHub Issues](https://github.com/pgEdge/pgedge-ai-kb/issues).
Include the following information:

- Builder version (the `pgedge-ai-kb-builder --version` output once
  versions ship; otherwise the commit SHA).

- The configuration file, with API keys redacted.

- The complete error output and the last several seconds of build
  logs.

- Any non-default flags passed on the command line.
