// Package types provides type definitions for structured data used throughout the resume-customizer system.
//
//nolint:revive // types is a standard Go package name pattern
package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepairAction_JSONMarshaling_ShortenBullet(t *testing.T) {
	targetChars := 80
	action := RepairAction{
		Type:        "shorten_bullet",
		BulletID:    "bullet_001",
		TargetChars: &targetChars,
		Reason:      "Bullet exceeds line length limit",
	}

	jsonBytes, err := json.MarshalIndent(action, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"type": "shorten_bullet"`)
	assert.Contains(t, string(jsonBytes), `"bullet_id": "bullet_001"`)
	assert.Contains(t, string(jsonBytes), `"target_chars": 80`)
	assert.Contains(t, string(jsonBytes), `"reason": "Bullet exceeds line length limit"`)

	var unmarshaled RepairAction
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, action.Type, unmarshaled.Type)
	assert.Equal(t, action.BulletID, unmarshaled.BulletID)
	assert.NotNil(t, unmarshaled.TargetChars)
	assert.Equal(t, targetChars, *unmarshaled.TargetChars)
	assert.Equal(t, action.Reason, unmarshaled.Reason)
}

func TestRepairAction_JSONMarshaling_DropBullet(t *testing.T) {
	action := RepairAction{
		Type:     "drop_bullet",
		BulletID: "bullet_002",
		Reason:   "Bullet causes page overflow",
	}

	jsonBytes, err := json.MarshalIndent(action, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"type": "drop_bullet"`)
	assert.Contains(t, string(jsonBytes), `"bullet_id": "bullet_002"`)

	var unmarshaled RepairAction
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, action.Type, unmarshaled.Type)
	assert.Equal(t, action.BulletID, unmarshaled.BulletID)
	assert.Nil(t, unmarshaled.TargetChars)
}

func TestRepairAction_JSONMarshaling_SwapStory(t *testing.T) {
	action := RepairAction{
		Type:    "swap_story",
		StoryID: "story_001",
		Reason:  "Story has forbidden phrases",
	}

	jsonBytes, err := json.MarshalIndent(action, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"type": "swap_story"`)
	assert.Contains(t, string(jsonBytes), `"story_id": "story_001"`)

	var unmarshaled RepairAction
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, action.Type, unmarshaled.Type)
	assert.Equal(t, action.StoryID, unmarshaled.StoryID)
}

func TestRepairActions_JSONMarshaling(t *testing.T) {
	targetChars := 75
	actions := RepairActions{
		Actions: []RepairAction{
			{
				Type:        "shorten_bullet",
				BulletID:    "bullet_001",
				TargetChars: &targetChars,
				Reason:      "Reduce length",
			},
			{
				Type:     "drop_bullet",
				BulletID: "bullet_002",
				Reason:   "Remove to fit page",
			},
		},
	}

	jsonBytes, err := json.MarshalIndent(actions, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"actions": [`)
	assert.Contains(t, string(jsonBytes), `"shorten_bullet"`)
	assert.Contains(t, string(jsonBytes), `"drop_bullet"`)

	var unmarshaled RepairActions
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	require.NoError(t, err)
	assert.Len(t, unmarshaled.Actions, 2)
}
