package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	estimateOutput     string
	estimateThemeName  string
	estimateExportFmt  string
	estimateImportPath string
	estimate           = Invoice{}
)

func init() {
	defaultEst := DefaultInvoice()
	defaultEst.Title = "ESTIMATE"
	defaultEst.Due = ""

	estimateCmd.Flags().StringVar(&estimateImportPath, "import", "", "Imported file (.json/.yaml)")
	estimateCmd.Flags().StringVar(&estimate.Id, "id", time.Now().Format("060102"), "ID")
	estimateCmd.Flags().StringVar(&estimate.Title, "title", "ESTIMATE", "Title")

	estimateCmd.Flags().Float64SliceVarP(&estimate.Rates, "rate", "r", defaultEst.Rates, "Rates")
	estimateCmd.Flags().IntSliceVarP(&estimate.Quantities, "quantity", "q", defaultEst.Quantities, "Quantities")
	estimateCmd.Flags().StringSliceVarP(&estimate.Items, "item", "i", defaultEst.Items, "Items")

	estimateCmd.Flags().StringVarP(&estimate.Logo, "logo", "l", "", "Company logo")
	estimateCmd.Flags().StringVarP(&estimate.From, "from", "f", defaultEst.From, "Issuing company")
	estimateCmd.Flags().StringVarP(&estimate.To, "to", "t", defaultEst.To, "Recipient company")
	estimateCmd.Flags().StringVar(&estimate.Date, "date", defaultEst.Date, "Date")

	estimateCmd.Flags().Float64Var(&estimate.Tax, "tax", defaultEst.Tax, "Tax")
	estimateCmd.Flags().Float64VarP(&estimate.Discount, "discount", "d", defaultEst.Discount, "Discount")
	estimateCmd.Flags().StringVarP(&estimate.Currency, "currency", "c", defaultEst.Currency, "Currency")

	estimateCmd.Flags().StringVarP(&estimate.Note, "note", "n", "", "Note")
	estimateCmd.Flags().StringVarP(&estimateOutput, "output", "o", "estimate.pdf", "Output file (.pdf, .yaml, or .json)")
	estimateCmd.Flags().StringVar(&estimateThemeName, "theme", "default", "Theme name (default, bitcoin) or path to a .yaml/.json theme file")
	_ = viper.BindPFlag("theme", estimateCmd.Flags().Lookup("theme"))
	estimateCmd.Flags().StringVar(&estimateExportFmt, "export", "", "Export data as 'yaml' or 'json' instead of generating a PDF")
}

var estimateCmd = &cobra.Command{
	Use:   "estimate",
	Short: "Generate an estimate",
	Long:  `Generate an estimate (PDF, YAML, or JSON)`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if estimateImportPath != "" {
			err := importData(estimateImportPath, &estimate, cmd.Flags())
			if err != nil {
				return err
			}
		}

		// If exporting to YAML or JSON, skip PDF generation.
		if estimateExportFmt != "" {
			return exportEstimate(&estimate, estimateExportFmt, estimateOutput)
		}

		// Resolve theme — prefer env var INVOICE_THEME over flag default.
		resolvedTheme := viper.GetString("theme")
		if resolvedTheme == "" {
			resolvedTheme = estimateThemeName
		}
		var err error
		activeTheme, err = loadTheme(resolvedTheme)
		if err != nil {
			return err
		}

		// CLI --logo overrides the theme logo; fall back to theme logo if not set.
		if estimate.Logo == "" && activeTheme.Logo != "" {
			estimate.Logo = activeTheme.Logo
		}

		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{
			PageSize: *gopdf.PageSizeLetter,
		})
		pdf.SetMargins(40, 40, 40, 40)
		pdf.AddPage()
		err = pdf.AddTTFFontData("Inter", interFont)
		if err != nil {
			return err
		}
		err = pdf.AddTTFFontData("Inter-Bold", interBoldFont)
		if err != nil {
			return err
		}

		writeLogo(&pdf, estimate.Logo, estimate.From)
		writeTitle(&pdf, estimate.Title, estimate.Id, estimate.Date)
		writeBillTo(&pdf, estimate.To, "PREPARED FOR")
		writeHeaderRow(&pdf)

		subtotal := 0.0
		for i := range estimate.Items {
			q := 1
			if len(estimate.Quantities) > i {
				q = estimate.Quantities[i]
			}
			r := 0.0
			if len(estimate.Rates) > i {
				r = estimate.Rates[i]
			}
			writeRow(&pdf, estimate.Items[i], q, r)
			subtotal += float64(q) * r
		}
		if estimate.Note != "" {
			writeNotes(&pdf, estimate.Note)
		}
		writeTotals(&pdf, subtotal, subtotal*estimate.Tax, subtotal*estimate.Discount)
		writeFooter(&pdf, estimate.Id)

		estimateOutput = strings.TrimSuffix(estimateOutput, ".pdf") + ".pdf"
		err = pdf.WritePdf(estimateOutput)
		if err != nil {
			return err
		}

		fmt.Printf("Generated %s\n", estimateOutput)
		return nil
	},
}

// exportEstimate serialises the estimate to YAML or JSON and writes it to disk.
// The output filename is derived from the --output flag or defaults based on format.
func exportEstimate(inv *Invoice, format, outputPath string) error {
	format = strings.ToLower(format)

	var data []byte
	var err error
	var defaultExt string

	switch format {
	case "yaml", "yml":
		data, err = yaml.Marshal(inv)
		defaultExt = ".yaml"
	case "json":
		data, err = json.MarshalIndent(inv, "", "  ")
		defaultExt = ".json"
	default:
		return fmt.Errorf("unsupported export format %q (use 'yaml' or 'json')", format)
	}
	if err != nil {
		return fmt.Errorf("failed to serialise estimate: %w", err)
	}

	// If the output path still has a .pdf extension (default), replace it.
	if strings.HasSuffix(outputPath, ".pdf") {
		outputPath = strings.TrimSuffix(outputPath, ".pdf") + defaultExt
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	fmt.Printf("Exported %s\n", outputPath)
	return nil
}
