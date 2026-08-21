# Issue tracker: GitHub

Issues and specifications for this repository live in GitHub Issues.
Use the `gh` command-line interface for all operations.

Repository: `alexradunet/balaur`

## Operations

- Create: `gh issue create --title "..." --body "..."`
- Read: `gh issue view <number> --comments`
- List: `gh issue list --state open`
- Comment: `gh issue comment <number> --body "..."`
- Add a label: `gh issue edit <number> --add-label "..."`
- Remove a label: `gh issue edit <number> --remove-label "..."`
- Close: `gh issue close <number> --comment "..."`

Use a heredoc for multi-line issue bodies.

## Pull requests

**PRs as a request surface: no.**

GitHub shares one number range between issues and pull requests.
Check the item type before an operation.

## Skill operations

When a skill says to publish an issue, create a GitHub issue.

When a skill requests a ticket, run:

`gh issue view <number> --comments`

## Blocking relationships

Use GitHub issue dependencies when they are available.

Use this fallback when dependencies are unavailable:

`Blocked by: #<number>, #<number>`

A ticket is ready when all blocking issues are closed.

## Wayfinder operations

Use one issue with the `wayfinder:map` label as the map.

Use child issues as decision tickets.
Label them with the applicable `wayfinder:<type>` label.

Assign a ticket before work starts.
Close the ticket after its decision is recorded.
