package models

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gogo/protobuf/types"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// User defines a User model in DB.
type User struct {
	accounts_proto.User
	PasswordHash []byte
}

// NewUserFromProto creates a new User instance from a proto message.
func NewUserFromProto(user *accounts_proto.User) *User {
	return &User{User: *user}
}

// SetPasswordHash generates a password hash from the given string and populates User.PasswordHash with it.
func (u *User) SetPasswordHash(password string, bcryptCost int) error {
	if password == "" {
		return status.Error(codes.InvalidArgument, "password can't be empty")
	}

	if bcryptCost == 0 {
		bcryptCost = bcrypt.DefaultCost
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return errors.Wrap(err, "bcrypt.GenerateFromPassword failed")
	}

	u.PasswordHash = hash
	return nil
}

// ComparePassword compares the given password with the stored user's password hash.
// Returns nil on success, or an error on failure.
func (u *User) ComparePassword(password string) error {
	return bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password))
}

// IsAllowedToAuthenticate checks if the user is allowed to authenticate.
// Currently it only checks if the user is banned or not. May become more complicated in future.
func (u *User) IsAllowedToAuthenticate() error {
	if u.Banned {
		return status.Error(codes.PermissionDenied, "user is banned")
	}
	return nil
}

// GenerateAuthTokens creates a new `accounts_proto.AuthTokens` proto message with access and refresh tokens
// for the given user. When authenticating a Node `nodeName` must be provided.
func (u *User) GenerateAuthTokens(
	ttlAccessToken,
	ttlRefreshToken time.Duration,
	jwtSecretKey []byte,
	nodeName string,
) (*accounts_proto.AuthTokens, error) {
	now := time.Now().UTC()

	claims := auth.NewClaims(jwt.StandardClaims{
		ExpiresAt: now.Add(ttlAccessToken).Unix(),
		Id:        uuid.New().String(),
		IssuedAt:  now.Unix(),
	}, u.Username, nodeName, auth.TokenTypeAccess)
	accessToken, err := claims.SignedString(jwtSecretKey)
	if err != nil {
		return nil, errors.Wrap(err, "claims.SignedString failed")
	}

	claims = auth.NewClaims(jwt.StandardClaims{
		ExpiresAt: now.Add(ttlRefreshToken).Unix(),
		Id:        uuid.New().String(),
		IssuedAt:  now.Unix(),
	}, u.Username, nodeName, auth.TokenTypeRefresh)

	if err != nil {
		return nil, errors.Wrap(err, "claims.SignedString failed")
	}
	refreshToken, err := claims.SignedString(jwtSecretKey)
	return &accounts_proto.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Create creates a new User.
func (u *User) Create(db utils.SqlxGetter) (*events_proto.Event, error) {
	if u.Id != 0 {
		return nil, status.Error(codes.FailedPrecondition, "can't create existing user")
	}
	if err := utils.HandleDBError(db.Get(u,
		"INSERT INTO users (username, email, banned, password_hash) VALUES ($1, $2, $3, $4) RETURNING *",
		u.Username, u.Email, u.Banned, u.PasswordHash)); err != nil {
		return nil, errors.Wrap(err, "db.Get failed")
	}
	return events.NewEvent(&u.User, events_proto.Event_CREATED, events_proto.Service_ACCOUNTS, nil), nil
}

// UserLookUp gets a User by UserLookupRequest.
func UserLookUp(db utils.SqlxGetter, req *accounts_proto.UserLookupRequest) (*User, error) {
	var err error
	user := &User{}
	switch r := req.Request.(type) {
	case *accounts_proto.UserLookupRequest_Id:
		err = utils.HandleDBError(db.Get(user, "SELECT * FROM users WHERE id = $1", r.Id))

	case *accounts_proto.UserLookupRequest_Username:
		err = utils.HandleDBError(db.Get(user, "SELECT * FROM users WHERE username = $1", r.Username))

	case *accounts_proto.UserLookupRequest_Email:
		err = utils.HandleDBError(db.Get(user, "SELECT * FROM users WHERE email = $1", r.Email))
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform a select request")
	}
	return user, nil
}

// ChangePassword sets and updates the user's password.
func (u *User) ChangePassword(db utils.SqlxGetter, password string, bcryptCost int) (*events_proto.Event, error) {
	if u.Id == 0 {
		return nil, status.Error(codes.FailedPrecondition, "user not saved in DB")
	}
	if err := u.SetPasswordHash(password, bcryptCost); err != nil {
		return nil, errors.Wrap(err, "failed to set password hash")
	}

	if err := utils.HandleDBError(
		db.Get(u, "UPDATE users SET password_hash = $1, password_changed_at = now() WHERE id = $2 RETURNING *",
			u.PasswordHash, u.Id));
		err != nil {
		return nil, errors.Wrap(err, "failed to update user's password")
	}

	fieldMask := &types.FieldMask{Paths: []string{"password_changed_at", "updated_at"}}
	return events.NewEvent(&u.User, events_proto.Event_UPDATED, events_proto.Service_ACCOUNTS, fieldMask), nil
}
