package db

import (
	"testing"

	"github.com/google/uuid"
)

// =============================================================================
// Skill Normalization Tests
// =============================================================================

func TestNormalizeSkillName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Go", "go"},
		{"Golang", "go"},
		{"golang", "go"},
		{"Python", "python"},
		{"PostgreSQL", "postgres"},
		{"postgresql", "postgres"},
		{"Kubernetes", "k8s"},
		{"kubernetes", "k8s"},
		{"Amazon Web Services", "aws"},
		{"JavaScript", "js"},
		{"TypeScript", "ts"},
		{"  spaces  ", "spaces"},
		{"", ""},
		{"React", "react"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeSkillName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSkillName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeSkillName_Consistency(t *testing.T) {
	// Same skill with different cases should normalize to the same value
	variations := []string{"Go", "go", "GO", "gO"}
	expected := "go"

	for _, v := range variations {
		result := NormalizeSkillName(v)
		if result != expected {
			t.Errorf("NormalizeSkillName(%q) = %q, want %q", v, result, expected)
		}
	}
}

func TestValidEvidenceStrength(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"high", true},
		{"medium", true},
		{"low", true},
		{"HIGH", true},
		{"Medium", true},
		{"LOW", true},
		{"invalid", false},
		{"", false},
		{"very_high", false},
		{"none", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ValidEvidenceStrength(tt.input)
			if result != tt.expected {
				t.Errorf("ValidEvidenceStrength(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectSkillCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Programming languages
		{"go", SkillCategoryProgramming},
		{"python", SkillCategoryProgramming},
		{"java", SkillCategoryProgramming},
		{"rust", SkillCategoryProgramming},
		{"js", SkillCategoryProgramming},
		{"ts", SkillCategoryProgramming},

		// Frameworks
		{"react", SkillCategoryFramework},
		{"django", SkillCategoryFramework},
		{"spring boot", SkillCategoryFramework},
		{"fastapi", SkillCategoryFramework},

		// Databases
		{"postgres", SkillCategoryDatabase},
		{"mongodb", SkillCategoryDatabase},
		{"redis", SkillCategoryDatabase},
		{"mysql", SkillCategoryDatabase},

		// Cloud
		{"aws", SkillCategoryCloud},
		{"gcp", SkillCategoryCloud},
		{"k8s", SkillCategoryCloud},
		{"docker", SkillCategoryCloud},
		{"terraform", SkillCategoryCloud},

		// Tools
		{"git", SkillCategoryTool},
		{"jenkins", SkillCategoryTool},
		{"jira", SkillCategoryTool},
		{"datadog", SkillCategoryTool},

		// Soft skills
		{"leadership", SkillCategorySoftSkill},
		{"communication", SkillCategorySoftSkill},
		{"teamwork", SkillCategorySoftSkill},

		// Other
		{"unknown-skill", SkillCategoryOther},
		{"custom framework", SkillCategoryOther},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := DetectSkillCategory(tt.input)
			if result != tt.expected {
				t.Errorf("DetectSkillCategory(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Constant Tests
// =============================================================================

func TestSkillCategoryConstants(t *testing.T) {
	categories := []string{
		SkillCategoryProgramming,
		SkillCategoryFramework,
		SkillCategoryDatabase,
		SkillCategoryTool,
		SkillCategoryCloud,
		SkillCategorySoftSkill,
		SkillCategoryOther,
	}

	for _, c := range categories {
		if c == "" {
			t.Error("Skill category constant should not be empty")
		}
	}

	// Verify expected values
	if SkillCategoryProgramming != "programming" {
		t.Errorf("SkillCategoryProgramming = %q, want 'programming'", SkillCategoryProgramming)
	}
	if SkillCategoryFramework != "framework" {
		t.Errorf("SkillCategoryFramework = %q, want 'framework'", SkillCategoryFramework)
	}
}

func TestEvidenceStrengthConstants(t *testing.T) {
	if EvidenceStrengthHigh != "high" {
		t.Errorf("EvidenceStrengthHigh = %q, want 'high'", EvidenceStrengthHigh)
	}
	if EvidenceStrengthMedium != "medium" {
		t.Errorf("EvidenceStrengthMedium = %q, want 'medium'", EvidenceStrengthMedium)
	}
	if EvidenceStrengthLow != "low" {
		t.Errorf("EvidenceStrengthLow = %q, want 'low'", EvidenceStrengthLow)
	}
}

// =============================================================================
// Type Tests
// =============================================================================

func TestSkillType(t *testing.T) {
	category := "programming"
	skill := Skill{
		ID:             uuid.New(),
		Name:           "Go",
		NameNormalized: "go",
		Category:       &category,
	}

	if skill.ID == uuid.Nil {
		t.Error("Skill ID should be set")
	}
	if skill.Name != "Go" {
		t.Errorf("Skill Name = %q, want 'Go'", skill.Name)
	}
	if skill.NameNormalized != "go" {
		t.Errorf("Skill NameNormalized = %q, want 'go'", skill.NameNormalized)
	}
	if *skill.Category != "programming" {
		t.Errorf("Skill Category = %q, want 'programming'", *skill.Category)
	}
}

func TestStoryType(t *testing.T) {
	title := "Analytics Platform"
	story := Story{
		ID:      uuid.New(),
		StoryID: "amazon-analytics",
		UserID:  uuid.New(),
		JobID:   uuid.New(),
		Title:   &title,
		Bullets: []Bullet{
			{
				ID:               uuid.New(),
				BulletID:         "bullet_001",
				Text:             "Built distributed system",
				EvidenceStrength: EvidenceStrengthHigh,
				Skills:           []string{"Go", "Distributed Systems"},
			},
		},
	}

	if story.ID == uuid.Nil {
		t.Error("Story ID should be set")
	}
	if story.StoryID != "amazon-analytics" {
		t.Errorf("Story StoryID = %q", story.StoryID)
	}
	if *story.Title != "Analytics Platform" {
		t.Errorf("Story Title = %q", *story.Title)
	}
	if len(story.Bullets) != 1 {
		t.Errorf("Story Bullets count = %d, want 1", len(story.Bullets))
	}
}

func TestBulletType(t *testing.T) {
	metrics := "1M requests/day"
	bullet := Bullet{
		ID:               uuid.New(),
		BulletID:         "bullet_001",
		StoryID:          uuid.New(),
		Text:             "Processed 1M requests/day",
		Metrics:          &metrics,
		LengthChars:      30,
		EvidenceStrength: EvidenceStrengthHigh,
		RiskFlags:        StringArray{},
		Ordinal:          1,
		Skills:           []string{"Go", "Distributed Systems"},
	}

	if bullet.ID == uuid.Nil {
		t.Error("Bullet ID should be set")
	}
	if bullet.BulletID != "bullet_001" {
		t.Errorf("Bullet BulletID = %q", bullet.BulletID)
	}
	if *bullet.Metrics != "1M requests/day" {
		t.Errorf("Bullet Metrics = %q", *bullet.Metrics)
	}
	if bullet.EvidenceStrength != EvidenceStrengthHigh {
		t.Errorf("Bullet EvidenceStrength = %q", bullet.EvidenceStrength)
	}
	if len(bullet.Skills) != 2 {
		t.Errorf("Bullet Skills count = %d, want 2", len(bullet.Skills))
	}
}

func TestEducationHighlightType(t *testing.T) {
	highlight := EducationHighlight{
		ID:          uuid.New(),
		EducationID: uuid.New(),
		Text:        "Dean's List",
		Ordinal:     1,
	}

	if highlight.ID == uuid.Nil {
		t.Error("Highlight ID should be set")
	}
	if highlight.Text != "Dean's List" {
		t.Errorf("Highlight Text = %q", highlight.Text)
	}
	if highlight.Ordinal != 1 {
		t.Errorf("Highlight Ordinal = %d, want 1", highlight.Ordinal)
	}
}

// =============================================================================
// Input Type Tests
// =============================================================================

func TestStoryCreateInput(t *testing.T) {
	input := StoryCreateInput{
		StoryID:     "amazon-analytics-project",
		UserID:      uuid.New(),
		JobID:       uuid.New(),
		Title:       "Analytics Platform",
		Description: "Built analytics platform",
		Bullets: []BulletCreateInput{
			{
				BulletID:         "bullet_001",
				Text:             "Built a distributed system",
				EvidenceStrength: EvidenceStrengthHigh,
				Skills:           []string{"Go", "Distributed Systems"},
			},
		},
	}

	if input.StoryID != "amazon-analytics-project" {
		t.Errorf("StoryID = %q", input.StoryID)
	}
	if input.Title != "Analytics Platform" {
		t.Errorf("Title = %q", input.Title)
	}
	if len(input.Bullets) != 1 {
		t.Errorf("Bullets count = %d, want 1", len(input.Bullets))
	}
}

func TestBulletCreateInput(t *testing.T) {
	input := BulletCreateInput{
		BulletID:         "bullet_001",
		Text:             "Improved latency by 40%",
		Metrics:          "40% improvement",
		EvidenceStrength: EvidenceStrengthHigh,
		RiskFlags:        []string{},
		Skills:           []string{"Go", "Performance"},
		Ordinal:          1,
	}

	if input.BulletID != "bullet_001" {
		t.Errorf("BulletID = %q", input.BulletID)
	}
	if input.Text != "Improved latency by 40%" {
		t.Errorf("Text = %q", input.Text)
	}
	if len(input.Skills) != 2 {
		t.Errorf("Skills count = %d, want 2", len(input.Skills))
	}
}

func TestBulletImportInput(t *testing.T) {
	input := BulletImportInput{
		ID:               "bullet_001",
		Text:             "Improved latency by 40%",
		Skills:           []string{"Go", "Performance"},
		Metrics:          "40% improvement",
		LengthChars:      25,
		EvidenceStrength: "high",
		RiskFlags:        []string{},
	}

	if input.ID != "bullet_001" {
		t.Errorf("ID = %q, want 'bullet_001'", input.ID)
	}
	if len(input.Skills) != 2 {
		t.Errorf("Skills count = %d, want 2", len(input.Skills))
	}
	if input.LengthChars != 25 {
		t.Errorf("LengthChars = %d, want 25", input.LengthChars)
	}
}

func TestStoryImportInput(t *testing.T) {
	input := StoryImportInput{
		ID:        "story_001",
		Company:   "Test Corp",
		Role:      "Software Engineer",
		StartDate: "2020-01",
		EndDate:   "2023-06",
		Bullets: []BulletImportInput{
			{
				ID:               "bullet_001",
				Text:             "Test bullet",
				Skills:           []string{"Go"},
				LengthChars:      11,
				EvidenceStrength: "medium",
				RiskFlags:        []string{},
			},
		},
	}

	if input.ID != "story_001" {
		t.Errorf("ID = %q", input.ID)
	}
	if input.Company != "Test Corp" {
		t.Errorf("Company = %q", input.Company)
	}
	if input.StartDate != "2020-01" {
		t.Errorf("StartDate = %q", input.StartDate)
	}
	if input.EndDate != "2023-06" {
		t.Errorf("EndDate = %q", input.EndDate)
	}
	if len(input.Bullets) != 1 {
		t.Errorf("Bullets count = %d, want 1", len(input.Bullets))
	}
}

func TestEducationImportInput(t *testing.T) {
	input := EducationImportInput{
		ID:         "edu_001",
		School:     "Test University",
		Degree:     "bachelor",
		Field:      "Computer Science",
		StartDate:  "2015-09",
		EndDate:    "2019-05",
		GPA:        "3.8",
		Highlights: []string{"Dean's List", "Summa Cum Laude"},
	}

	if input.ID != "edu_001" {
		t.Errorf("ID = %q", input.ID)
	}
	if input.School != "Test University" {
		t.Errorf("School = %q", input.School)
	}
	if input.Degree != "bachelor" {
		t.Errorf("Degree = %q", input.Degree)
	}
	if len(input.Highlights) != 2 {
		t.Errorf("Highlights count = %d, want 2", len(input.Highlights))
	}
}

func TestExperienceBankImportInput(t *testing.T) {
	input := ExperienceBankImportInput{
		UserID: uuid.New(),
		Stories: []StoryImportInput{
			{
				ID:        "story_001",
				Company:   "Test Corp",
				Role:      "Engineer",
				StartDate: "2020-01",
				EndDate:   "2023-06",
				Bullets:   []BulletImportInput{},
			},
		},
		Education: []EducationImportInput{
			{
				ID:     "edu_001",
				School: "Test University",
				Degree: "bachelor",
				Field:  "Computer Science",
			},
		},
	}

	if input.UserID == uuid.Nil {
		t.Error("UserID should be set")
	}
	if len(input.Stories) != 1 {
		t.Errorf("Stories count = %d, want 1", len(input.Stories))
	}
	if len(input.Education) != 1 {
		t.Errorf("Education count = %d, want 1", len(input.Education))
	}
}

// =============================================================================
// Skill Synonym Tests
// =============================================================================

func TestSkillSynonyms(t *testing.T) {
	synonymTests := []struct {
		input    string
		expected string
	}{
		{"Golang", "go"},
		{"golang", "go"},
		{"PostgreSQL", "postgres"},
		{"postgresql", "postgres"},
		{"JavaScript", "js"},
		{"javascript", "js"},
		{"TypeScript", "ts"},
		{"typescript", "ts"},
		{"Kubernetes", "k8s"},
		{"kubernetes", "k8s"},
		{"Amazon Web Services", "aws"},
		{"amazon web services", "aws"},
		{"Google Cloud Platform", "gcp"},
		{"Microsoft Azure", "azure"},
	}

	for _, tt := range synonymTests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeSkillName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSkillName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
