// Package validation provides functionality to validate LaTeX resumes against constraints.
package validation

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestAnalyzePageOverflow_NoOverflow(t *testing.T) {
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "b1", EstimatedLines: 2},
			{OriginalBulletID: "b2", EstimatedLines: 2},
		},
	}
	plan := &types.ResumePlan{}

	analysis := AnalyzePageOverflow(1, 1, bullets, plan)

	assert.Equal(t, 0.0, analysis.ExcessPages)
	assert.Equal(t, 0, analysis.ExcessLines)
	assert.Equal(t, 0.0, analysis.ExcessBullets)
	assert.False(t, analysis.MustDrop)
	assert.False(t, analysis.CanShorten)
}

func TestAnalyzePageOverflow_UnderLimit(t *testing.T) {
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "b1", EstimatedLines: 2},
		},
	}
	plan := &types.ResumePlan{}

	analysis := AnalyzePageOverflow(1, 2, bullets, plan)

	assert.Equal(t, 0.0, analysis.ExcessPages)
	assert.False(t, analysis.MustDrop)
	assert.False(t, analysis.CanShorten)
}

func TestAnalyzePageOverflow_HalfPageOver(t *testing.T) {
	// Create bullets with known line counts
	// Average of 2 lines per bullet
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "b1", EstimatedLines: 2},
			{OriginalBulletID: "b2", EstimatedLines: 2},
			{OriginalBulletID: "b3", EstimatedLines: 2},
			{OriginalBulletID: "b4", EstimatedLines: 2},
		},
	}
	plan := &types.ResumePlan{}

	// Simulate 2 pages when max is 1 (but in reality we're testing the logic)
	// With 50 lines per page and avg 2 lines per bullet:
	// ExcessLines = 50, ExcessBullets = 50/2 = 25
	analysis := AnalyzePageOverflow(2, 1, bullets, plan)

	assert.Equal(t, 1.0, analysis.ExcessPages)
	assert.Equal(t, 50, analysis.ExcessLines)
	assert.True(t, analysis.MustDrop)
	assert.False(t, analysis.CanShorten)
}

func TestAnalyzePageOverflow_FullPageOver(t *testing.T) {
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "b1", EstimatedLines: 2},
			{OriginalBulletID: "b2", EstimatedLines: 2},
		},
	}
	plan := &types.ResumePlan{}

	analysis := AnalyzePageOverflow(3, 1, bullets, plan)

	assert.Equal(t, 2.0, analysis.ExcessPages)
	assert.Equal(t, 100, analysis.ExcessLines)
	assert.True(t, analysis.MustDrop)
	assert.False(t, analysis.CanShorten)
}

func TestAnalyzePageOverflow_EmptyBullets(t *testing.T) {
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{},
	}
	plan := &types.ResumePlan{}

	// Should use default 2 lines per bullet
	analysis := AnalyzePageOverflow(2, 1, bullets, plan)

	assert.Equal(t, 1.0, analysis.ExcessPages)
	assert.Equal(t, 50, analysis.ExcessLines)
	// With default 2 lines per bullet: 50/2 = 25
	assert.Equal(t, 25.0, analysis.ExcessBullets)
	assert.True(t, analysis.MustDrop)
}

func TestAnalyzePageOverflow_NilBullets(t *testing.T) {
	plan := &types.ResumePlan{}

	analysis := AnalyzePageOverflow(2, 1, nil, plan)

	assert.Equal(t, 1.0, analysis.ExcessPages)
	assert.True(t, analysis.MustDrop)
}

func TestAnalyzePageOverflow_CalculatesFromLengthChars(t *testing.T) {
	// Bullets with LengthChars but no EstimatedLines
	// 200 chars / 100 chars per line = 2 lines per bullet
	bullets := &types.RewrittenBullets{
		Bullets: []types.RewrittenBullet{
			{OriginalBulletID: "b1", LengthChars: 200, EstimatedLines: 0},
			{OriginalBulletID: "b2", LengthChars: 200, EstimatedLines: 0},
		},
	}
	plan := &types.ResumePlan{}

	analysis := AnalyzePageOverflow(2, 1, bullets, plan)

	// Avg lines per bullet = 2
	// ExcessLines = 50
	// ExcessBullets = 50/2 = 25
	assert.Equal(t, 25.0, analysis.ExcessBullets)
}

func TestBulletsToDropCount_NoDropNeeded(t *testing.T) {
	analysis := &OverflowAnalysis{
		ExcessBullets: 0.5,
		MustDrop:      false,
	}

	assert.Equal(t, 0, analysis.BulletsToDropCount())
}

func TestBulletsToDropCount_DropOne(t *testing.T) {
	analysis := &OverflowAnalysis{
		ExcessBullets: 1.0,
		MustDrop:      true,
	}

	assert.Equal(t, 1, analysis.BulletsToDropCount())
}

func TestBulletsToDropCount_DropMultiple(t *testing.T) {
	analysis := &OverflowAnalysis{
		ExcessBullets: 2.5,
		MustDrop:      true,
	}

	// Ceil(2.5) = 3
	assert.Equal(t, 3, analysis.BulletsToDropCount())
}

func TestCalculateAverageLinesPerBullet(t *testing.T) {
	tests := []struct {
		name     string
		bullets  *types.RewrittenBullets
		expected float64
	}{
		{
			name:     "nil bullets",
			bullets:  nil,
			expected: 0,
		},
		{
			name:     "empty bullets",
			bullets:  &types.RewrittenBullets{Bullets: []types.RewrittenBullet{}},
			expected: 0,
		},
		{
			name: "single bullet",
			bullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{OriginalBulletID: "b1", EstimatedLines: 3},
				},
			},
			expected: 3.0,
		},
		{
			name: "multiple bullets",
			bullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{OriginalBulletID: "b1", EstimatedLines: 2},
					{OriginalBulletID: "b2", EstimatedLines: 4},
				},
			},
			expected: 3.0, // (2+4)/2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAverageLinesPerBullet(tt.bullets)
			assert.Equal(t, tt.expected, result)
		})
	}
}
