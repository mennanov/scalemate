package models_test

import (
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"

	"github.com/mennanov/scalemate/accounts/accounts_proto"

	"github.com/mennanov/scalemate/accounts/models"
	"github.com/mennanov/scalemate/shared/testutils"
)

func (s *ModelsTestSuite) TestUser_SetPasswordHash_ComparePassword() {
	user := new(models.User)
	s.Require().NoError(user.SetPasswordHash("password", bcrypt.MinCost))
	s.NoError(user.ComparePassword("password"))
	s.Error(user.ComparePassword("password2"))
}

func (s *ModelsTestSuite) TestUser_Create_Lookup() {
	for _, userTest := range []*models.User{
		{
			User: accounts_proto.User{
				Username: "username",
				Email:    "email@mail.com",
				Banned:   false,
			},
			PasswordHash: []byte("password"),
		},
		{
			User: accounts_proto.User{
				Username: "username2",
				Email:    "email2@mail.com",
				Banned:   true,
			},
			PasswordHash: []byte("password"),
		},
	} {
		event, err := userTest.Create(s.db)
		s.Require().NoError(err)
		s.NotNil(event)

		for _, lookupRequest := range []*accounts_proto.UserLookupRequest{
			{
				Request: &accounts_proto.UserLookupRequest_Email{
					Email: userTest.Email,
				},
			},
			{
				Request: &accounts_proto.UserLookupRequest_Username{
					Username: userTest.Username,
				},
			},
			{
				Request: &accounts_proto.UserLookupRequest_Id{
					Id: userTest.Id,
				},
			},
		} {
			userFromDb, err := models.UserLookUp(s.db, lookupRequest)
			s.Require().NoError(err)
			// User selected from DB is identical to userTest.
			s.Equal(userTest, userFromDb)
			// A lookup for a non-existing user fails.
			u, err := models.UserLookUp(s.db, &accounts_proto.UserLookupRequest{
				Request: &accounts_proto.UserLookupRequest_Id{
					Id: userTest.Id + 1,
				},
			})
			testutils.AssertErrorCode(s.T(), err, codes.NotFound)
			s.Nil(u)
		}
		// The same user can not be created again (ID check, no DB query needed).
		_, err = userTest.Create(s.db)
		testutils.AssertErrorCode(s.T(), err, codes.FailedPrecondition)
		// The same user can not be created again (hits the DB).
		userTest.Id = 0
		_, err = userTest.Create(s.db)
		testutils.AssertErrorCode(s.T(), err, codes.AlreadyExists)
	}
}

func (s *ModelsTestSuite) TestUser_ChangePassword() {
	user := models.NewUserFromProto(&accounts_proto.User{
		Username: "username",
		Email:    "email@mail.com",
		Banned:   false,
	})
	s.Require().NoError(user.SetPasswordHash("password", bcrypt.MinCost))
	_, err := user.Create(s.db)
	s.Require().NoError(err)
	s.NoError(user.ComparePassword("password"))
	event, err := user.ChangePassword(s.db, "new password", bcrypt.MinCost)
	s.Require().NoError(err)
	s.NotNil(event)
	s.NoError(user.ComparePassword("new password"))
}
