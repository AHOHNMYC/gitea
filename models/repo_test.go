// Copyright 2017 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/markup"

	"github.com/stretchr/testify/assert"
)

func TestCheckRepoStats(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	assert.NoError(t, CheckRepoStats(db.DefaultContext))
}

func TestWatchRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	const repoID = 3
	const userID = 2

	assert.NoError(t, repo_model.WatchRepo(db.DefaultContext, userID, repoID, true))
	unittest.AssertExistsAndLoadBean(t, &repo_model.Watch{RepoID: repoID, UserID: userID})
	unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: repoID})

	assert.NoError(t, repo_model.WatchRepo(db.DefaultContext, userID, repoID, false))
	unittest.AssertNotExistsBean(t, &repo_model.Watch{RepoID: repoID, UserID: userID})
	unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: repoID})
}

func TestMetas(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := &repo_model.Repository{Name: "testRepo"}
	repo.Owner = &user_model.User{Name: "testOwner"}
	repo.OwnerName = repo.Owner.Name

	repo.Units = nil

	metas := repo.ComposeMetas()
	assert.Equal(t, "testRepo", metas["repo"])
	assert.Equal(t, "testOwner", metas["user"])

	externalTracker := repo_model.RepoUnit{
		Type: unit.TypeExternalTracker,
		Config: &repo_model.ExternalTrackerConfig{
			ExternalTrackerFormat: "https://someurl.com/{user}/{repo}/{issue}",
		},
	}

	testSuccess := func(expectedStyle string) {
		repo.Units = []*repo_model.RepoUnit{&externalTracker}
		repo.RenderingMetas = nil
		metas := repo.ComposeMetas()
		assert.Equal(t, expectedStyle, metas["style"])
		assert.Equal(t, "testRepo", metas["repo"])
		assert.Equal(t, "testOwner", metas["user"])
		assert.Equal(t, "https://someurl.com/{user}/{repo}/{issue}", metas["format"])
	}

	testSuccess(markup.IssueNameStyleNumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markup.IssueNameStyleAlphanumeric
	testSuccess(markup.IssueNameStyleAlphanumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markup.IssueNameStyleNumeric
	testSuccess(markup.IssueNameStyleNumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markup.IssueNameStyleRegexp
	testSuccess(markup.IssueNameStyleRegexp)

	repo, err := repo_model.GetRepositoryByID(3)
	assert.NoError(t, err)

	metas = repo.ComposeMetas()
	assert.Contains(t, metas, "org")
	assert.Contains(t, metas, "teams")
	assert.Equal(t, "user3", metas["org"])
	assert.Equal(t, ",owners,team1,", metas["teams"])
}

func TestDoctorUserStarNum(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	assert.NoError(t, DoctorUserStarNum())
}
