package models

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
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

// stringEye is a string identity function used to create field masks.
func stringEye(s string) string {
	return s
}

// User defines a User model in DB.
type User struct {
	gorm.Model
	Username string                   `gorm:"type:varchar(32);unique_index"`
	Email    string                   `gorm:"type:varchar(100);unique_index"`
	Role     accounts_proto.User_Role `gorm:"not null"`
	// Banned indicates whether the user is banned (deactivated).
	Banned            bool `gorm:"not null;default:false"`
	PasswordHash      string
	PasswordChangedAt time.Time
}

// FromProto populates the User fields from accounts_proto.User protobuf.
// Fields `CreatedAt` and `UpdatedAt` are not populated.
func (user *User) FromProto(p *accounts_proto.User) error {
	user.ID = uint(p.GetId())
	user.Username = p.GetUsername()
	user.Email = p.GetEmail()
	user.Role = p.GetRole()
	user.Banned = p.GetBanned()

	if p.PasswordChangedAt != nil {
		passwordChangedAt, err := ptypes.Timestamp(p.PasswordChangedAt)
		if err != nil {
			return errors.Wrap(err, "ptypes.Timestamp failed")
		}
		user.PasswordChangedAt = passwordChangedAt
	}
	return nil
}

// ToProto creates a proto message `*accounts_proto.User` and populates it with the User struct contents.
// Only fields that are mentioned in the `fieldMask` are populated. `User.ID` and `User.Username` are always populated
// regardless of the `fieldmask`. If `fieldmask` is nil then all fields are populated.
func (user *User) ToProto(fieldMask *field_mask.FieldMask) (*accounts_proto.User, error) {
	createdAt, err := ptypes.TimestampProto(user.CreatedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	updatedAt, err := ptypes.TimestampProto(user.UpdatedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	passwordChangedAt, err := ptypes.TimestampProto(user.PasswordChangedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	p := &accounts_proto.User{
		Id:                uint64(user.ID),
		Username:          user.Username,
		Email:             user.Email,
		Role:              user.Role,
		Banned:            user.Banned,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		PasswordChangedAt: passwordChangedAt,
	}

	if fieldMask != nil && len(fieldMask.Paths) != 0 {
		mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask, generator.CamelCase)
		if err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.MaskFromProtoFieldMask failed")
		}
		// Always include ID and username regardless of the field mask.
		pFiltered := &accounts_proto.User{Id: uint64(user.ID), Username: user.Username}
		if err := fieldmask_utils.StructToStruct(mask, p, pFiltered); err != nil {
			return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
		}
		return pFiltered, nil
	}

	return p, nil
}

// SetPasswordHash generates a password hash from the given string and populates User.PasswordHash with it.
func (user *User) SetPasswordHash(password string, bcryptCost int) error {
	if password == "" {
		return status.Error(codes.InvalidArgument, "password can't be empty")
	}

	if bcryptCost == 0 {
		bcryptCost = bcrypt.DefaultCost
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	user.PasswordHash = string(hash[:])

	return nil
}

// ComparePassword compares the given password with the stored user's password hash.
// Returns nil on success, or an error on failure.
func (user *User) ComparePassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
}

// IsAllowedToAuthenticate checks if the user is allowed to authenticate.
// Currently it only checks if the user is banned or not. May become more complicated in future.
func (user *User) IsAllowedToAuthenticate() error {
	if user.Banned {
		return status.Error(codes.PermissionDenied, "user is blocked")
	}
	return nil
}

// NewJWTSigned creates a new JWT and signs it with a secret key.
// When authenticating Nodes `nodeName` argument must not be empty.
// It returns an encoded JWT string to be used over the wire.
func (user *User) NewJWTSigned(
	ttl time.Duration,
	tokenType auth.TokenType,
	jwtSecretKey []byte,
	nodeName string,
) (string, error) {
	now := time.Now()
	expiresAt := now.Add(ttl).Unix()

	claims := &auth.Claims{
		Username:  user.Username,
		NodeName:  nodeName,
		Role:      user.Role,
		TokenType: tokenType,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt,
			Issuer:    "Scalemate.io",
			IssuedAt:  now.Unix(),
			Id:        uuid.New().String(),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecretKey)
	if err != nil {
		return "", status.Errorf(codes.Internal, "could not sign a JWT: %s", err.Error())
	}

	return token, nil
}

// GenerateAuthTokensResponse creates a new `accounts_proto.AuthTokens` response with access and refresh tokens
// for the given user. When authenticating a Node `nodeName` must not be empty.
func (user *User) GenerateAuthTokensResponse(
	ttlAccessToken,
	ttlRefreshToken time.Duration,
	jwtSecretKey []byte,
	nodeName string,
) (*accounts_proto.AuthTokens, error) {
	accessToken, err := user.NewJWTSigned(ttlAccessToken, auth.TokenTypeAccess, jwtSecretKey, nodeName)
	if err != nil {
		return nil, err
	}

	refreshToken, err := user.NewJWTSigned(ttlRefreshToken, auth.TokenTypeRefresh, jwtSecretKey, nodeName)
	if err != nil {
		return nil, err
	}

	return &accounts_proto.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Create creates a new user.
func (user *User) Create(db *gorm.DB) (*events_proto.Event, error) {
	if user.ID != 0 {
		return nil, status.Error(codes.InvalidArgument, "can't create existing user")
	}
	if err := utils.HandleDBError(db.Create(user)); err != nil {
		return nil, err
	}
	userProto, err := user.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "user.ToProto failed")
	}
	event, err := events.
		NewEventFromPayload(userProto, events_proto.Event_CREATED, events_proto.Service_ACCOUNTS, nil)
	if err != nil {
		return nil, err
	}
	return event, nil
}

// LookUp gets a user by UserLookupRequest.
func (user *User) LookUp(db *gorm.DB, req *accounts_proto.UserLookupRequest) error {
	if req.GetId() != 0 {
		return utils.HandleDBError(db.First(user, req.GetId()))
	} else if req.GetUsername() != "" {
		return utils.HandleDBError(db.Where("username = ?", req.GetUsername()).First(user))
	} else if req.GetEmail() != "" {
		return utils.HandleDBError(db.Where("email = ?", req.GetEmail()).First(user))
	}
	return status.Error(codes.InvalidArgument, "invalid UserLookupRequest: all fields are empty")
}

// Delete soft-deletes a user.
func (user *User) Delete(db *gorm.DB) (*events_proto.Event, error) {
	if user.ID == 0 {
		// gorm will delete all records if PK is zero.
		return nil, status.Error(codes.InvalidArgument, "can't delete a user without a primary key")
	}
	if err := utils.HandleDBError(db.Delete(user)); err != nil {
		return nil, err
	}
	userProto, err := user.ToProto(nil)
	if err != nil {
		return nil, errors.Wrap(err, "user.ToProto failed")
	}
	event, err := events.
		NewEventFromPayload(userProto, events_proto.Event_DELETED, events_proto.Service_ACCOUNTS, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new accounts_user.delete event")
	}
	return event, nil
}

// Update updates the user data by the given payload and field mask.
func (user *User) Update(
	db *gorm.DB,
	payload *accounts_proto.User,
	fieldMask *field_mask.FieldMask,
) (*events_proto.Event, error) {
	u := &User{}
	if err := u.FromProto(payload); err != nil {
		return nil, errors.Wrap(err, "failed to populate user from proto")
	}
	mask, err := fieldmask_utils.MaskFromProtoFieldMask(fieldMask, generator.CamelCase)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Update the `user` struct field values.
	if err := fieldmask_utils.StructToStruct(mask, u, user); err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.StructToStruct failed")
	}

	// Prepare the UPDATE query map.
	data := make(map[string]interface{})
	if err := fieldmask_utils.StructToMap(mask, u, data); err != nil {
		return nil, errors.Wrap(err, "fieldmask_utils.StructToMap failed")
	}
	if err := utils.HandleDBError(db.Model(user).Updates(data)); err != nil {
		return nil, err
	}
	userProto, err := user.ToProto(fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "user.ToProto failed")
	}
	event, err := events.
		NewEventFromPayload(userProto, events_proto.Event_UPDATED, events_proto.Service_ACCOUNTS, fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new account_users.updated event")
	}
	return event, nil
}

// ChangePassword sets and updates the user's password.
func (user *User) ChangePassword(db *gorm.DB, password string, bcryptCost int) (*events_proto.Event, error) {
	if err := user.SetPasswordHash(password, bcryptCost); err != nil {
		return nil, errors.Wrap(err, "failed to set password hash")
	}
	user.PasswordChangedAt = time.Now()

	if err := utils.HandleDBError(db.Model(user).
		Updates(map[string]interface{}{
			"password_hash":       user.PasswordHash,
			"password_changed_at": user.PasswordChangedAt,
		})); err != nil {
		return nil, err
	}
	fieldMask := &field_mask.FieldMask{Paths: []string{"password_changed_at", "updated_at"}}
	userProto, err := user.ToProto(fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "user.ToProto failed")
	}
	event, err := events.NewEventFromPayload(
		userProto, events_proto.Event_UPDATED, events_proto.Service_ACCOUNTS, fieldMask)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create a new account_users.updated.password event")
	}
	return event, nil
}

// Users represent a slice of `User`s.
// Some specific methods are defined on that type to manipulate a slice of `User`s.
type Users []User

// List queries DB for Users by the given constraints in accounts_proto.ListUsersRequest.
func (users *Users) List(db *gorm.DB, r *accounts_proto.ListUsersRequest) (uint32, error) {
	query := db.Model(&User{})
	ordering := r.GetOrdering()

	if len(ordering) == 0 {
		query = query.Order("created_at DESC")
	}

	var orderBySQL string
	for _, orderBy := range ordering {
		switch orderBy {
		case accounts_proto.ListUsersRequest_CREATED_AT_ASC:
			orderBySQL = "created_at"
		case accounts_proto.ListUsersRequest_CREATED_AT_DESC:
			orderBySQL = "created_at DESC"
		case accounts_proto.ListUsersRequest_UPDATED_AT_ASC:
			orderBySQL = "updated_at"
		case accounts_proto.ListUsersRequest_UPDATED_AT_DESC:
			orderBySQL = "updated_at DESC"
		case accounts_proto.ListUsersRequest_USERNAME_ASC:
			orderBySQL = "username"
		case accounts_proto.ListUsersRequest_USERNAME_DESC:
			orderBySQL = "username DESC"
		case accounts_proto.ListUsersRequest_EMAIL_ASC:
			orderBySQL = "email"
		case accounts_proto.ListUsersRequest_EMAIL_DESC:
			orderBySQL = "email DESC"
		}
		query = query.Order(orderBySQL)
	}

	roles := r.GetRole()

	if len(roles) != 0 {
		query = query.Where("role in (?)", roles)
	}

	if r.GetBannedOnly() {
		query = query.Where("banned = ?", true)
	} else if r.GetActiveOnly() {
		query = query.Where("banned = ?", false)
	}

	// Perform a COUNT query with no limit and offset applied.
	var count uint32
	if err := utils.HandleDBError(query.Count(&count)); err != nil {
		return 0, err
	}

	// Apply offset.
	if r.GetOffset() != 0 {
		query = query.Offset(r.GetOffset())
	}

	// Apply limit.
	var limit uint32
	if r.GetLimit() != 0 {
		limit = r.GetLimit()
	} else {
		limit = 50
	}
	query = query.Limit(limit)

	// Perform a SELECT query.
	if err := utils.HandleDBError(query.Find(&users)); err != nil {
		return 0, err
	}
	return count, nil
}
