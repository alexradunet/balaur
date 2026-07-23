# Issue tracker: GitHub

Issues and PRDs for this repository live in [GitHub Issues](https://github.com/alexradunet/balaur/issues). Use the `gh` CLI from this clone so repository resolution follows the configured remote.

## Conventions

- Create: `gh issue create --title "..." --body-file <file>`.
- Read: `gh issue view <number> --comments --json number,title,body,state,labels,author,comments`.
- List: `gh issue list --state open --json number,title,body,labels,assignees,comments` with appropriate label filters.
- Comment: `gh issue comment <number> --body-file <file>`.
- Apply or remove labels: `gh issue edit <number> --add-label "..."` or `--remove-label "..."`.
- Close: `gh issue close <number> --comment "..."`.

Use temporary files outside the repository for multiline bodies when shell quoting would be fragile. Triage comments must carry the AI disclaimer required by the `triage` skill.

## Pull requests as a triage surface

**PRs as a request surface: no.** External pull requests do not appear in routine triage discovery. An explicitly named PR may still be inspected with `gh pr view <number> --comments` and `gh pr diff <number>`.

GitHub shares one number space across issues and pull requests. Resolve an ambiguous `#42` with `gh pr view 42`, falling back to `gh issue view 42`.

## Publishing and fetching

When a skill says to publish to the issue tracker, create a GitHub issue. When a skill says to fetch a ticket, read the full issue body, labels, author, and comments rather than relying on its title.

## Agent implementation

An issue labelled `ready-for-agent` is eligible for the visible-worker issue-to-PR workflow in `docs/agents/development-workflow.md`. User direction may explicitly override the label gate for one issue.

The human-steered lead starts visible workers that create an isolated worktree, branch, commits, push that non-main branch, and open a pull request. Workers must never merge the pull request or push directly to `main`.
