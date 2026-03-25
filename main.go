package main

import (
	_ "embed"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed "Inter/Inter Variable/Inter.ttf"
var interFont []byte

//go:embed "Inter/Inter Hinted for Windows/Desktop/Inter-Bold.ttf"
var interBoldFont []byte

// Document holds all data for an invoice or estimate.
type Document struct {
	Id   string `json:"id"   yaml:"id"`
	Logo string `json:"logo" yaml:"logo"`
	From string `json:"from" yaml:"from"`
	To   string `json:"to"   yaml:"to"`
	Date string `json:"date" yaml:"date"`
	Due  string `json:"due"  yaml:"due"`

	Items      []string  `json:"items"      yaml:"items"`
	Quantities []int     `json:"quantities" yaml:"quantities"`
	Rates      []float64 `json:"rates"      yaml:"rates"`

	// Tax and Discount are whole-number percentages (e.g. 8 = 8%).
	Tax      float64 `json:"tax"      yaml:"tax"`
	Discount float64 `json:"discount" yaml:"discount"`
	Currency string  `json:"currency" yaml:"currency"`

	Note string `json:"note" yaml:"note"`

	BitcoinAddress   string `json:"bitcoin_address"   yaml:"bitcoin_address"`
	LightningAddress string `json:"lightning_address" yaml:"lightning_address"`
}

func DefaultDocument() Document {
	return Document{
		Id:         time.Now().Format("060102"),
		Rates:      []float64{25},
		Quantities: []int{2},
		Items:      []string{"Paper Cranes"},
		From:       "Project Folded, Inc.",
		To:         "Untitled Corporation, Inc.",
		Date:       time.Now().Format("Jan 02, 2006"),
		Tax:        0,
		Discount:   0,
		Currency:   "USD",
	}
}

var activeTheme Theme

// buildPDF constructs and saves the PDF for the given Document.
// When estimate is true the document is rendered as an estimate: the title
// becomes "ESTIMATE", the recipient label becomes "PREPARED FOR", and the
// due date / payment sections are suppressed.
func buildPDF(doc *Document, outputPath string, estimate bool) error {
	// Validate that items/quantities/rates have matching counts and warn on mismatch.
	n := len(doc.Items)
	if len(doc.Quantities) != n {
		fmt.Printf("warning: %d item(s) but %d quantity value(s) — missing quantities default to 1\n", n, len(doc.Quantities))
	}
	if len(doc.Rates) != n {
		fmt.Printf("warning: %d item(s) but %d rate value(s) — missing rates default to 0.00\n", n, len(doc.Rates))
	}

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{
		PageSize: *gopdf.PageSizeLetter,
	})
	pdf.SetMargins(40, 40, 40, 40)
	pdf.AddPage()

	if err := pdf.AddTTFFontData("Inter", interFont); err != nil {
		return err
	}
	if err := pdf.AddTTFFontData("Inter-Bold", interBoldFont); err != nil {
		return err
	}

	title := "INVOICE"
	billToLabel := "BILL TO"
	if estimate {
		title = "ESTIMATE"
		billToLabel = "PREPARED FOR"
	}

	writeLogo(&pdf, doc.Logo, doc.From)
	writeTitle(&pdf, title, doc.Id, doc.Date)
	writeBillTo(&pdf, doc.To, billToLabel)
	writeHeaderRow(&pdf)

	subtotal := 0.0
	for i := range doc.Items {
		q := 1
		if i < len(doc.Quantities) {
			q = doc.Quantities[i]
		}
		r := 0.0
		if i < len(doc.Rates) {
			r = doc.Rates[i]
		}
		writeRow(&pdf, doc, doc.Items[i], q, r)
		subtotal += float64(q) * r
	}

	// Convert whole-number percentages to multipliers.
	taxAmount := subtotal * (doc.Tax / 100)
	discountAmount := subtotal * (doc.Discount / 100)

	writeNotesAndTotals(&pdf, doc, doc.Note, subtotal, taxAmount, discountAmount)

	if !estimate && doc.Due != "" {
		writeDueDate(&pdf, doc.Due)
	}
	if !estimate && (doc.BitcoinAddress != "" || doc.LightningAddress != "") {
		writePayments(&pdf, doc.BitcoinAddress, doc.LightningAddress)
	}
	writeFooter(&pdf, doc.Id)

	outputPath = strings.TrimSuffix(outputPath, ".pdf") + ".pdf"
	if err := pdf.WritePdf(outputPath); err != nil {
		return err
	}
	fmt.Printf("Generated %s\n", outputPath)
	return nil
}

// resolveOutputPath returns a sensible default output path when none was given.
func resolveOutputPath(given, id, ext string) string {
	if given != "" {
		return given
	}
	return id + "." + ext
}

// loadActiveTheme resolves the theme from the flag value, falling back to the
// INVOICE_THEME env var (via viper), then to "default".
func loadActiveTheme(flagValue string) error {
	resolved := flagValue
	if resolved == "default" {
		if v := viper.GetString("theme"); v != "" {
			resolved = v
		}
	}
	var err error
	activeTheme, err = loadTheme(resolved)
	return err
}

var rootCmd = &cobra.Command{
	Use:   "invoice",
	Short: "Invoice generates invoices and estimates from the command line.",
	Long:  `Invoice generates invoices and estimates from the command line.`,
}

// --- shared flag registration ------------------------------------------------

// registerDocFlags adds all Document-level flags to cmd, binding each scalar
// flag to viper so that INVOICE_<FLAG> environment variables work automatically.
func registerDocFlags(cmd *cobra.Command, doc *Document) {
	defaults := DefaultDocument()

	cmd.Flags().StringVar(&doc.Id, "id", defaults.Id, "Document ID")
	cmd.Flags().StringVarP(&doc.Logo, "logo", "l", "", "Path to logo image")
	cmd.Flags().StringVarP(&doc.From, "from", "f", defaults.From, "Issuing company")
	cmd.Flags().StringVarP(&doc.To, "to", "t", defaults.To, "Recipient")
	cmd.Flags().StringVar(&doc.Date, "date", defaults.Date, "Document date")
	cmd.Flags().StringVar(&doc.Due, "due", "", "Payment due date (leave empty to omit)")

	cmd.Flags().Float64SliceVarP(&doc.Rates, "rate", "r", defaults.Rates, "Rates per unit (repeatable)")
	cmd.Flags().IntSliceVarP(&doc.Quantities, "quantity", "q", defaults.Quantities, "Quantities (repeatable)")
	cmd.Flags().StringSliceVarP(&doc.Items, "item", "i", defaults.Items, "Line item descriptions (repeatable)")

	cmd.Flags().Float64Var(&doc.Tax, "tax", defaults.Tax, "Tax percentage (e.g. 8 for 8%)")
	cmd.Flags().Float64VarP(&doc.Discount, "discount", "d", defaults.Discount, "Discount percentage (e.g. 10 for 10%)")
	cmd.Flags().StringVarP(&doc.Currency, "currency", "c", defaults.Currency, "Currency code (e.g. USD, EUR, GBP)")

	cmd.Flags().StringVarP(&doc.Note, "note", "n", "", "Optional note printed at the bottom")

	// Bind scalar flags to viper so INVOICE_<FLAG> env vars are respected.
	for _, name := range []string{"id", "logo", "from", "to", "date", "due", "tax", "discount", "currency", "note"} {
		_ = viper.BindPFlag(name, cmd.Flags().Lookup(name))
	}
}

// --- generate command --------------------------------------------------------

var (
	generateDoc       = Document{}
	generateOutput    string
	generateTheme     string
	generateEstimate  bool
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate an invoice or estimate PDF",
	Long:  `Generate an invoice (default) or estimate as a PDF.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		applyViperDefaults(&generateDoc, cmd)

		if err := loadActiveTheme(generateTheme); err != nil {
			return err
		}
		if generateDoc.Logo == "" && activeTheme.Logo != "" {
			generateDoc.Logo = activeTheme.Logo
		}

		outPath := resolveOutputPath(generateOutput, generateDoc.Id, "pdf")
		return buildPDF(&generateDoc, outPath, generateEstimate)
	},
}

// --- export command ----------------------------------------------------------

var (
	exportDoc    = Document{}
	exportOutput string
	exportTheme  string
	exportFormat string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export document data to YAML or JSON",
	Long:  `Export document data to a YAML (default) or JSON file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		applyViperDefaults(&exportDoc, cmd)

		if err := loadActiveTheme(exportTheme); err != nil {
			return err
		}
		if exportDoc.Logo == "" && activeTheme.Logo != "" {
			exportDoc.Logo = activeTheme.Logo
		}

		format := exportFormat
		if format == "" {
			format = "yaml"
		}
		outPath := resolveOutputPath(exportOutput, exportDoc.Id, strings.ToLower(format))
		return exportDocument(&exportDoc, format, outPath)
	},
}

// --- import command ----------------------------------------------------------

var (
	importDoc      = Document{}
	importOutput   string
	importTheme    string
	importEstimate bool
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import a YAML or JSON file and generate a PDF",
	Long:  `Import document data from a YAML or JSON file and generate a PDF. Any flags provided override the file values.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := importData(args[0], &importDoc, cmd.Flags()); err != nil {
			return err
		}

		if err := loadActiveTheme(importTheme); err != nil {
			return err
		}
		if importDoc.Logo == "" && activeTheme.Logo != "" {
			importDoc.Logo = activeTheme.Logo
		}

		outPath := resolveOutputPath(importOutput, importDoc.Id, "pdf")
		return buildPDF(&importDoc, outPath, importEstimate)
	},
}

// applyViperDefaults fills in any Document fields that were not explicitly set
// by a CLI flag but have a value in viper (env var or config file).
func applyViperDefaults(doc *Document, cmd *cobra.Command) {
	applyStringViper := func(flagName string, field *string) {
		if !cmd.Flags().Changed(flagName) {
			if v := viper.GetString(flagName); v != "" {
				*field = v
			}
		}
	}
	applyFloat64Viper := func(flagName string, field *float64) {
		if !cmd.Flags().Changed(flagName) {
			if v := viper.GetFloat64(flagName); v != 0 {
				*field = v
			}
		}
	}

	applyStringViper("id", &doc.Id)
	applyStringViper("logo", &doc.Logo)
	applyStringViper("from", &doc.From)
	applyStringViper("to", &doc.To)
	applyStringViper("date", &doc.Date)
	applyStringViper("due", &doc.Due)
	applyStringViper("currency", &doc.Currency)
	applyStringViper("note", &doc.Note)
	applyFloat64Viper("tax", &doc.Tax)
	applyFloat64Viper("discount", &doc.Discount)
}

func init() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("INVOICE")

	// generate
	registerDocFlags(generateCmd, &generateDoc)
	generateCmd.Flags().BoolVarP(&generateEstimate, "estimate", "e", false, "Render as an estimate instead of an invoice")
	generateCmd.Flags().StringVar(&generateDoc.BitcoinAddress, "bitcoin", "", "Bitcoin on-chain address for payment section")
	generateCmd.Flags().StringVar(&generateDoc.LightningAddress, "lightning", "", "Lightning address or BOLT11 invoice for payment section")
	generateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "Output file path (default: <id>.pdf)")
	generateCmd.Flags().StringVar(&generateTheme, "theme", "default", "Theme name or path to a .yaml/.json theme file")
	_ = viper.BindPFlag("theme", generateCmd.Flags().Lookup("theme"))

	// export
	registerDocFlags(exportCmd, &exportDoc)
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "Output file path (default: <id>.yaml or <id>.json)")
	exportCmd.Flags().StringVar(&exportTheme, "theme", "default", "Theme name or path to a .yaml/.json theme file")
	exportCmd.Flags().StringVar(&exportFormat, "format", "yaml", "Export format: yaml or json")

	// import
	registerDocFlags(importCmd, &importDoc)
	importCmd.Flags().BoolVarP(&importEstimate, "estimate", "e", false, "Render as an estimate instead of an invoice")
	importCmd.Flags().StringVar(&importDoc.BitcoinAddress, "bitcoin", "", "Bitcoin on-chain address for payment section")
	importCmd.Flags().StringVar(&importDoc.LightningAddress, "lightning", "", "Lightning address or BOLT11 invoice for payment section")
	importCmd.Flags().StringVarP(&importOutput, "output", "o", "", "Output file path (default: <id>.pdf)")
	importCmd.Flags().StringVar(&importTheme, "theme", "default", "Theme name or path to a .yaml/.json theme file")
}

func main() {
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(importCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
