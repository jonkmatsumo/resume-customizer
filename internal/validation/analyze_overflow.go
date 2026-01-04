// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"math"

	"github.com/jonathan/resume-customizer/internal/types"
)

const (
	// linesPerPage is the estimated number of content lines per page
	linesPerPage = 50
	// charsPerLine is the estimated number of characters per line
	charsPerLine = 100
)

// OverflowAnalysis contains the results of analyzing page overflow
type OverflowAnalysis struct {
	ExcessPages   float64 // How many pages over (e.g., 0.5 = half a page)
	ExcessLines   int     // Estimated lines that need to be removed
	ExcessBullets float64 // Estimated bullets that need to be removed
	CanShorten    bool    // Can we fix by shortening bullets?
	MustDrop      bool    // Must we drop bullets?
}

// AnalyzePageOverflow calculates how many bullets/lines need to be removed
// to fit within the page limit.
func AnalyzePageOverflow(
	currentPages int,
	maxPages int,
	bullets *types.RewrittenBullets,
	_ *types.ResumePlan, // Reserved for future use (e.g., section-level analysis)
) *OverflowAnalysis {
	analysis := &OverflowAnalysis{}

	// If no overflow, return empty analysis
	if currentPages <= maxPages {
		return analysis
	}

	// Calculate excess pages
	analysis.ExcessPages = float64(currentPages - maxPages)

	// Calculate excess lines (using linesPerPage estimate)
	analysis.ExcessLines = int(analysis.ExcessPages * linesPerPage)

	// Calculate average lines per bullet
	avgLinesPerBullet := calculateAverageLinesPerBullet(bullets)
	if avgLinesPerBullet <= 0 {
		avgLinesPerBullet = 2.0 // Default assumption: 2 lines per bullet
	}

	// Calculate how many bullets need to be removed
	analysis.ExcessBullets = float64(analysis.ExcessLines) / avgLinesPerBullet

	// Determine strategy: can we shorten, or must we drop?
	// If we need to remove less than 1 bullet's worth, we can probably shorten
	// If we need to remove 1 or more bullets, we must drop
	analysis.MustDrop = analysis.ExcessBullets >= 1.0
	analysis.CanShorten = analysis.ExcessBullets < 1.0

	return analysis
}

// calculateAverageLinesPerBullet computes the average lines per bullet from rewritten bullets
func calculateAverageLinesPerBullet(bullets *types.RewrittenBullets) float64 {
	if bullets == nil || len(bullets.Bullets) == 0 {
		return 0
	}

	totalLines := 0
	for _, bullet := range bullets.Bullets {
		if bullet.EstimatedLines > 0 {
			totalLines += bullet.EstimatedLines
		} else {
			// Estimate from length if EstimatedLines not set
			totalLines += int(math.Ceil(float64(bullet.LengthChars) / charsPerLine))
		}
	}

	return float64(totalLines) / float64(len(bullets.Bullets))
}

// BulletsToDropCount returns the number of bullets that should be dropped
// to resolve the overflow. Returns 0 if no drops are needed.
func (a *OverflowAnalysis) BulletsToDropCount() int {
	if !a.MustDrop {
		return 0
	}
	return int(math.Ceil(a.ExcessBullets))
}
