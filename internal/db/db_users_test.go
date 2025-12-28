package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB connects to the local DB for integration testing
// Skipped if DATABASE_URL is not set or connection fails
func setupTestDB(t *testing.T) *DB {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Default to local docker connection
		dbURL = "postgres://resume:resume_dev@localhost:5432/resume_customizer?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	db, err := Connect(ctx, dbURL)
	if err != nil {
		t.Skipf("Skipping integration test: failed to connect to DB: %v", err)
	}
	return db
}

func TestUserCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// 1. Create
	name := "Test User"
	email := "test-" + uuid.New().String() + "@example.com"
	phone := "555-0100"
	id, err := db.CreateUser(ctx, name, email, phone)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)

	// 2. Get
	u, err := db.GetUser(ctx, id)
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, name, u.Name)
	assert.Equal(t, email, u.Email)
	assert.Equal(t, phone, u.Phone)

	// 3. Update
	u.Name = "Updated Name"
	err = db.UpdateUser(ctx, u)
	require.NoError(t, err)

	u2, err := db.GetUser(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", u2.Name)

	// 4. Delete
	err = db.DeleteUser(ctx, id)
	require.NoError(t, err)

	u3, err := db.GetUser(ctx, id)
	require.NoError(t, err)
	assert.Nil(t, u3)
}

func TestJobCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Create parent user
	uid, err := db.CreateUser(ctx, "Job Tester", "job-"+uuid.New().String()+"@test.com", "123")
	require.NoError(t, err)
	defer db.DeleteUser(ctx, uid) // Cleanup

	// 1. Create Job
	now := time.Now()
	job := &Job{
		UserID:         uid,
		Company:        "Acme Corp",
		RoleTitle:      "Engineer",
		Location:       "Remote",
		EmploymentType: "full-time",
		StartDate:      &Date{Time: now.AddDate(-1, 0, 0)},
		EndDate:        nil, // Current job
	}
	jid, err := db.CreateJob(ctx, job)
	require.NoError(t, err)

	// 2. List
	jobs, err := db.ListJobs(ctx, uid)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "Acme Corp", jobs[0].Company)

	// 3. Update
	job.ID = jid
	job.Company = "Acme Inc"
	err = db.UpdateJob(ctx, job)
	require.NoError(t, err)

	jobs2, err := db.ListJobs(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, "Acme Inc", jobs2[0].Company)

	// 4. Delete
	err = db.DeleteJob(ctx, jid)
	require.NoError(t, err)

	jobs3, err := db.ListJobs(ctx, uid)
	require.NoError(t, err)
	assert.Len(t, jobs3, 0)
}

func TestExperienceCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Setup User & Job
	uid, _ := db.CreateUser(ctx, "Exp Tester", "exp-"+uuid.New().String()+"@test.com", "123")
	defer db.DeleteUser(ctx, uid)
	jid, _ := db.CreateJob(ctx, &Job{UserID: uid, Company: "Test", RoleTitle: "Role"})

	// 1. Create Experience with Skills
	exp := &Experience{
		JobID:            jid,
		BulletText:       "Did cool things",
		Skills:           []string{"Go", "SQL"},
		EvidenceStrength: "high",
		RiskFlags:        []string{},
	}
	eid, err := db.CreateExperience(ctx, exp)
	require.NoError(t, err)

	// 2. List
	exps, err := db.ListExperiences(ctx, jid)
	require.NoError(t, err)
	require.Len(t, exps, 1)
	assert.Equal(t, "Did cool things", exps[0].BulletText)
	assert.Equal(t, []string{"Go", "SQL"}, []string(exps[0].Skills))

	// 3. Update (change skills)
	exp.ID = eid
	exp.Skills = []string{"Python"}
	err = db.UpdateExperience(ctx, exp)
	require.NoError(t, err)

	exps2, err := db.ListExperiences(ctx, jid)
	require.NoError(t, err)
	assert.Equal(t, []string{"Python"}, []string(exps2[0].Skills))

	// 4. Delete
	err = db.DeleteExperience(ctx, eid)
	require.NoError(t, err)
}

func TestEducationCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// Setup User
	uid, _ := db.CreateUser(ctx, "Edu Tester", "edu-"+uuid.New().String()+"@test.com", "123")
	defer db.DeleteUser(ctx, uid)

	// 1. Create
	edu := &Education{
		UserID:     uid,
		School:     "University of Test",
		DegreeType: "Bachelor",
		Field:      "CS",
		GPA:        "4.0",
	}
	eid, err := db.CreateEducation(ctx, edu)
	require.NoError(t, err)

	// 2. List
	edus, err := db.ListEducation(ctx, uid)
	require.NoError(t, err)
	require.Len(t, edus, 1)
	assert.Equal(t, "University of Test", edus[0].School)

	// 3. Update
	edu.ID = eid
	edu.GPA = "3.9"
	err = db.UpdateEducation(ctx, edu)
	require.NoError(t, err)

	edus2, err := db.ListEducation(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, "3.9", edus2[0].GPA)

	// 4. Delete
	err = db.DeleteEducation(ctx, eid)
	require.NoError(t, err)
}
