# invoice

A command-line tool for generating professional invoices and estimates as PDFs.

![demo](demo.gif)

---

## Installation

```bash
go install github.com/jgb95/invoice@latest
```

Or clone and build locally:

```bash
git clone https://github.com/jgb95/invoice.git
cd invoice
go build -o invoice .
```

---

## Quick start

```bash
# Generate an invoice PDF with defaults
invoice generate

# Generate an estimate instead of an invoice
invoice generate --estimate

# Specify the key fields
invoice generate \
  --from "Acme LLC\n123 Main St" \
  --to   "Client Corp\n456 Oak Ave" \
  --item "Logo design" --quantity 1 --rate 1200 \
  --item "Brand guide"  --quantity 1 --rate 800 \
  --tax 8 \
  --due "April 30, 2026" \
  --output invoice.pdf
```

---

## Commands

### `generate`

Generate a PDF directly from flags.

```
invoice generate [flags]
```

### `export`

Save document data to a YAML or JSON file (no PDF generated).

```
invoice export [flags]
invoice export --format json --output quote.json
```

### `import`

Load a previously exported YAML/JSON file and render it as a PDF. Any flag
provided on the command line overrides the value in the file.

```
invoice import <file> [flags]
invoice import data.yaml
invoice import data.yaml --estimate          # render the same data as an estimate
invoice import data.yaml --to "New Client"   # override the recipient
```

---

## Flags

All three commands share the document flags below. `generate` and `import` also
accept `--estimate` and the payment flags.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--id` | | today (YYMMDD) | Document identifier |
| `--from` | `-f` | — | Issuing company / person (use `\n` for line breaks) |
| `--to` | `-t` | — | Recipient (use `\n` for line breaks) |
| `--logo` | `-l` | — | Path to a logo image |
| `--date` | | today | Document date |
| `--due` | | — | Payment due date (omit to hide the section) |
| `--item` | `-i` | — | Line item description (repeatable) |
| `--quantity` | `-q` | 1 | Quantity for each item (repeatable) |
| `--rate` | `-r` | — | Unit rate for each item (repeatable) |
| `--tax` | | 0 | Tax percentage (e.g. `8` for 8%) |
| `--discount` | `-d` | 0 | Discount percentage (e.g. `10` for 10%) |
| `--currency` | `-c` | USD | ISO 4217 currency code (e.g. `EUR`, `GBP`) |
| `--note` | `-n` | — | Optional note printed at the bottom |
| `--output` | `-o` | `<id>.pdf` | Output file path |
| `--theme` | | default | Theme name or path to a theme file |
| `--estimate` | `-e` | false | Render as an estimate (hides due date & payment section) |
| `--bitcoin` | | — | Bitcoin address for the payment section |
| `--lightning` | | — | Lightning address / BOLT11 for the payment section |

`export` also accepts:

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `yaml` | Output format: `yaml` or `json` |

---

## Multiple line items

Repeat `--item`, `--quantity`, and `--rate` in matching order:

```bash
invoice generate \
  --item "Discovery call" --quantity 2  --rate 150 \
  --item "Design"          --quantity 10 --rate 120 \
  --item "Development"     --quantity 20 --rate 140
```

---

## Invoices vs. Estimates

The `--estimate` flag is a **render-time decision** — it does not modify or get
saved to the data file. This means you can:

1. Export your data once: `invoice export`
2. Render as an invoice: `invoice import data.yaml`
3. Render the exact same data as an estimate: `invoice import data.yaml --estimate`

When `--estimate` is active:
- The document title is **ESTIMATE** (instead of INVOICE)
- The recipient label is **PREPARED FOR** (instead of BILL TO)
- The due date and payment (Bitcoin/Lightning) sections are hidden

---

## Themes

A theme is a YAML or JSON file that sets colors and an optional default logo.

```yaml
accent:         [30, 30, 30]
primary_text:   [30, 30, 30]
secondary_text: [100, 100, 100]
line:           [200, 200, 200]
logo:           "logo.png"      # optional default logo
```

Pass it with `--theme path/to/theme.yaml` or set the `INVOICE_THEME` environment
variable.

---

## Environment variables

Any flag can be set via an environment variable prefixed with `INVOICE_`:

```bash
export INVOICE_FROM="Acme LLC"
export INVOICE_CURRENCY="EUR"
export INVOICE_THEME="dark"
invoice generate --item "Widget" --rate 99
```

---

## Supported currencies

AED · AUD · BRL · CAD · CHF · CNY · CZK · DKK · EUR · GBP · HKD · HUF · IDR ·
ILS · INR · JPY · KRW · MXN · MYR · NOK · NZD · PHP · PLN · RUB · SAR · SEK ·
SGD · THB · TRY · TWD · UAH · USD · ZAR

Any other ISO 4217 code is accepted and displayed as-is (e.g. `XAG` → `XAG 99.00`).
