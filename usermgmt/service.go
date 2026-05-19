package usermgmt

import (
	"fmt"
	"time"

	"github.com/casbin/casbin/v3"
)

const defaultSessionTTL = 24 * time.Hour

type Service struct {
	users      UserStore
	sessions   SessionStore
	enforcer   *casbin.Enforcer
	sessionTTL time.Duration
}

type ServiceConfig struct {
	UserStore    UserStore
	SessionStore SessionStore
	Enforcer     *casbin.Enforcer
	SessionTTL   time.Duration
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
	if cfg.Enforcer == nil {
		e, err := NewEnforcer()
		if err != nil {
			return nil, fmt.Errorf("create enforcer: %w", err)
		}
		cfg.Enforcer = e
	}

	return &Service{
		users:      cfg.UserStore,
		sessions:   cfg.SessionStore,
		enforcer:   cfg.Enforcer,
		sessionTTL: cfg.SessionTTL,
	}, nil
}

func (s *Service) Enforcer() *casbin.Enforcer {
	return s.enforcer
}

type RegisterRequest struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type RegisterResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

func (s *Service) Register(req RegisterRequest) (*RegisterResponse, error) {
	existing, err := s.users.FindByEmail(req.Email)
	if err == nil && existing != nil {
		return nil, ErrEmailExists
	}

	user := NewUser(req.ID, req.Email, req.DisplayName)
	if err := user.SetPassword(req.Password); err != nil {
		return nil, fmt.Errorf("set password: %w", err)
	}

	user.AddRole(RoleUser)

	if err := s.users.Save(user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	if err := AssignRole(s.enforcer, user.ID, RoleUser); err != nil {
		return nil, fmt.Errorf("assign role: %w", err)
	}

	session, err := s.sessions.Create(user.ID, s.sessionTTL)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &RegisterResponse{User: user, Session: session}, nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	user, err := s.users.FindByEmail(req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.CheckPassword(req.Password) {
		return nil, ErrInvalidCredentials
	}

	session, err := s.sessions.Create(user.ID, s.sessionTTL)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &LoginResponse{User: user, Session: session}, nil
}

func (s *Service) Logout(token string) error {
	return s.sessions.Delete(token)
}

func (s *Service) Authenticate(token string) (*User, error) {
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

func (s *Service) Authorize(userID, object, action string) (bool, error) {
	return CheckPermission(s.enforcer, userID, object, action)
}

func (s *Service) GetUser(id string) (*User, error) {
	return s.users.FindByID(id)
}

func (s *Service) UpdateRoles(userID string, roles []string) error {
	user, err := s.users.FindByID(userID)
	if err != nil {
		return err
	}

	currentRoles, err := RolesForUser(s.enforcer, userID)
	if err != nil {
		return fmt.Errorf("get roles: %w", err)
	}

	for _, role := range currentRoles {
		if err := RevokeRole(s.enforcer, userID, role); err != nil {
			return fmt.Errorf("revoke role %s: %w", role, err)
		}
	}

	for _, role := range roles {
		if err := AssignRole(s.enforcer, userID, role); err != nil {
			return fmt.Errorf("assign role %s: %w", role, err)
		}
	}

	user.Roles = roles
	user.UpdatedAt = time.Now().UTC()
	return s.users.Save(user)
}
