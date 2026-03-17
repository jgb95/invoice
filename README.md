<img width="1200" alt="Invoice" src="https://github.com/maaslalani/nap/assets/42545625/16dae9d9-390c-49b6-aedd-3f882b17f57b">

# Invoice

Generate invoices and estimates from the command line.

## Command Line Interface

### Generate an Invoice

```bash
invoice generate --from "Dream, Inc." --to "Imagine, Inc." \
    --item "Rubber Duck" --quantity 2 --rate 25 \
    --tax 0.13 --discount 0.15 \
    --note "For debugging purposes."
```

View the generated PDF at `invoice.pdf`. Customize the output location with `--output`.

```bash
open invoice.pdf
```

<img width="574" alt="Example invoice" src="https://github.com/maaslalani/nap/assets/42545625/13153de2-dfa1-41e6-a18e-4d3a5cea5b74">

### Generate an Estimate

```bash
invoice estimate --from "Dream, Inc." --to "Imagine, Inc." \
    --item "Rubber Duck" --quantity 2 --rate 25 \
    --note "Valid for 30 days."
```

Estimates can also be exported to YAML or JSON instead of PDF:

```bash
invoice estimate --export yaml --output estimate.yaml
invoice estimate --export json --output estimate.json
```

An exported estimate can later be imported into `generate` to convert it into a
full invoice — the title and due date are automatically updated:

```bash
invoice generate --import estimate.yaml --output invoice.pdf
```

### Invoice / Estimate IDs

By default, the ID is set to today's date in `YYMMDD` format (e.g. `260317` for
March 17, 2026). Override it with `--id`:

```bash
invoice generate --id 260317 ...
```

### Environment Variables

Save repeated information with environment variables:

```bash
export INVOICE_LOGO=/path/to/image.png
export INVOICE_FROM="Dream, Inc."
export INVOICE_TO="Imagine, Inc."
export INVOICE_TAX=0.13
export INVOICE_RATE=25
export INVOICE_THEME=bitcoin
```

Generate a new invoice using the saved defaults:

```bash
invoice generate \
    --item "Yellow Rubber Duck" --quantity 5 \
    --item "Special Edition Plaid Rubber Duck" --quantity 1 \
    --note "For debugging purposes." \
    --output duck-invoice.pdf
```

### Configuration File

Save repeated information with JSON or YAML:

```json
{
    "logo": "/path/to/image.png",
    "from": "Dream, Inc.",
    "to": "Imagine, Inc.",
    "tax": 0.13,
    "items": ["Yellow Rubber Duck", "Special Edition Plaid Rubber Duck"],
    "quantities": [5, 1],
    "rates": [25, 25]
}
```

Import the configuration file when generating:

```bash
invoice generate --import path/to/data.json --output duck-invoice.pdf
```

### Themes

Choose a built-in theme or supply your own:

```bash
# Built-in themes: default, bitcoin
invoice generate --theme bitcoin ...

# Or point to a custom theme file (.yaml or .json)
invoice generate --theme /path/to/mytheme.yaml ...
```

A custom theme file defines RGB colours for each element:

```yaml
primary_text:   [0, 0, 0]
secondary_text: [75, 75, 75]
accent:         [99, 102, 241]   # indigo
line:           [199, 210, 254]
logo:           /path/to/logo.png  # optional
```

The `INVOICE_THEME` environment variable is also supported.

### Bitcoin & Lightning Payments

Add a payment section with QR codes and clickable links:

```bash
invoice generate \
    --bitcoin "bc1qxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
    --lightning "lnbc..."
```

Both flags are optional — include one or both.

## Installation

Install with Go:

```sh
go install github.com/jgb95/invoice@main
```

Or download a binary from the [releases](https://github.com/jgb95/invoice/releases).

## License

[MIT](https://github.com/jgb95/invoice/blob/master/LICENSE)
