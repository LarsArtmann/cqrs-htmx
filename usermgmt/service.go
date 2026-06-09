package usermgmt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const (
	defaultSessionTTL    = 24 * time.Hour
	maxDisplayNameLength = 100
)

// Service orchestrates user registration, authentication, authorization, and session management.
type Service struct {
	authz        *Authz
	users        UserStore
	sessions     SessionStore
	sessionTTL   time.Duration
	bcryptCost   int
	logger       *slog.Logger
	lockout      *AccountLockout
	eventHandler EventHandler
}

// ServiceConfig holds optional dependencies for NewService.
// Zero-valued fields are replaced with sensible defaults.
type ServiceConfig struct {
	// Authz is the authorization engine. Defaults to a new Authz with default policies.
	Authz *Authz
	// UserStore is the user persistence backend. Defaults to InMemoryUserStore.
	UserStore UserStore
	// SessionStore is the session persistence backend. Defaults to InMemorySessionStore.
	SessionStore SessionStore
	// SessionTTL is the default time-to-live for new sessions. Defaults to 24 hours.
	SessionTTL time.Duration
	// BcryptCost is the bcrypt hashing cost. Defaults to 12. Values below 4 are clamped.
	BcryptCost int
	// Logger is used for structured auth event logging. Defaults to slog.Default().
	Logger *slog.Logger
	// Lockout, if provided, enables account lockout after repeated login failures.
	Lockout *AccountLockout
	// EventHandler, if provided, is called after successful domain operations.
	// See events.go for the event types that may be passed.
	EventHandler EventHandler
}

// NewService creates a Service from the given config, applying defaults for nil/zero fields.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.UserStore == nil {
		cfg.UserStore = NewInMemoryUserStore()
	}
	if cfg.SessionStore == nil {
		cfg.SessionStore = NewInMemorySessionStore()
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = defaultSessionTTL
	}
	if cfg.Authz == nil {
		a, err := NewAuthz()
		if err != nil {
			return nil, event.NewTransient("internal", "create authz").WithCause(err)
		}
		cfg.Authz = a
	}

	cost := cfg.BcryptCost
	if cost < minBcryptCost {
		cost = defaultBcryptCost
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		authz:        cfg.Authz,
		users:        cfg.UserStore,
		sessions:     cfg.SessionStore,
		sessionTTL:   cfg.SessionTTL,
		bcryptCost:   cost,
		logger:       logger,
		lockout:      cfg.Lockout,
		eventHandler: cfg.EventHandler,
	}, nil
}

// Authz returns the underlying authorization engine for direct policy queries.
// Mutations to the returned Authz affect the Service's authorization behavior.
func (s *Service) Authz() *Authz { return s.authz }

// RegisterRequest is the input for user registration.
type RegisterRequest struct {
	ID          UserID `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

func formatValidationErrors(errs []string) error {
	if len(errs) == 0 {
		return nil
	}

	return event.NewRejection("validation", strings.Join(errs, "; ")).WithCause(ErrValidation)
}

// Validate checks the RegisterRequest fields and returns ErrValidation with
// a joined list of problems if any field is invalid.
// It trims leading/trailing whitespace from Email and DisplayName in-place.
func (r *RegisterRequest) Validate() error {
	var errs []string
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	if r.ID.IsZero() {
		errs = append(errs, "id is required")
	}
	if _, err := mail.ParseAddress(r.Email); err != nil {
		errs = append(errs, "invalid email")
	}
	if err := validatePassword(r.Password); err != nil {
		errStr := err.Error()
		errs = append(errs, errStr)
	}
	if len(r.DisplayName) > maxDisplayNameLength {
		errs = append(errs,
			"display name must be under "+strconv.Itoa(maxDisplayNameLength)+" characters")
	}
	return formatValidationErrors(errs)
}

// RegisterResponse contains the newly created User and active Session.
type RegisterResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

// Register validates the request, creates the user, assigns the "user" role,
// and opens a session. Partial failures are compensated: if role assignment or
// session creation fails after the user is created, the user and role are rolled back.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	user := NewUser(req.ID, req.Email, req.DisplayName)
	if err := user.SetPasswordWithCost(req.Password, s.bcryptCost); err != nil {
		return nil, event.NewTransient("internal", "set password").WithCause(err)
	}

	user.AddRole(RoleUser)

	if err := s.users.Create(ctx, user); err != nil {
		return nil, event.NewTransient("internal", "create user").WithCause(err)
	}

	policy := GroupPolicy{
		Subject: user.ID.Get(), Role: RoleUser, Domain: user.ID.Get(),
	}
	if err := s.authz.AddGroupPolicy(policy); err != nil {
		if delErr := s.users.Delete(ctx, user.ID); delErr != nil {
			s.logAuth("register_rollback_delete_failed", user.ID, "rollback_error", delErr)
		}
		return nil, event.NewTransient("internal", "assign role").WithCause(err)
	}

	session, err := s.sessions.Create(ctx, user.ID, s.sessionTTL)
	if err != nil {
		if rmErr := s.authz.RemoveGroupPolicy(policy); rmErr != nil {
			s.logAuth("register_rollback_policy_failed", user.ID, "rollback_error", rmErr)
		}
		if delErr := s.users.Delete(ctx, user.ID); delErr != nil {
			s.logAuth("register_rollback_delete_failed", user.ID, "rollback_error", delErr)
		}
		return nil, event.NewTransient("internal", "create session").WithCause(err)
	}

	s.emit(user.ID, UserRegisteredEvent{
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Roles:       append([]Role(nil), user.Roles...),
		OccurredAt:  time.Now().UTC(),
	})
	return &RegisterResponse{User: user, Session: session}, nil
}

func (s *Service) logAuth(event string, userID UserID, attrs ...any) {
	args := make([]any, 0, 4+len(attrs))
	args = append(args, "event", event, "user_id", userID)
	args = append(args, attrs...)
	s.logger.Info("usermgmt: "+event, args...)
}

func (s *Service) emit(userID UserID, evt any) {
	if s.eventHandler == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.Warn("usermgmt: event handler panicked", "user_id", userID, "recover", r)
		}
	}()
	s.eventHandler(userID, evt)
}

// LoginRequest is the input for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks that email and password are non-empty.
// It trims leading/trailing whitespace from Email in-place.
func (r *LoginRequest) Validate() error {
	var errs []string
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	if r.Email == "" {
		errs = append(errs, "email is required")
	}
	if r.Password == "" {
		errs = append(errs, "password is required")
	} else if len(r.Password) > maxPasswordLength {
		errs = append(errs, errMsgPasswordTooLong)
	}
	return formatValidationErrors(errs)
}

// LoginResponse contains the authenticated User and active Session.
type LoginResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

// Login validates credentials, enforces account lockout, and opens a session.
// Returns ErrInvalidCredentials, ErrAccountLocked, or ErrValidation on failure.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if s.lockout != nil && s.lockout.IsLocked(req.Email) {
		s.logger.Warn("usermgmt: login rejected — account locked", "email", req.Email)
		return nil, ErrAccountLocked
	}
	user, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, classifyLoginError(err)
	}

	if !user.CheckPassword(req.Password) {
		s.logger.Warn("usermgmt: login failed", "email", req.Email, "reason", "invalid_password")
		if s.lockout != nil && s.lockout.RecordFailure(req.Email) {
			s.logger.Warn("usermgmt: account locked", "email", req.Email)
		}
		return nil, ErrInvalidCredentials
	}

	if s.lockout != nil {
		s.lockout.Reset(req.Email)
	}

	session, err := s.sessions.Create(ctx, user.ID, s.sessionTTL)
	if err != nil {
		return nil, event.NewTransient("internal", "create session").WithCause(err)
	}

	s.emit(user.ID, UserLoggedInEvent{
		Email:      user.Email,
		OccurredAt: time.Now().UTC(),
	})
	return &LoginResponse{User: user, Session: session}, nil
}

// Logout deletes the session associated with the given token.
func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.sessions.Delete(ctx, token); err != nil {
		return event.NewTransient("internal", "logout").WithCause(err)
	}
	return nil
}

// Authenticate validates a session token and returns the associated User.
// Expired or invalid tokens result in ErrSessionExpired or ErrUnauthorized.
func (s *Service) Authenticate(ctx context.Context, token string) (*User, error) {
	session, err := s.sessions.Find(ctx, token)
	if err != nil {
		return nil, ErrUnauthorized
	}

	// Proactively clean up expired sessions from the store.
	if session.IsExpired() {
		_ = s.sessions.Delete(ctx, token)
		return nil, ErrSessionExpired
	}

	if !session.Valid(token) {
		return nil, ErrUnauthorized
	}

	user, err := s.users.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// Authorize checks whether the subject can perform the action on the object
// within the given domain. Returns ErrForbidden on denial.
func (s *Service) Authorize(_ context.Context, sub, dom, obj string, act Action) error {
	return s.authz.Authorize(sub, dom, obj, act)
}

// GetUser retrieves a user by ID.
func (s *Service) GetUser(ctx context.Context, id UserID) (*User, error) {
	u, err := s.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("get user %q: %w", id, err)
		}
		return nil, event.NewTransient("internal", "get user").WithCause(err)
	}
	return u, nil
}

// UpdateRoles replaces the user's roles in both the Casbin policy and the user store.
func (s *Service) UpdateRoles(
	ctx context.Context,
	userID UserID,
	roles []Role,
	domain string,
) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return fmt.Errorf("update roles: find user %q in domain %q: %w", userID, domain, err)
		}
		return event.NewTransient("internal", fmt.Sprintf("find user %q", userID)).WithCause(err)
	}

	currentRoles, err := s.authz.RolesForUser(userID, domain)
	if err != nil {
		return event.NewTransient("internal", fmt.Sprintf("get roles for user %q in domain %q", userID, domain)).
			WithCause(err)
	}

	var remove []GroupPolicy
	for _, role := range currentRoles {
		remove = append(remove, GroupPolicy{
			Subject: userID.Get(), Role: role, Domain: domain,
		})
	}

	var add []GroupPolicy
	for _, role := range roles {
		add = append(add, GroupPolicy{
			Subject: userID.Get(), Role: role, Domain: domain,
		})
	}

	user.SetRoles(roles)

	if err := s.authz.Apply(PolicyUpdate{
		RemoveGroups: remove,
		AddGroups:    add,
	}); err != nil {
		return event.NewTransient("internal", fmt.Sprintf("apply role update for user %q", userID)).
			WithCause(err)
	}

	if err := s.saveUser(ctx, user, "after role update", userID); err != nil {
		return err
	}

	s.logAuth("roles_updated", userID, "roles", formatRoles(roles), "domain", domain)

	s.emit(userID, RolesUpdatedEvent{
		Roles:      append([]Role(nil), roles...),
		Domain:     domain,
		OccurredAt: time.Now().UTC(),
	})
	return nil
}

func formatRoles(roles []Role) string {
	strs := make([]string, len(roles))
	for i, r := range roles {
		strs[i] = string(r)
	}
	return strings.Join(strs, ",")
}

func (s *Service) saveUser(ctx context.Context, user *User, context string, userID UserID) error {
	if err := s.users.Save(ctx, user); err != nil {
		return event.NewTransient("internal", fmt.Sprintf("save user %q %s", userID, context)).
			WithCause(err)
	}
	return nil
}

func classifyLoginError(err error) error {
	if errors.Is(err, ErrUserNotFound) {
		return ErrInvalidCredentials
	}
	return event.NewTransient("internal", "find user by email").WithCause(err)
}

// ChangePassword verifies the old password, validates the new password length,
// and updates the stored hash.
func (s *Service) ChangePassword(
	ctx context.Context,
	userID UserID,
	oldPassword, newPassword string,
) error {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return event.NewTransient("internal", fmt.Sprintf("find user %q", userID)).WithCause(err)
	}

	matched, err := user.ChangePassword(oldPassword, newPassword, s.bcryptCost)
	if err != nil {
		return err
	}
	if !matched {
		return ErrInvalidCredentials
	}

	if err := s.saveUser(ctx, user, "after password change", userID); err != nil {
		return err
	}
	s.emit(userID, PasswordChangedEvent{OccurredAt: time.Now().UTC()})
	return nil
}
