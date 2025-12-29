package db

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// SkillCategory constants for categorizing skills
const (
	SkillCategoryProgramming = "programming"
	SkillCategoryFramework   = "framework"
	SkillCategoryDatabase    = "database"
	SkillCategoryTool        = "tool"
	SkillCategoryCloud       = "cloud"
	SkillCategorySoftSkill   = "soft_skill"
	SkillCategoryOther       = "other"
)

// EvidenceStrength constants
const (
	EvidenceStrengthHigh   = "high"
	EvidenceStrengthMedium = "medium"
	EvidenceStrengthLow    = "low"
)

// Skill represents a normalized skill entry
type Skill struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	NameNormalized string    `json:"name_normalized"`
	Category       *string   `json:"category,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Story represents a group of related experience bullets
type Story struct {
	ID          uuid.UUID `json:"id"`
	StoryID     string    `json:"story_id"` // human-readable ID
	UserID      uuid.UUID `json:"user_id"`
	JobID       uuid.UUID `json:"job_id"`
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Denormalized for convenience (loaded via joins or separate queries)
	Job     *Job     `json:"job,omitempty"`
	Bullets []Bullet `json:"bullets,omitempty"`
}

// Bullet represents an individual experience bullet point
type Bullet struct {
	ID               uuid.UUID   `json:"id"`
	BulletID         string      `json:"bullet_id"` // stable identifier
	StoryID          uuid.UUID   `json:"story_id"`
	JobID            *uuid.UUID  `json:"job_id,omitempty"`
	Text             string      `json:"text"`
	Metrics          *string     `json:"metrics,omitempty"`
	LengthChars      int         `json:"length_chars"`
	EvidenceStrength string      `json:"evidence_strength"`
	RiskFlags        StringArray `json:"risk_flags"`
	Ordinal          int         `json:"ordinal"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`

	// Denormalized skills (loaded via join)
	Skills []string `json:"skills,omitempty"`
}

// BulletSkill represents the many-to-many relationship
type BulletSkill struct {
	BulletID uuid.UUID `json:"bullet_id"`
	SkillID  uuid.UUID `json:"skill_id"`
}

// EducationHighlight represents a notable achievement
type EducationHighlight struct {
	ID          uuid.UUID `json:"id"`
	EducationID uuid.UUID `json:"education_id"`
	Text        string    `json:"text"`
	Ordinal     int       `json:"ordinal"`
	CreatedAt   time.Time `json:"created_at"`
}

// StoryCreateInput is used when importing a story from experience_bank.json
type StoryCreateInput struct {
	StoryID     string // human-readable ID
	UserID      uuid.UUID
	JobID       uuid.UUID
	Title       string
	Description string
	Bullets     []BulletCreateInput
}

// BulletCreateInput is used when creating a bullet
type BulletCreateInput struct {
	BulletID         string
	Text             string
	Metrics          string
	EvidenceStrength string
	RiskFlags        []string
	Skills           []string // skill names to be normalized
	Ordinal          int
}

// ExperienceBankImportInput matches the experience_bank.json structure
type ExperienceBankImportInput struct {
	UserID    uuid.UUID
	Stories   []StoryImportInput
	Education []EducationImportInput
}

// StoryImportInput matches the story structure in experience_bank.json
type StoryImportInput struct {
	ID        string              `json:"id"` // story_id
	Company   string              `json:"company"`
	Role      string              `json:"role"`
	StartDate string              `json:"start_date"` // YYYY-MM
	EndDate   string              `json:"end_date"`   // YYYY-MM or "present"
	Bullets   []BulletImportInput `json:"bullets"`
}

// BulletImportInput matches the bullet structure in experience_bank.json
type BulletImportInput struct {
	ID               string   `json:"id"` // bullet_id
	Text             string   `json:"text"`
	Skills           []string `json:"skills"`
	Metrics          string   `json:"metrics,omitempty"`
	LengthChars      int      `json:"length_chars"`
	EvidenceStrength string   `json:"evidence_strength"`
	RiskFlags        []string `json:"risk_flags"`
}

// EducationImportInput matches the education structure in experience_bank.json
type EducationImportInput struct {
	ID         string   `json:"id"`
	School     string   `json:"school"`
	Degree     string   `json:"degree"` // bachelor, master, phd, etc.
	Field      string   `json:"field"`
	StartDate  string   `json:"start_date,omitempty"`
	EndDate    string   `json:"end_date,omitempty"`
	GPA        string   `json:"gpa,omitempty"`
	Highlights []string `json:"highlights,omitempty"`
}

// skillSynonyms maps common variations to canonical names
var skillSynonyms = map[string]string{
	"golang":                "go",
	"postgresql":            "postgres",
	"javascript":            "js",
	"typescript":            "ts",
	"kubernetes":            "k8s",
	"amazon web services":   "aws",
	"google cloud platform": "gcp",
	"microsoft azure":       "azure",
}

// NormalizeSkillName normalizes a skill name for matching
func NormalizeSkillName(name string) string {
	// Lowercase and trim
	normalized := strings.ToLower(strings.TrimSpace(name))

	// Apply common synonyms
	if canonical, ok := skillSynonyms[normalized]; ok {
		return canonical
	}

	return normalized
}

// ValidEvidenceStrength checks if a strength value is valid
func ValidEvidenceStrength(strength string) bool {
	switch strings.ToLower(strength) {
	case EvidenceStrengthHigh, EvidenceStrengthMedium, EvidenceStrengthLow:
		return true
	default:
		return false
	}
}

// DetectSkillCategory attempts to categorize a skill
func DetectSkillCategory(skillName string) string {
	normalized := strings.ToLower(skillName)

	// Programming languages
	programming := []string{"go", "python", "java", "rust", "c++", "c#", "ruby", "scala", "kotlin", "swift", "js", "ts", "php", "r", "julia", "c", "perl", "haskell", "erlang", "elixir"}
	for _, lang := range programming {
		if normalized == lang {
			return SkillCategoryProgramming
		}
	}

	// Frameworks
	frameworks := []string{"react", "vue", "angular", "django", "flask", "spring", "rails", "express", "fastapi", "gin", "echo", "next", "nuxt", "svelte", "laravel", "asp.net"}
	for _, fw := range frameworks {
		if strings.Contains(normalized, fw) {
			return SkillCategoryFramework
		}
	}

	// Databases
	databases := []string{"postgres", "mysql", "mongodb", "redis", "elasticsearch", "cassandra", "dynamodb", "sqlite", "oracle", "sql server", "mariadb", "cockroachdb", "neo4j"}
	for _, db := range databases {
		if strings.Contains(normalized, db) {
			return SkillCategoryDatabase
		}
	}

	// Cloud
	cloud := []string{"aws", "gcp", "azure", "k8s", "docker", "terraform", "cloudformation", "pulumi", "heroku", "vercel", "netlify"}
	for _, c := range cloud {
		if strings.Contains(normalized, c) {
			return SkillCategoryCloud
		}
	}

	// Tools
	tools := []string{"git", "jenkins", "github", "gitlab", "jira", "confluence", "datadog", "grafana", "prometheus", "splunk", "kibana", "ansible", "chef", "puppet"}
	for _, tool := range tools {
		if strings.Contains(normalized, tool) {
			return SkillCategoryTool
		}
	}

	// Soft skills
	softSkills := []string{"leadership", "communication", "mentoring", "collaboration", "problem-solving", "teamwork", "management", "agile", "scrum"}
	for _, ss := range softSkills {
		if strings.Contains(normalized, ss) {
			return SkillCategorySoftSkill
		}
	}

	return SkillCategoryOther
}
