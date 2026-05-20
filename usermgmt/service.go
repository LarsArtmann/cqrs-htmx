package usermgmt

import (
	"context"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

const (
	defaultSessionTTL = 24 * time.Hour
	minPasswordLength = 8
)

type Service struct {
	authz      *Authz
	users      UserStore
	sessions   SessionStore
	sessionTTL time.Duration
	bcryptCost int
	logger     *slog.Logger
	lockout    *AccountLockout
}

type ServiceConfig struct {
	Authz        *Authz
	UserStore    UserStore
	SessionStore SessionStore
	SessionTTL   time.Duration
	BcryptCost   int
	Logger       *slog.Logger
	Lockout      *AccountLockout
}

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
			return nil, errors.Wrapf(err, "create authz")
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
		authz:      cfg.Authz,
		users:      cfg.UserStore,
		sessions:   cfg.SessionStore,
		sessionTTL: cfg.SessionTTL,
		bcryptCost: cost,
		logger:     logger,
		lockout:    cfg.Lockout,
	}, nil
}

func (s *Service) Authz() *Authz { return s.authz }

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

	return errors.WithMessagef(ErrValidation, "%s", strings.Join(errs, "; "))
}

func (r RegisterRequest) Validate() error {
	var errs []string
	r.Email = strings.TrimSpace(r.Email)
	r.DisplayName = strings.TrimSpace(r.DisplayName)
	if r.ID.IsZero() {
		errs = append(errs, "id is required")
	}
	if _, err := mail.ParseAddress(r.Email); err != nil {
		errs = append(errs, "invalid email")
	}
	if len(r.Password) < minPasswordLength {
		errs = append(errs, "password must be at least 8 characters")
	}
	if len(r.DisplayName) > 100 {
		errs = append(errs, "display name must be under 100 characters")
	}
	return formatValidationErrors(errs)
}

type RegisterResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	user := NewUser(req.ID, req.Email, req.DisplayName)
	if err := user.SetPasswordWithCost(req.Password, s.bcryptCost); err != nil {
		return nil, errors.Wrapf(err, "set password")
	}

	user.AddRole(RoleUser)

	if err := s.users.Create(user); err != nil {
		return nil, err
	}

	if err := s.authz.AddGroupPolicy(GroupPolicy{
		Subject: user.ID.String(), Role: RoleUser, Domain: user.ID.String(),
	}); err != nil {
		return nil, errors.Wrapf(err, "assign role")
	}

	session, err := s.sessions.Create(user.ID, s.sessionTTL)
	if err != nil {
		return nil, errors.Wrapf(err, "create session")
	}

	return &RegisterResponse{User: user, Session: session}, nil
}

func (s *Service) logAuth(event string, userID UserID, attrs ...any) {
	args := []any{"event", event, "user_id", userID}
	args = append(args, attrs...)
	s.logger.Info("usermgmt: "+event, args...)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r LoginRequest) Validate() error {
	var errs []string
	r.Email = strings.TrimSpace(r.Email)
	if r.Email == "" {
		errs = append(errs, "email is required")
	}
	if r.Password == "" {
		errs = append(errs, "password is required")
	}
	return formatValidationErrors(errs)
}

type LoginResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if s.lockout != nil && s.lockout.IsLocked(req.Email) {
		s.logger.Warn("usermgmt: login rejected — account locked", "email", req.Email)
		return nil, ErrAccountLocked
	}
	user, err := s.users.FindByEmail(req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
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

	session, err := s.sessions.Create(user.ID, s.sessionTTL)
	if err != nil {
		return nil, errors.Wrapf(err, "create session")
	}

	return &LoginResponse{User: user, Session: session}, nil
}

func (s *Service) Logout(_ context.Context, token string) error {
	return s.sessions.Delete(token)
}

func (s *Service) Authenticate(_ context.Context, token string) (*User, error) {
	session, err := s.sessions.Find(token)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if session.IsExpired() {
		_ = s.sessions.Delete(token)
		return nil, ErrSessionExpired
	}

	if !session.Valid(token) {
		return nil, ErrUnauthorized
	}

	user, err := s.users.FindByID(session.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (s *Service) Authorize(_ context.Context, sub, dom, obj string, act Action) error {
	return s.authz.Authorize(sub, dom, obj, act)
}

func (s *Service) GetUser(_ context.Context, id UserID) (*User, error) {
	return s.users.FindByID(id)
}

func (s *Service) UpdateRoles(
	_ context.Context,
	userID UserID,
	roles []Role,
	domain string,
) error {
	user, err := s.users.FindByID(userID)
	if err != nil {
		return errors.Wrapf(err, "find user %q", userID)
	}

	currentRoles, err := s.authz.RolesForUser(userID, domain)
	if err != nil {
		return errors.Wrapf(err, "get roles for user %q in domain %q", userID, domain)
	}

	var remove []GroupPolicy
	for _, role := range currentRoles {
		remove = append(remove, GroupPolicy{
			Subject: userID.String(), Role: role, Domain: domain,
		})
	}

	var add []GroupPolicy
	for _, role := range roles {
		add = append(add, GroupPolicy{
			Subject: userID.String(), Role: role, Domain: domain,
		})
	}

	if err := s.authz.Apply(PolicyUpdate{
		RemoveGroups: remove,
		AddGroups:    add,
	}); err != nil {
		return errors.Wrapf(err, "apply role update for user %q", userID)
	}

	s.logAuth("roles_updated", userID, "roles", formatRoles(roles), "domain", domain)

	user.Roles = roles
	user.UpdatedAt = time.Now().UTC()
	return s.users.Save(user)
}

func formatRoles(roles []Role) string {
	strs := make([]string, len(roles))
	for i, r := range roles {
		strs[i] = string(r)
	}
	return strings.Join(strs, ",")
}

func (s *Service) ChangePassword(
	_ context.Context,
	userID UserID,
	oldPassword, newPassword string,
) error {
	user, err := s.users.FindByID(userID)
	if err != nil {
		return errors.Wrapf(err, "find user %q", userID)
	}

	if !user.CheckPassword(oldPassword) {
		return ErrInvalidCredentials
	}

	if len(newPassword) < minPasswordLength {
		return errors.WithMessagef(ErrValidation,
			"password must be at least 8 characters for user %q", userID)
	}

	if err := user.SetPasswordWithCost(newPassword, s.bcryptCost); err != nil {
		return errors.Wrapf(err, "set password for user %q", userID)
	}

	return s.users.Save(user)
}
