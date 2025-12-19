package experience

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadExperienceBank_ValidFile(t *testing.T) {
	// Use existing valid test fixture
	path := filepath.Join("..", "..", "testdata", "valid", "experience_bank.json")

	bank, err := LoadExperienceBank(path)
	require.NoError(t, err)
	require.NotNil(t, bank)
	require.Len(t, bank.Stories, 1)

	story := bank.Stories[0]
	assert.Equal(t, "story_001", story.ID)
	assert.Equal(t, "Previous Company", story.Company)
	assert.Equal(t, "Software Engineer", story.Role)
	assert.Equal(t, "2020-01", story.StartDate)
	assert.Equal(t, "2023-06", story.EndDate)
	require.Len(t, story.Bullets, 2)

	bullet1 := story.Bullets[0]
	assert.Equal(t, "bullet_001", bullet1.ID)
	assert.Equal(t, "Built distributed system processing 1M requests/day", bullet1.Text)
	assert.Equal(t, []string{"Go", "Distributed Systems"}, bullet1.Skills)
	assert.Equal(t, "1M requests/day", bullet1.Metrics)
	assert.Equal(t, 45, bullet1.LengthChars)
	assert.Equal(t, "high", bullet1.EvidenceStrength)
}

func TestLoadExperienceBank_FileNotFound(t *testing.T) {
	_, err := LoadExperienceBank("nonexistent_file.json")
	require.Error(t, err)

	loadErr, ok := err.(*LoadError)
	require.True(t, ok, "error should be LoadError type")
	assert.Contains(t, loadErr.Error(), "failed to read file")
}

func TestLoadExperienceBank_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	invalidJSON := filepath.Join(tmpDir, "invalid.json")
	err := os.WriteFile(invalidJSON, []byte("{ invalid json }"), 0644)
	require.NoError(t, err)

	_, err = LoadExperienceBank(invalidJSON)
	require.Error(t, err)

	loadErr, ok := err.(*LoadError)
	require.True(t, ok, "error should be LoadError type")
	assert.Contains(t, loadErr.Error(), "schema validation failed")
}

func TestLoadExperienceBank_SchemaValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	invalidJSON := filepath.Join(tmpDir, "invalid_schema.json")
	// Valid JSON but doesn't match schema (missing required field)
	invalidContent := `{
		"stories": [
			{
				"id": "story_001",
				"company": "Test Company"
			}
		]
	}`
	err := os.WriteFile(invalidJSON, []byte(invalidContent), 0644)
	require.NoError(t, err)

	_, err = LoadExperienceBank(invalidJSON)
	require.Error(t, err)

	loadErr, ok := err.(*LoadError)
	require.True(t, ok, "error should be LoadError type")
	assert.Contains(t, loadErr.Error(), "schema validation failed")
}

func TestLoadExperienceBank_EmptyStories(t *testing.T) {
	tmpDir := t.TempDir()
	emptyBank := filepath.Join(tmpDir, "empty.json")
	validContent := `{"stories": []}`
	err := os.WriteFile(emptyBank, []byte(validContent), 0644)
	require.NoError(t, err)

	bank, err := LoadExperienceBank(emptyBank)
	require.NoError(t, err)
	require.NotNil(t, bank)
	assert.Len(t, bank.Stories, 0)
}

func TestLoadExperienceBank_ComplexStructure(t *testing.T) {
	tmpDir := t.TempDir()
	complexBank := filepath.Join(tmpDir, "complex.json")
	validContent := `{
		"stories": [
			{
				"id": "story_001",
				"company": "Company A",
				"role": "Engineer",
				"start_date": "2020-01",
				"end_date": "present",
				"bullets": [
					{
						"id": "bullet_001",
						"text": "Built system",
						"skills": ["Go", "Python"],
						"length_chars": 12,
						"evidence_strength": "high",
						"risk_flags": ["flag1", "flag2"]
					},
					{
						"id": "bullet_002",
						"text": "Optimized performance",
						"skills": ["JavaScript"],
						"metrics": "50% improvement",
						"length_chars": 22,
						"evidence_strength": "medium",
						"risk_flags": []
					}
				]
			}
		]
	}`
	err := os.WriteFile(complexBank, []byte(validContent), 0644)
	require.NoError(t, err)

	bank, err := LoadExperienceBank(complexBank)
	require.NoError(t, err)
	require.NotNil(t, bank)
	require.Len(t, bank.Stories, 1)

	story := bank.Stories[0]
	assert.Equal(t, "present", story.EndDate)
	require.Len(t, story.Bullets, 2)

	assert.Equal(t, "bullet_001", story.Bullets[0].ID)
	assert.Equal(t, []string{"Go", "Python"}, story.Bullets[0].Skills)
	assert.Equal(t, []string{"flag1", "flag2"}, story.Bullets[0].RiskFlags)

	assert.Equal(t, "bullet_002", story.Bullets[1].ID)
	assert.Equal(t, "50% improvement", story.Bullets[1].Metrics)
}

// Test that LoadExperienceBank returns a pointer to types.ExperienceBank
func TestLoadExperienceBank_ReturnType(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "valid", "experience_bank.json")
	bank, err := LoadExperienceBank(path)
	require.NoError(t, err)
	assert.NotNil(t, bank)
}
