# Claude Standing Instructions

> Standing instructions for Claude Code when working on this project.

## Project Structure

The pgEdge AI Knowledgebase Builder is a single Go project that produces
a CLI binary for building SQLite knowledgebase databases used by other
pgEdge tools (the Postgres MCP Server, the AI DBA Workbench, and so on):

- `/cmd/kb-builder` - Command-line entry point.

- `/internal` - Core Go packages (sources, converters, chunkers,
  embeddings, database, types).

- `/docs` - Documentation in markdown format with lowercase filenames.

- `/examples` - Example configuration files.

## Key Files

Reference these files for project context:

- `docs/changelog.md` - Notable changes by release.

- `mkdocs.yml` - Documentation site navigation.

- `Makefile` - Build and test commands.

- `.claude/CLAUDE.md` - This file; project guidelines.

## Task Workflow

Follow this workflow for implementation tasks:

1. Read relevant code before proposing changes.

2. Run `make test` before marking implementation complete.

3. Update `docs/changelog.md` for user-facing changes.

## Documentation

### General Guidelines

- Place comprehensive documentation in `/docs`.

- Create an `index.md` as the entry point; link to this from the README.

- Wrap all markdown files at 79 characters or less.

- Use lowercase filenames for all files in `/docs`.

### Writing Style

- Use active voice.

- Write grammatically correct sentences between 7 and 20 words.

- Use semicolons to link related ideas or manage long sentences.

- Use articles (a, an, the) appropriately.

- Avoid ambiguous pronoun references; only use "it" when the referent is
  in the same sentence.

### Document Structure

- Use one first-level heading per file with multiple second-level headings.

- Limit third and fourth-level headings to prominent content only.

- Include an introductory sentence or paragraph after each heading.

- For Features or Overview sections, use the format: "The KB Builder
  includes the following features:" followed by a bulleted list.

### Lists

- Leave a blank line before the first item in any list or sub-list.

- Write each bullet as a complete sentence with articles.

- Do not bold bullet items.

- Use numbered lists only for sequential steps.

### Code Snippets

- Precede code with an explanatory sentence: "In the following example,
  the `command_name` command uses..."

- Use backticks for inline code: `SELECT * FROM table;`

- Use fenced code blocks with language tags for multi-line code.

- Format `stdio`, `stdin`, `stdout`, and `stderr` in backticks.

- Capitalise SQL keywords; use lowercase for variables.

### Links and References

- Link files outside `/docs` to their GitHub location.

- Include third-party installation/documentation links in Prerequisites.

- Link to the GitHub repo when referencing cloning or project work.

- Do not link to github.io.

### README.md Files

At the top of each README:

- GitHub Action badges for repository actions.

- Table of Contents mirroring the `mkdocs.yml` nav section.

- Link to online docs at docs.pgedge.com.

README body content:

- Getting started steps.

- Prerequisites with commands and third-party links.

- Build/install commands and minimal configuration notes.

At the end of each README:

- Issues link: "To report an issue with the software, visit:"

- Online docs link: "For more information, visit
  [docs.pgedge.com](https://docs.pgedge.com)"

- License (final line): "This project is licensed under the
  [PostgreSQL License](LICENSE.md)."

### Additional Documentation Requirements

- Match all sample output to actual output.

- Document all command-line options.

- Include well-commented examples for all configuration options.

- Keep documentation synchronized with code for CLI options, configuration,
  and environment variables.

- Update `changelog.md` with notable changes since the last release.

## Tests

- Provide unit tests for all Go packages.

- Execute tests with `go test` or `make test`.

- Write automated tests for all functions and features; use mocking where
  needed.

- Run all tests after any changes; check for errors and warnings that may
  be hidden by output redirection or truncation.

- Clean up temporary test files on completion; retain log files for
  debugging.

- Modify existing tests only when the tested functionality changes or to
  fix bugs.

- Include linting in standard test suites using locally installable tools.

- Enable coverage checking in standard test suites.

- Run `gofmt` on all Go files.

- Ensure `make test` runs all test suites.

## Security

- Never log API keys, tokens, or other credentials.

- Read API keys from files with restrictive permissions (mode 0600).

- Follow industry best practices for defensive secure coding.

- Review all changes for security implications; report potential issues.

- Validate all configuration input before use.

## Code Style

- Use four spaces for indentation in YAML and Markdown; Go uses tabs.

- Write readable, extensible, and appropriately modularised code.

- Minimise code duplication; refactor as needed.

- Follow language-specific best practices.

- Remove unused code.

- Include this copyright notice at the top of every source file (not
  configuration files); adjust comment style for the language:

  ```text
  /*-------------------------------------------------------------------------
   *
   * pgEdge AI Knowledgebase Builder
   *
   * Copyright (c) 2025 - 2026, pgEdge, Inc.
   * This software is released under The PostgreSQL License
   *
   *-------------------------------------------------------------------------
   */
  ```

## Example Checklist

When making changes, verify:

- [ ] Code uses tabs for Go and 4-space indentation for YAML/Markdown
- [ ] Tests added for new functionality
- [ ] All tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Documentation updated in `/docs`
- [ ] Markdown files properly formatted (79 chars, blank lines before lists)
- [ ] Security considerations addressed
- [ ] No temporary files left behind

## Questions?

If you're unsure about any of these guidelines, refer to:

- Existing code patterns in the repository.

- Documentation in `/docs`.

- Recent git commits for context.
