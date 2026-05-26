# sbx2pdf

`sbx2pdf` converts a **StudioBinder Exchange** (`.sbx`) file into a **print-ready PDF** with better pagination than StudioBinder’s native PDF export.

StudioBinder’s PDF export has two core issues this tool addresses:

- **Optional suppression** of **scene numbers** and/or **page numbers**.
- **Correct pagination** to avoid bad page breaks such as:
  - a **slug line** (scene heading) appearing as the last line of a page
  - a **speaker name** at the bottom of a page with their **dialogue split onto the next page**

## Installation

## Installation

### Mac / Linux

```bash
curl -fsSL https://privman.github.io/sbx-to-pdf/install.sh | bash
```

### Windows

```powershell
irm https://privman.github.io/sbx-to-pdf/install.ps1 | iex
```

## Usage

```bash
sbx2pdf <input.sbx> [--out <output.pdf>] [--omit-scene-nums] [--omit-page-nums] [--overwrite]
```

### Arguments

- **`<input.sbx>`**: Path to the input StudioBinder Exchange file.

### Options

- **`--out <output.pdf>`**: Output PDF path.
  - If omitted, defaults to the input filename with only the suffix replaced: `*.sbx` → `*.pdf`.
  - Example: `My_Script_v1.sbx` → `My_Script_v1.pdf`
- **`--omit-scene-nums`**: Do not render scene numbers in the output PDF.
- **`--omit-page-nums`**: Do not render page numbers in the output PDF.
- **`--overwrite`**: Allow overwriting an existing output file.

### Examples

Convert to default output name:

```bash
sbx2pdf My_Script_v1_05-26-26_18-57.sbx
```

Write to a specific output path and omit both scene numbers and page numbers:

```bash
sbx2pdf My_Script_v1_05-26-26_18-57.sbx --out My_Script_clean.pdf --omit-scene-nums --omit-page-nums
```

Overwrite an existing output file:

```bash
sbx2pdf My_Script_v1_05-26-26_18-57.sbx --out My_Script.pdf --overwrite
```

## What is an `.sbx` file?

StudioBinder Exchange files are screenplay exports that contain the script content in an HTML-like format (e.g. `<div class="divtype…">` blocks and attributes like `data-scene="…"`).

`sbx2pdf` treats the `.sbx` as the **source of truth** and produces a PDF that preserves screenplay content and semantic structure while enforcing better layout rules.

## Screenplay elements

StudioBinder defines the standard screenplay element types in [Understanding screenplay elements](https://support.studiobinder.com/en/articles/2941370-understanding-screenplay-elements). `sbx2pdf` supports the element types that appear in typical `.sbx` exports:

| StudioBinder element | Supported | SBX representation |
|---|---|---|
| Scene heading | Yes | `divtype0` with `data-scene="…"` |
| Action | Yes | `divtype2` |
| Character | Yes | `divtype3` |
| Dialogue | Yes | `divtype5` |
| Parenthetical | Yes | `divtype4` |
| Transition | Yes | `divtype6` |
| General | Yes | `divtype1` |
| Extension | No | From anecdotal data, it seems the SBX exports include extensions as part of the Character element |
| Subheaders | No | Unknown |

## Pagination and layout rules (the core of this tool)

The goal is to produce a PDF that reads like a professionally formatted screenplay and avoids disruptive page breaks.

### Orphan-avoidance rules

At minimum, `sbx2pdf` will enforce these constraints:

#### **Rule 1: Headings are never separated from the following paragraph**

For this rule, assume that dialogue and action elements are paragraphs, and all other elements, including scene slugs, character and general (the latter are often used for shots or subheadings) are headings.

This rule ensures no page ends on a heading.

If there isn't enough room in the page to include at least the first line of the next paragraph after a heading, a page break must be inserted after the last paragraph.

Note that scripts can contain multiple consecutive headings. E.g., a scene slug, followed by a character, followed by a parenthetical. In this case, all three must be kept with the first line of dialogue after the parenthetical.

#### **Rule 2: A page cannot start with a dialogue without a character name**

If a page break disects a dialogue paragraph, the last speaker name (character element) from the last page must be repeated in the PDF at the top of the page.

The repeated character speaker name will be suffixed by "(CONT'D)" in this case, unless it already ends with "(CONT'D)" or "(CONT)".

Any parenthetical that appears between the bisected dialogue and the last speaker name will also be repeated below the repeated speaker name.

### Block-level pagination strategy (intended behavior)

The renderer will treat screenplay components as layout blocks with “keep-with-next” semantics where appropriate. Examples:

- **Scene heading (slug line)**: keep with the next block (typically action).
- **Character**: keep with the dialogue that follows.
- **Dialogue**: keep lines together; if it must split across pages, follow standard screenplay conventions (exact split rules to be finalized).
- **Parentheticals**: keep with the dialogue lines they modify.

See [Screenplay elements](#screenplay-elements) for the `divtype` mapping used during layout.

