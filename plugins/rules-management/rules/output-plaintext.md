# Plain-Text Output — No Pseudographics

INSTRUCTION: NEVER USE TABLES OR BOX-DRAWING CHARACTERS

Applies to chat responses AND every file you generate (`.md`, `.txt`, docs, runbooks,
reports, commit bodies, PR descriptions).

Banned:

- Markdown tables (`| col | col |` + `|---|---|`). The terminal renders these as
  box-drawing pseudographics (`└───────┴───────┘`), which are noise, break on narrow
  terminals, and are unusable when copy-pasted into a ticket or a config file.
- Box-drawing / line-drawing characters: `─ │ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼ ═ ║ ╔ ╗ ╚ ╝ ╠ ╣`
- ASCII-art table borders: `+-----+-----+`, `|-----|`
- Boxed callouts, banners, or figlet-style headers.

Use instead:

- Indented lists with `label: value`, or a leading count/key then an indented
  explanation.
- Plain section headers: a bare uppercase line, or `#`/`##` in Markdown.
- Blank lines and two-space indentation for grouping.
- Fenced code blocks for anything that gets copy-pasted (config, commands, IP lists).
  Align columns inside a code block with plain spaces if columns genuinely help.

Example — not this:

    | IP | Owner | Verdict |
    |---|---|---|
    | 3.64.0.0/12 | AWS euc1 | keep |

This:

    3.64.0.0/12   AWS eu-central-1 -- keep

Or this:

    3.64.0.0/12
        owner: AWS eu-central-1
        verdict: keep

Exception: the user explicitly asks for a table, or the target format requires one
(e.g. editing an existing file that already uses Markdown tables — match the file).
Diagrams the user requested (sequence, topology, tree) may use `|`, `+`, `-`, `/`, `\`
and arrows; they are content, not decoration.
