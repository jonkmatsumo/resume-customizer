package db

import (
	"testing"

	"github.com/google/uuid"
)

// =============================================================================
// Constant Tests
// =============================================================================

func TestSeverityConstants(t *testing.T) {
	if SeverityError != "error" {
		t.Errorf("SeverityError = %q, want 'error'", SeverityError)
	}
	if SeverityWarning != "warning" {
		t.Errorf("SeverityWarning = %q, want 'warning'", SeverityWarning)
	}
}

func TestViolationTypeConstants(t *testing.T) {
	types := []string{
		ViolationPageOverflow,
		ViolationLineTooLong,
		ViolationMissingSkill,
		ViolationTooFewBullets,
		ViolationTooManyPages,
	}

	for _, vt := range types {
		if vt == "" {
			t.Error("Violation type constant should not be empty")
		}
	}

	// Verify expected values
	if ViolationPageOverflow != "page_overflow" {
		t.Errorf("ViolationPageOverflow = %q, want 'page_overflow'", ViolationPageOverflow)
	}
	if ViolationLineTooLong != "line_too_long" {
		t.Errorf("ViolationLineTooLong = %q, want 'line_too_long'", ViolationLineTooLong)
	}
}

func TestSectionConstants(t *testing.T) {
	sections := []string{
		SectionExperience,
		SectionProjects,
		SectionEducation,
		SectionSkills,
	}

	for _, s := range sections {
		if s == "" {
			t.Error("Section constant should not be empty")
		}
	}

	// Verify expected values
	if SectionExperience != "experience" {
		t.Errorf("SectionExperience = %q, want 'experience'", SectionExperience)
	}
	if SectionProjects != "projects" {
		t.Errorf("SectionProjects = %q, want 'projects'", SectionProjects)
	}
}

func TestValidSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"error", true},
		{"warning", true},
		{"info", false},
		{"", false},
		{"ERROR", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ValidSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("ValidSeverity(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Type Tests
// =============================================================================

func TestRunRankedStoryType(t *testing.T) {
	score := 0.85
	rs := RunRankedStory{
		ID:             uuid.New(),
		RunID:          uuid.New(),
		StoryIDText:    "amazon-analytics",
		RelevanceScore: &score,
		MatchedSkills:  []string{"Go", "AWS"},
		Ordinal:        1,
	}

	if rs.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if rs.StoryIDText != "amazon-analytics" {
		t.Errorf("StoryIDText = %q", rs.StoryIDText)
	}
	if *rs.RelevanceScore != 0.85 {
		t.Errorf("RelevanceScore = %f", *rs.RelevanceScore)
	}
	if len(rs.MatchedSkills) != 2 {
		t.Errorf("MatchedSkills count = %d, want 2", len(rs.MatchedSkills))
	}
}

func TestRunResumePlanType(t *testing.T) {
	plan := RunResumePlan{
		ID:               uuid.New(),
		RunID:            uuid.New(),
		MaxBullets:       8,
		MaxLines:         40,
		SkillMatchRatio:  0.7,
		SectionBudgets:   map[string]int{"experience": 30, "projects": 10},
		TopSkillsCovered: []string{"Go", "Python", "AWS"},
		CoverageScore:    0.85,
	}

	if plan.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if plan.MaxBullets != 8 {
		t.Errorf("MaxBullets = %d, want 8", plan.MaxBullets)
	}
	if plan.SectionBudgets["experience"] != 30 {
		t.Errorf("SectionBudgets[experience] = %d, want 30", plan.SectionBudgets["experience"])
	}
	if len(plan.TopSkillsCovered) != 3 {
		t.Errorf("TopSkillsCovered count = %d, want 3", len(plan.TopSkillsCovered))
	}
}

func TestRunSelectedBulletType(t *testing.T) {
	metrics := "40% improvement"
	sb := RunSelectedBullet{
		ID:           uuid.New(),
		RunID:        uuid.New(),
		BulletIDText: "bullet_001",
		StoryIDText:  "amazon-analytics",
		Text:         "Improved system performance by 40%",
		Skills:       []string{"Go", "Performance"},
		Metrics:      &metrics,
		LengthChars:  35,
		Section:      SectionExperience,
		Ordinal:      1,
	}

	if sb.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if sb.BulletIDText != "bullet_001" {
		t.Errorf("BulletIDText = %q", sb.BulletIDText)
	}
	if *sb.Metrics != "40% improvement" {
		t.Errorf("Metrics = %q", *sb.Metrics)
	}
	if sb.Section != SectionExperience {
		t.Errorf("Section = %q", sb.Section)
	}
}

func TestRunRewrittenBulletType(t *testing.T) {
	rb := RunRewrittenBullet{
		ID:                   uuid.New(),
		RunID:                uuid.New(),
		OriginalBulletIDText: "bullet_001",
		FinalText:            "Engineered high-performance system achieving 40% latency reduction",
		LengthChars:          65,
		EstimatedLines:       1,
		StyleStrongVerb:      true,
		StyleQuantified:      true,
		StyleNoTaboo:         true,
		StyleTargetLength:    true,
		Ordinal:              1,
	}

	if rb.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if rb.OriginalBulletIDText != "bullet_001" {
		t.Errorf("OriginalBulletIDText = %q", rb.OriginalBulletIDText)
	}
	if !rb.StyleStrongVerb {
		t.Error("StyleStrongVerb should be true")
	}
	if !rb.StyleQuantified {
		t.Error("StyleQuantified should be true")
	}
}

func TestRunViolationType(t *testing.T) {
	lineNum := 42
	charCount := 150
	rv := RunViolation{
		ID:               uuid.New(),
		RunID:            uuid.New(),
		ViolationType:    ViolationLineTooLong,
		Severity:         SeverityError,
		LineNumber:       &lineNum,
		CharCount:        &charCount,
		AffectedSections: []string{"experience"},
	}

	if rv.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if rv.ViolationType != ViolationLineTooLong {
		t.Errorf("ViolationType = %q", rv.ViolationType)
	}
	if rv.Severity != SeverityError {
		t.Errorf("Severity = %q", rv.Severity)
	}
	if *rv.LineNumber != 42 {
		t.Errorf("LineNumber = %d, want 42", *rv.LineNumber)
	}
}

// =============================================================================
// Input Type Tests
// =============================================================================

func TestRunRankedStoryInput(t *testing.T) {
	llmScore := 0.9
	input := RunRankedStoryInput{
		StoryIDText:      "google-ml-platform",
		RelevanceScore:   0.85,
		SkillOverlap:     0.7,
		KeywordOverlap:   0.6,
		EvidenceStrength: 0.9,
		HeuristicScore:   0.75,
		LLMScore:         &llmScore,
		LLMReasoning:     "Strong match for ML requirements",
		MatchedSkills:    []string{"Python", "TensorFlow"},
		Notes:            "Top candidate",
		Ordinal:          1,
	}

	if input.StoryIDText != "google-ml-platform" {
		t.Errorf("StoryIDText = %q", input.StoryIDText)
	}
	if input.RelevanceScore != 0.85 {
		t.Errorf("RelevanceScore = %f", input.RelevanceScore)
	}
	if *input.LLMScore != 0.9 {
		t.Errorf("LLMScore = %f", *input.LLMScore)
	}
}

func TestRunResumePlanInput(t *testing.T) {
	input := RunResumePlanInput{
		MaxBullets:       8,
		MaxLines:         40,
		SkillMatchRatio:  0.7,
		SectionBudgets:   map[string]int{"experience": 30, "projects": 10},
		TopSkillsCovered: []string{"Go", "Python"},
		CoverageScore:    0.85,
	}

	if input.MaxBullets != 8 {
		t.Errorf("MaxBullets = %d", input.MaxBullets)
	}
	if input.CoverageScore != 0.85 {
		t.Errorf("CoverageScore = %f", input.CoverageScore)
	}
}

func TestRunSelectedBulletInput(t *testing.T) {
	input := RunSelectedBulletInput{
		BulletIDText: "bullet_001",
		StoryIDText:  "amazon-analytics",
		Text:         "Built distributed system",
		Skills:       []string{"Go", "AWS"},
		Metrics:      "1M requests/day",
		LengthChars:  25,
		Section:      SectionExperience,
		Ordinal:      1,
	}

	if input.BulletIDText != "bullet_001" {
		t.Errorf("BulletIDText = %q", input.BulletIDText)
	}
	if input.Section != SectionExperience {
		t.Errorf("Section = %q", input.Section)
	}
}

func TestRunRewrittenBulletInput(t *testing.T) {
	input := RunRewrittenBulletInput{
		OriginalBulletIDText: "bullet_001",
		FinalText:            "Architected distributed system processing 1M requests daily",
		LengthChars:          58,
		EstimatedLines:       1,
		StyleStrongVerb:      true,
		StyleQuantified:      true,
		StyleNoTaboo:         true,
		StyleTargetLength:    true,
		Ordinal:              1,
	}

	if input.OriginalBulletIDText != "bullet_001" {
		t.Errorf("OriginalBulletIDText = %q", input.OriginalBulletIDText)
	}
	if input.FinalText == "" {
		t.Error("FinalText should not be empty")
	}
	if !input.StyleStrongVerb {
		t.Error("StyleStrongVerb should be true")
	}
}

func TestRunViolationInput(t *testing.T) {
	lineNum := 42
	charCount := 150
	input := RunViolationInput{
		ViolationType:    ViolationLineTooLong,
		Severity:         SeverityError,
		Details:          "Line exceeds 120 characters",
		LineNumber:       &lineNum,
		CharCount:        &charCount,
		AffectedSections: []string{"experience"},
	}

	if input.ViolationType != ViolationLineTooLong {
		t.Errorf("ViolationType = %q", input.ViolationType)
	}
	if input.Severity != SeverityError {
		t.Errorf("Severity = %q", input.Severity)
	}
	if *input.LineNumber != 42 {
		t.Errorf("LineNumber = %d", *input.LineNumber)
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestRunRankedStoryNilOptionalFields(t *testing.T) {
	rs := RunRankedStory{
		ID:          uuid.New(),
		RunID:       uuid.New(),
		StoryIDText: "test-story",
		Ordinal:     1,
		// All optional fields nil
	}

	if rs.StoryID != nil {
		t.Error("StoryID should be nil")
	}
	if rs.RelevanceScore != nil {
		t.Error("RelevanceScore should be nil")
	}
	if rs.LLMScore != nil {
		t.Error("LLMScore should be nil")
	}
}

func TestRunViolationNilOptionalFields(t *testing.T) {
	rv := RunViolation{
		ID:            uuid.New(),
		RunID:         uuid.New(),
		ViolationType: ViolationPageOverflow,
		Severity:      SeverityWarning,
		// All optional fields nil
	}

	if rv.Details != nil {
		t.Error("Details should be nil")
	}
	if rv.LineNumber != nil {
		t.Error("LineNumber should be nil")
	}
	if rv.CharCount != nil {
		t.Error("CharCount should be nil")
	}
}

func TestEmptySliceFields(t *testing.T) {
	rs := RunRankedStory{
		ID:            uuid.New(),
		RunID:         uuid.New(),
		StoryIDText:   "test",
		MatchedSkills: []string{}, // empty, not nil
		Ordinal:       1,
	}

	if rs.MatchedSkills == nil {
		t.Error("MatchedSkills should be empty slice, not nil")
	}
	if len(rs.MatchedSkills) != 0 {
		t.Errorf("MatchedSkills should be empty, got %d", len(rs.MatchedSkills))
	}
}
