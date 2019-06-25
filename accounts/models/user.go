package models

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/utils"
)

var psq = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

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
	data := map[string]interface{}{
		"username":      u.Username,
		"email":         u.Email,
		"banned":        u.Banned,
		"password_hash": u.PasswordHash,
	}
	if !u.CreatedAt.IsZero() {
		data["created_at"] = u.CreatedAt
	}
	query, args, err := psq.Insert("users").SetMap(data).Suffix("RETURNING *").ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := utils.HandleDBError(db.Get(u, query, args...)); err != nil {
		return nil, errors.WithStack(err)
	}

	return &events_proto.Event{
		Payload: &events_proto.Event_AccountsUserCreated{
			AccountsUserCreated: &accounts_proto.UserCreatedEvent{
				User: &u.User,
			},
		},
		CreatedAt: time.Now().UTC(),
	}, nil
}

// UserLookUp gets a User by UserLookupRequest.
func UserLookUp(db utils.SqlxGetter, req *accounts_proto.UserLookupRequest) (*User, error) {
	var err error
	user := &User{}
	query := psq.Select("*").From("users").Limit(1)
	switch r := req.Request.(type) {
	case *accounts_proto.UserLookupRequest_Id:
		query = query.Where("id = ?", r.Id)

	case *accounts_proto.UserLookupRequest_Username:
		query = query.Where("username = ?", r.Username)

	case *accounts_proto.UserLookupRequest_Email:
		query = query.Where("email = ?", r.Email)
	}
	queryString, args, err := query.ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := db.Get(user, queryString, args...); err != nil {
		return nil, utils.HandleDBError(err)
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
	query, args, err := psq.Update("users").Where("id = ?", u.Id).Set("password_hash", u.PasswordHash).
		Suffix("RETURNING *").ToSql()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := utils.HandleDBError(db.Get(u, query, args...)); err != nil {
		return nil, errors.Wrap(err, "failed to update user's password")
	}

	return &events_proto.Event{
		Payload: &events_proto.Event_AccountsUserPasswordChanged{
			AccountsUserPasswordChanged: &accounts_proto.UserPasswordChangedEvent{
				UserId: u.Id,
			},
		},
		CreatedAt: time.Now().UTC(),
	}, nil
}
