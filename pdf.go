package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"

	"github.com/signintech/gopdf"
	qrcode "github.com/skip2/go-qrcode"
)

const (
	// Item table column X positions.
	quantityColumnOffset = 370 // QTY header / values
	rateColumnOffset     = 420 // RATE header / values
	// amountRightX is the right edge for both the line-item AMOUNT column and the totals block.
	// Everything right-aligns here so decimals always line up across the whole document.
	amountRightX = 572

	// totalsLabelX is where the label column in the totals/due-date block starts.
	totalsLabelX = 330
)

const (
	subtotalLabel = "Subtotal"
	discountLabel = "Discount"
	taxLabel      = "Tax"
	totalLabel    = "Total"
)

// --- helpers -----------------------------------------------------------------

func setTextColor(pdf *gopdf.GoPdf, c Color) {
	pdf.SetTextColor(uint8(c[0]), uint8(c[1]), uint8(c[2]))
}

func setStrokeColor(pdf *gopdf.GoPdf, c Color) {
	pdf.SetStrokeColor(uint8(c[0]), uint8(c[1]), uint8(c[2]))
}

func setFillColor(pdf *gopdf.GoPdf, c Color) {
	pdf.SetFillColor(uint8(c[0]), uint8(c[1]), uint8(c[2]))
}

// currencySymbol returns the symbol for a currency code, falling back to the
// code itself (e.g. "CAD") when not found in the lookup table.
func currencySymbol(code string) string {
	if sym, ok := currencySymbols[code]; ok {
		return sym
	}
	if code != "" {
		return code + " "
	}
	return ""
}

// --- sections ----------------------------------------------------------------

func writeLogo(pdf *gopdf.GoPdf, logo string, from string) {
	if logo != "" {
		width, height := getImageDimension(logo)
		scaledWidth := 100.0
		scaledHeight := float64(height) * scaledWidth / float64(width)
		_ = pdf.Image(logo, pdf.GetX(), pdf.GetY(), &gopdf.Rect{W: scaledWidth, H: scaledHeight})
		pdf.Br(scaledHeight + 24)
	}
	setTextColor(pdf, activeTheme.Accent)

	formattedFrom := strings.ReplaceAll(from, `\n`, "\n")
	fromLines := strings.Split(formattedFrom, "\n")

	for i, line := range fromLines {
		if i == 0 {
			_ = pdf.SetFont("Inter", "", 12)
			_ = pdf.Cell(nil, line)
			pdf.Br(18)
		} else {
			_ = pdf.SetFont("Inter", "", 10)
			_ = pdf.Cell(nil, line)
			pdf.Br(15)
		}
	}
	pdf.Br(21)
	setStrokeColor(pdf, activeTheme.Line)
	pdf.Line(pdf.GetX(), pdf.GetY(), 260, pdf.GetY())
	pdf.Br(36)
}

func writeTitle(pdf *gopdf.GoPdf, title, id, date string) {
	_ = pdf.SetFont("Inter-Bold", "", 24)
	setTextColor(pdf, activeTheme.Accent)
	_ = pdf.Cell(nil, title)
	pdf.Br(36)
	_ = pdf.SetFont("Inter", "", 12)
	setTextColor(pdf, activeTheme.SecondaryText)
	_ = pdf.Cell(nil, "#")
	_ = pdf.Cell(nil, id)
	setTextColor(pdf, activeTheme.Line)
	_ = pdf.Cell(nil, "  ·  ")
	setTextColor(pdf, activeTheme.SecondaryText)
	_ = pdf.Cell(nil, date)
	pdf.Br(48)
}

func writeDueDate(pdf *gopdf.GoPdf, due string) {
	_ = pdf.SetFont("Inter", "", 9)
	setTextColor(pdf, activeTheme.SecondaryText)
	pdf.SetX(totalsLabelX)
	_ = pdf.Cell(nil, "Due Date")
	setTextColor(pdf, activeTheme.PrimaryText)
	_ = pdf.SetFontSize(11)
	// Right-align the due date value at amountRightX.
	if w, err := pdf.MeasureTextWidth(due); err == nil {
		pdf.SetX(amountRightX - w)
	}
	_ = pdf.Cell(nil, due)
	pdf.Br(12)
}

func writeBillTo(pdf *gopdf.GoPdf, to string, label string) {
	setTextColor(pdf, activeTheme.SecondaryText)
	_ = pdf.SetFont("Inter", "", 9)
	_ = pdf.Cell(nil, label)
	pdf.Br(18)

	formattedTo := strings.ReplaceAll(to, `\n`, "\n")
	toLines := strings.Split(formattedTo, "\n")

	for i, line := range toLines {
		if i == 0 {
			_ = pdf.SetFont("Inter", "", 15)
			setTextColor(pdf, activeTheme.SecondaryText)
			_ = pdf.Cell(nil, line)
			pdf.Br(20)
		} else {
			_ = pdf.SetFont("Inter", "", 10)
			setTextColor(pdf, activeTheme.SecondaryText)
			_ = pdf.Cell(nil, line)
			pdf.Br(15)
		}
	}
	pdf.Br(64)
}

func writeHeaderRow(pdf *gopdf.GoPdf) {
	_ = pdf.SetFont("Inter", "", 9)
	setTextColor(pdf, activeTheme.Accent)
	_ = pdf.Cell(nil, "ITEM")
	pdf.SetX(quantityColumnOffset)
	_ = pdf.Cell(nil, "QTY")
	pdf.SetX(rateColumnOffset)
	_ = pdf.Cell(nil, "RATE")
	// Right-align "AMOUNT" header to amountRightX.
	if w, err := pdf.MeasureTextWidth("AMOUNT"); err == nil {
		pdf.SetX(amountRightX - w)
	}
	_ = pdf.Cell(nil, "AMOUNT")
	pdf.Br(24)
}

func writeRow(pdf *gopdf.GoPdf, doc *Document, item string, quantity int, rate float64) {
	_ = pdf.SetFont("Inter", "", 11)
	setTextColor(pdf, activeTheme.PrimaryText)

	sym := currencySymbol(doc.Currency)
	total := float64(quantity) * rate
	amountStr := sym + strconv.FormatFloat(total, 'f', 2, 64)

	_ = pdf.Cell(nil, item)
	pdf.SetX(quantityColumnOffset)
	_ = pdf.Cell(nil, strconv.Itoa(quantity))
	pdf.SetX(rateColumnOffset)
	_ = pdf.Cell(nil, sym+strconv.FormatFloat(rate, 'f', 2, 64))
	// Right-align amount value to amountRightX.
	if w, err := pdf.MeasureTextWidth(amountStr); err == nil {
		pdf.SetX(amountRightX - w)
	}
	_ = pdf.Cell(nil, amountStr)
	pdf.Br(24)
}

// writeNotesAndTotals renders the notes section (left column) and the subtotal/
// tax/discount/total block (right column) at the same Y position, dynamically
// positioned below the last line item rather than at a hardcoded offset.
func writeNotesAndTotals(pdf *gopdf.GoPdf, doc *Document, notes string, subtotal, tax, discount float64) {
	// Use a minimum Y so content never crowds the line items, but don't
	// hardcode a single fixed value — allow more items to push it down.
	const minY = 500.0
	currentY := pdf.GetY()
	if currentY < minY {
		currentY = minY
	}
	pdf.SetY(currentY)

	if notes != "" {
		writeNotes(pdf, notes, currentY)
	}
	writeTotals(pdf, doc, subtotal, tax, discount, currentY)
}

func writeNotes(pdf *gopdf.GoPdf, notes string, startY float64) {
	pdf.SetXY(40, startY)

	_ = pdf.SetFont("Inter", "", 9)
	setTextColor(pdf, activeTheme.Accent)
	_ = pdf.Cell(nil, "NOTES")
	pdf.Br(18)
	_ = pdf.SetFont("Inter", "", 9)
	setTextColor(pdf, activeTheme.PrimaryText)

	formattedNotes := strings.ReplaceAll(notes, `\n`, "\n")
	for _, line := range strings.Split(formattedNotes, "\n") {
		pdf.SetX(40)
		_ = pdf.Cell(nil, line)
		pdf.Br(15)
	}
}

func writeTotals(pdf *gopdf.GoPdf, doc *Document, subtotal, tax, discount float64, startY float64) {
	pdf.SetXY(40, startY)

	writeTotal(pdf, doc, subtotalLabel, subtotal, false)
	if tax > 0 {
		label := formatPercentLabel(taxLabel, doc.Tax)
		writeTotal(pdf, doc, label, tax, false)
	}
	if discount > 0 {
		label := formatPercentLabel(discountLabel, doc.Discount)
		// Discount is displayed as a negative value to make it visually clear.
		writeTotal(pdf, doc, label, -discount, false)
	}
	writeTotal(pdf, doc, totalLabel, subtotal+tax-discount, true)
}

// formatPercentLabel returns a label like "Tax (8.25%)" or "Discount (5%)".
// Trailing zeros after the decimal are stripped (e.g. 5.00 → "5%").
func formatPercentLabel(base string, pct float64) string {
	// Use %g to strip unnecessary trailing zeros.
	return base + " (" + strconv.FormatFloat(pct, 'f', -1, 64) + "%)"
}

func writeTotal(pdf *gopdf.GoPdf, doc *Document, label string, amount float64, isTotal bool) {
	// Draw the label in the left part of the totals block.
	_ = pdf.SetFont("Inter", "", 9)
	setTextColor(pdf, activeTheme.SecondaryText)
	pdf.SetX(totalsLabelX)
	_ = pdf.Cell(nil, label)

	// Format the amount string.
	sym := currencySymbol(doc.Currency)
	var formatted string
	if amount < 0 {
		formatted = "-" + sym + strconv.FormatFloat(-amount, 'f', 2, 64)
	} else {
		formatted = sym + strconv.FormatFloat(amount, 'f', 2, 64)
	}

	// Set font for the amount, then right-align it at totalsAmountRightX.
	if isTotal {
		_ = pdf.SetFont("Inter-Bold", "", 11.5)
		setTextColor(pdf, activeTheme.Accent)
	} else {
		_ = pdf.SetFont("Inter", "", 12)
		setTextColor(pdf, activeTheme.PrimaryText)
	}
	if w, err := pdf.MeasureTextWidth(formatted); err == nil {
		pdf.SetX(amountRightX - w)
	}
	_ = pdf.Cell(nil, formatted)
	pdf.Br(24)
}

// wrapText splits a string into lines that fit within maxWidth at the current
// font. Long addresses with no spaces are character-wrapped.
func wrapText(pdf *gopdf.GoPdf, text string, maxWidth float64) []string {
	var lines []string
	for _, segment := range strings.Split(text, "\n") {
		words := strings.Fields(segment)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		current := words[0]
		for _, word := range words[1:] {
			candidate := current + " " + word
			measuredW, err := pdf.MeasureTextWidth(candidate)
			if err != nil || measuredW > maxWidth {
				lines = append(lines, current)
				current = word
			} else {
				current = candidate
			}
		}
		// If the single "word" (e.g. a BOLT11 invoice) is still too wide,
		// character-wrap it.
		if mw, err := pdf.MeasureTextWidth(current); err == nil && mw > maxWidth {
			runes := []rune(current)
			chunk := ""
			for _, r := range runes {
				candidate := chunk + string(r)
				cw, err2 := pdf.MeasureTextWidth(candidate)
				if err2 == nil && cw > maxWidth {
					lines = append(lines, chunk)
					chunk = string(r)
				} else {
					chunk = candidate
				}
			}
			if chunk != "" {
				lines = append(lines, chunk)
			}
		} else {
			lines = append(lines, current)
		}
	}
	return lines
}

// writePayments renders a payment section with QR codes and clickable URIs for
// Bitcoin and/or Lightning addresses.
//
// Layout (US Letter, 40 pt margins):
//
//	contentWidth = 612 - 2×40 = 532 pt
//	Each block  = qrSize + colGap + textColWidth
//	Two blocks  = 2×block + blockGap  ≤ contentWidth
//	One block   = block  (centered)
func writePayments(pdf *gopdf.GoPdf, bitcoinAddr, lightningAddr string) {
	// Leave a small gap after the due date / totals area.
	pdf.Br(24)

	// Section heading.
	_ = pdf.SetFont("Inter", "", 9)
	setTextColor(pdf, activeTheme.Accent)
	_ = pdf.Cell(nil, "PAYMENT")
	pdf.Br(18)

	const (
		pageWidth   = 612.0 // US Letter
		margin      = 40.0
		qrSize      = 80.0 // QR code display size in points
		colGap      = 12.0 // gap between QR and its text column
		blockGap    = 24.0 // gap between two side-by-side blocks
		fontSize    = 7.0
		lineH       = 10.0 // line height for address text
		labelExtraH = 14.0 // space reserved for the block label above the address
	)

	contentWidth := pageWidth - 2*margin // 532

	// Decide how many blocks are present.
	twoBlocks := bitcoinAddr != "" && lightningAddr != ""

	// Work out textColWidth so that everything fits within contentWidth.
	var textColWidth float64
	if twoBlocks {
		// 2*(qrSize + colGap + textColWidth) + blockGap = contentWidth
		textColWidth = (contentWidth - 2*(qrSize+colGap) - blockGap) / 2
	} else {
		// qrSize + colGap + textColWidth = contentWidth  (single block, full width)
		textColWidth = contentWidth - qrSize - colGap
	}
	if textColWidth < 40 {
		textColWidth = 40 // safety floor
	}

	// Compute total group width for centering.
	blockWidth := qrSize + colGap + textColWidth
	var groupWidth float64
	if twoBlocks {
		groupWidth = 2*blockWidth + blockGap
	} else {
		groupWidth = blockWidth
	}
	groupStartX := margin + (contentWidth-groupWidth)/2

	startY := pdf.GetY()
	xCursor := groupStartX

	renderPaymentBlock := func(label, address, uriScheme string) {
		blockX := xCursor

		// Generate QR code PNG in memory.
		qrPNG, err := qrcode.Encode(uriScheme+address, qrcode.Medium, int(qrSize*2))
		if err == nil {
			imgHolder, imgErr := gopdf.ImageHolderByReader(bytes.NewReader(qrPNG))
			if imgErr == nil {
				_ = pdf.ImageByHolder(imgHolder, blockX, startY, &gopdf.Rect{W: qrSize, H: qrSize})
			}
		}

		// Text column to the right of the QR code.
		textX := blockX + qrSize + colGap
		textY := startY

		pdf.SetXY(textX, textY)
		_ = pdf.SetFont("Inter", "", 8)
		setTextColor(pdf, activeTheme.SecondaryText)
		_ = pdf.Cell(nil, label)

		textY += labelExtraH
		_ = pdf.SetFont("Inter", "", fontSize)
		setTextColor(pdf, activeTheme.PrimaryText)

		uri := uriScheme + address

		// Word-wrap the full address within the available column width.
		addrLines := wrapText(pdf, address, textColWidth)
		for _, line := range addrLines {
			pdf.SetXY(textX, textY)
			_ = pdf.Cell(nil, line)
			// Overlay a clickable link on each address line.
			pdf.AddExternalLink(uri, textX, textY-1, textColWidth, lineH)
			textY += lineH
		}

		// Advance x cursor for next block.
		xCursor += blockWidth + blockGap
	}

	if bitcoinAddr != "" {
		renderPaymentBlock("BITCOIN", bitcoinAddr, "bitcoin:")
	}
	if lightningAddr != "" {
		renderPaymentBlock("LIGHTNING", lightningAddr, "lightning:")
	}

	// Move Y past the payment block back to left margin.
	pdf.SetXY(margin, startY+qrSize+12)
}

func writeFooter(pdf *gopdf.GoPdf, id string) {
	pdf.SetY(750)

	_ = pdf.SetFont("Inter", "", 10)
	setTextColor(pdf, activeTheme.Accent)
	_ = pdf.Cell(nil, id)
	setStrokeColor(pdf, activeTheme.Line)
	pdf.Line(pdf.GetX()+10, pdf.GetY()+6, amountRightX, pdf.GetY()+6)
	pdf.Br(48)
}

// --- utilities ---------------------------------------------------------------

func getImageDimension(imagePath string) (int, int) {
	f, err := os.Open(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 0, 0
	}
	defer f.Close()

	img, _, err := image.DecodeConfig(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", imagePath, err)
		return 0, 0
	}
	return img.Width, img.Height
}
