// Package rewriting provides functionality to rewrite bullet points to match job requirements and company brand voice.
package rewriting

import (
	"testing"

	"github.com/jonathan/resume-customizer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForbiddenPhrasesInText(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		tabooPhrases  []string
		expectedFound []string
	}{
		{
			name:          "no forbidden phrases",
			text:          "I am a software engineer",
			tabooPhrases:  []string{"ninja", "rockstar", "guru"},
			expectedFound: nil,
		},
		{
			name:          "single forbidden phrase",
			text:          "I am a coding ninja",
			tabooPhrases:  []string{"ninja", "rockstar", "guru"},
			expectedFound: []string{"ninja"},
		},
		{
			name:          "multiple forbidden phrases",
			text:          "I am a coding ninja and a rockstar developer",
			tabooPhrases:  []string{"ninja", "rockstar", "guru"},
			expectedFound: []string{"ninja", "rockstar"},
		},
		{
			name:          "case insensitive",
			text:          "I am a NINJA developer",
			tabooPhrases:  []string{"ninja", "rockstar"},
			expectedFound: []string{"ninja"},
		},
		{
			name:          "phrase as substring",
			text:          "I am a ninja developer",
			tabooPhrases:  []string{"ninja"},
			expectedFound: []string{"ninja"},
		},
		{
			name:          "empty taboo phrases",
			text:          "I am a ninja",
			tabooPhrases:  []string{},
			expectedFound: nil,
		},
		{
			name:          "empty text",
			text:          "",
			tabooPhrases:  []string{"ninja"},
			expectedFound: nil,
		},
		{
			name:          "duplicate phrases in list",
			text:          "I am a ninja",
			tabooPhrases:  []string{"ninja", "ninja", "rockstar"},
			expectedFound: []string{"ninja"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := checkForbiddenPhrasesInText(tt.text, tt.tabooPhrases)
			if tt.expectedFound == nil {
				assert.Nil(t, found)
			} else {
				require.Equal(t, len(tt.expectedFound), len(found))
				// Check that all expected phrases are found (order doesn't matter)
				foundSet := make(map[string]bool)
				for _, phrase := range found {
					foundSet[phrase] = true
				}
				for _, expected := range tt.expectedFound {
					assert.True(t, foundSet[expected], "Expected phrase %q not found", expected)
				}
			}
		})
	}
}

func TestCheckForbiddenPhrasesInBullets(t *testing.T) {
	tests := []struct {
		name           string
		bullets        *types.RewrittenBullets
		companyProfile *types.CompanyProfile
		expectedMap    map[string][]string
	}{
		{
			name: "no forbidden phrases",
			bullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{
						OriginalBulletID: "bullet_001",
						FinalText:        "I am a software engineer",
					},
				},
			},
			companyProfile: &types.CompanyProfile{
				TabooPhrases: []string{"ninja", "rockstar"},
			},
			expectedMap: map[string][]string{},
		},
		{
			name: "single bullet with forbidden phrase",
			bullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{
						OriginalBulletID: "bullet_001",
						FinalText:        "I am a coding ninja",
					},
				},
			},
			companyProfile: &types.CompanyProfile{
				TabooPhrases: []string{"ninja", "rockstar"},
			},
			expectedMap: map[string][]string{
				"bullet_001": []string{"ninja"},
			},
		},
		{
			name: "multiple bullets with forbidden phrases",
			bullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{
						OriginalBulletID: "bullet_001",
						FinalText:        "I am a coding ninja",
					},
					{
						OriginalBulletID: "bullet_002",
						FinalText:        "I am a rockstar developer",
					},
					{
						OriginalBulletID: "bullet_003",
						FinalText:        "I am a software engineer",
					},
				},
			},
			companyProfile: &types.CompanyProfile{
				TabooPhrases: []string{"ninja", "rockstar"},
			},
			expectedMap: map[string][]string{
				"bullet_001": []string{"ninja"},
				"bullet_002": []string{"rockstar"},
			},
		},
		{
			name:           "nil bullets",
			bullets:        nil,
			companyProfile: &types.CompanyProfile{TabooPhrases: []string{"ninja"}},
			expectedMap:    nil,
		},
		{
			name: "nil company profile",
			bullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{
						OriginalBulletID: "bullet_001",
						FinalText:        "I am a ninja",
					},
				},
			},
			companyProfile: nil,
			expectedMap:    nil,
		},
		{
			name: "empty taboo phrases",
			bullets: &types.RewrittenBullets{
				Bullets: []types.RewrittenBullet{
					{
						OriginalBulletID: "bullet_001",
						FinalText:        "I am a ninja",
					},
				},
			},
			companyProfile: &types.CompanyProfile{
				TabooPhrases: []string{},
			},
			expectedMap: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckForbiddenPhrasesInBullets(tt.bullets, tt.companyProfile)
			if tt.expectedMap == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				require.Equal(t, len(tt.expectedMap), len(result))
				for bulletID, expectedPhrases := range tt.expectedMap {
					foundPhrases, exists := result[bulletID]
					require.True(t, exists, "Bullet ID %q not found in result", bulletID)
					require.Equal(t, len(expectedPhrases), len(foundPhrases))
					// Check that all expected phrases are found (order doesn't matter)
					foundSet := make(map[string]bool)
					for _, phrase := range foundPhrases {
						foundSet[phrase] = true
					}
					for _, expected := range expectedPhrases {
						assert.True(t, foundSet[expected], "Expected phrase %q not found for bullet %q", expected, bulletID)
					}
				}
			}
		})
	}
}
