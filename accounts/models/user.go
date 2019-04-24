package models

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/mennanov/fieldmask-utils"
	"github.com/mennanov/scalemate/accounts/accounts_proto"
	"github.com/mennanov/scalemate/shared/events_proto"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mennanov/scalemate/shared/auth"
	"github.com/mennanov/scalemate/shared/events"
	"github.com/mennanov/scalemate/shared/utils"
)

// User defines a User model in DB.
type User struct {
	utils.Model
	Username          string
	Email             string
	Banned            bool
	PasswordHash      []byte
	PasswordChangedAt *time.Time
}

// NewUserFromProto creates a new User instance from a proto message.
func NewUserFromProto(p *accounts_proto.User) (*User, error) {
	user := &User{
		Model: utils.Model{
			ID: p.Id,
		},
		Username: p.Username,
		Email:    p.Email,
		Banned:   p.Banned,
	}

	if p.CreatedAt != nil {
		createdAt, err := ptypes.Timestamp(p.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		user.CreatedAt = createdAt
	}

	if p.UpdatedAt != nil {
		updatedAt, err := ptypes.Timestamp(p.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		user.UpdatedAt = &updatedAt
	}

	if p.PasswordChangedAt != nil {
		passwordChangedAt, err := ptypes.Timestamp(p.PasswordChangedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.Timestamp failed")
		}
		user.PasswordChangedAt = &passwordChangedAt
	}
	return user, nil
}

// ToProto creates a proto message `*accounts_proto.User` and populates it with the User struct contents.
// Only fields that are mentioned in the `fieldMask` are populated. `User.ID` and `User.Username` are always populated
// regardless of the `fieldmask`. If `fieldmask` is nil then all fields are populated.
func (u *User) ToProto(fieldMask *field_mask.FieldMask) (*accounts_proto.User, error) {

	p := &accounts_proto.User{
		Id:       u.ID,
		Username: u.Username,
		Email:    u.Email,
		Banned:   u.Banned,
	}

	if !u.CreatedAt.IsZero() {
		createdAt, err := ptypes.TimestampProto(u.CreatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.CreatedAt = createdAt
	}

	if u.UpdatedAt != nil {
		updatedAt, err := ptypes.TimestampProto(*u.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.UpdatedAt = updatedAt
	}

	if u.PasswordChangedAt != nil {
		passwordChangedAt, err := ptypes.TimestampProto(*u.PasswordChangedAt)
		if err != nil {
			return nil, errors.Wrap(err, "ptypes.TimestampProto failed")
		}
		p.PasswordChangedAt = passwordChangedAt
	}

	if fieldMask != nil && len(fieldMask.Paths) != 0 {
		mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask, generator.CamelCase)
		if err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
		}
		// Always include ID and username regardless of the field mask.
		pFiltered := &accounts_proto.User{Id: u.ID, Username: u.Username}
		if err := fieldmask_utils.StructToStruct(mask, p, pFiltered); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return pFiltered, nil
	}

	return p, nil
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
	if u.ID != 0 {
		return nil, status.Error(codes.FailedPrecondition, "can't create existing user")
	}
	if err := utils.HandleDBError(db.Get(u,
		"INSERT INTO users (username, email, banned, password_hash) VALUES ($1, $2, $3, $4) RETURNING *",
		u.Username, u.Email, u.Banned, u.PasswordHash)); err != nil {
		return nil, errors.Wrap(err, "db.Get failed")
	}
	userProto, err := u.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "user.ToProto failed")
	}
	event, err := events.NewEvent(userProto, events_proto.Event_CREATED, events_proto.Service_ACCOUNTS, nil)
	if err != nil {
		return nil, errors.Wrap(err, "events.NewEvent failed")
	}
	return event, nil
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
	if u.ID == 0 {
		return nil, status.Error(codes.FailedPrecondition, "user not saved in DB")
	}
	if err := u.SetPasswordHash(password, bcryptCost); err != nil {
		return nil, errors.Wrap(err, "failed to set password hash")
	}

	if err := utils.HandleDBError(
		db.Get(u, "UPDATE users SET password_hash = $1, password_changed_at = now() WHERE id = $2 RETURNING *",
			u.PasswordHash, u.ID));
		err != nil {
		return nil, errors.Wrap(err, "failed to update user's password")
	}

	fieldMask := &field_mask.FieldMask{Paths: []string{"password_changed_at", "updated_at"}}
	userProto, err := u.ToProto(fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "user.ToProto failed")
	}
	event, err := events.NewEvent(
		userProto, events_proto.Event_UPDATED, events_proto.Service_ACCOUNTS, fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "events.NewEvent failed")
	}
	return event, nil
}
